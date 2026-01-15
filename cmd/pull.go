// Package cmd implements the CLI commands for git-multirepo
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-multirepo/internal/backup"
	"github.com/yejune/git-multirepo/internal/common"
	"github.com/yejune/git-multirepo/internal/git"
	"github.com/yejune/git-multirepo/internal/i18n"
	"github.com/yejune/git-multirepo/internal/interactive"
	"github.com/yejune/git-multirepo/internal/patch"
)

var pullCmd = &cobra.Command{
	Use:   "pull [path]",
	Short: "Pull latest changes for repositories",
	Long: `Pull latest changes for registered repositories.

Examples:
  git multirepo pull              # Pull all repositories with confirmation
  git multirepo pull apps/admin   # Pull specific repository only

For each repository:
  1. Shows current branch and uncommitted files
  2. Asks for confirmation (Y/n)
  3. Pulls from remote
  4. Shows result (✓ Updated / ✗ Failed)`,
	RunE: runPull,
}

func init() {
	// Command registered in root.go init() in workflow order
}

func runPull(cmd *cobra.Command, args []string) error {
	// Load workspace context
	ctx, err := common.LoadWorkspaceContext()
	if err != nil {
		return err
	}

	if len(ctx.Manifest.Workspaces) == 0 {
		fmt.Println(i18n.T("no_subs_registered"))
		return nil
	}

	// Filter workspaces if path argument provided
	workspacesToProcess, err := ctx.FilterWorkspaces(args)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("sub_not_found", args[0]))
	}

	for _, workspace := range workspacesToProcess {
		fullPath := filepath.Join(ctx.RepoRoot, workspace.Path)

		// Check if directory exists and is a git repo
		if !git.IsRepo(fullPath) {
			fmt.Printf("%s:\n", workspace.Path)
			fmt.Printf("  %s\n", i18n.T("not_git_repo"))
			fmt.Println()
			continue
		}

		// Get current branch
		branch, err := git.GetCurrentBranch(fullPath)
		if err != nil {
			fmt.Printf("%s:\n", workspace.Path)
			fmt.Printf("  %s\n", i18n.T("failed_get_branch", err))
			fmt.Println()
			continue
		}

		// Get workspace status using unified pattern
		status, err := git.GetWorkspaceStatus(fullPath, workspace.Keep)
		if err != nil {
			fmt.Printf("%s:\n", workspace.Path)
			fmt.Printf("  Failed to get status: %v\n", err)
			fmt.Println()
			continue
		}

		// Show current status
		fmt.Printf("%s (%s):\n", workspace.Path, branch)
		if status.TotalUncommitted > 0 {
			fmt.Printf("  %s\n", i18n.T("uncommitted_files", status.TotalUncommitted))
		} else {
			fmt.Printf("  %s\n", i18n.T("clean_directory"))
		}

		// Ask for confirmation using unified prompt
		confirmed, err := interactive.ConfirmYesNo("  " + i18n.T("pull_confirm"))
		if err != nil {
			fmt.Printf("  %s\n", i18n.T("failed_read_input", err))
			fmt.Println()
			continue
		}

		if !confirmed {
			fmt.Printf("  %s\n", i18n.T("pull_skipped"))
			fmt.Println()
			continue
		}

		// Fetch remote changes first
		if err := git.Fetch(fullPath); err != nil {
			fmt.Printf("  %s\n", i18n.T("fetch_failed"))
			fmt.Println()
			continue
		}

		// Handle keep files before pulling
		keepFiles := workspace.Keep
		if len(keepFiles) > 0 {
			if err := handleKeepFiles(fullPath, branch, keepFiles, ctx.RepoRoot, workspace.Path); err != nil {
				fmt.Printf("  Keep file handling failed: %v\n", err)
				fmt.Println()
				continue
			}
		}

		// Pull from remote
		if err := git.Pull(fullPath); err != nil {
			fmt.Printf("  %s\n", i18n.T("pull_failed"))
			fmt.Printf("  %s\n", i18n.T("run_status", workspace.Path))
			fmt.Println()
			continue
		}

		// Count changed files
		changedCount := 0
		if output, err := git.CountChangedFiles(fullPath); err == nil {
			changedCount = output
		}

		if changedCount > 0 {
			fmt.Printf("  %s\n", i18n.T("pull_updated", changedCount))
		} else {
			fmt.Printf("  %s\n", i18n.T("pull_already_uptodate"))
		}
		fmt.Println()
	}

	return nil
}

// handleKeepFiles handles keep files with remote changes interactively
func handleKeepFiles(wsPath, branch string, keepFiles []string, repoRoot string, workspacePath string) error {
	// Use transaction pattern for skip-worktree handling
	return git.WithSkipWorktreeTransaction(wsPath, keepFiles, func() error {
		return handleKeepFilesWork(wsPath, branch, keepFiles, repoRoot, workspacePath)
	})
}

// handleKeepFilesWork contains the actual work logic (extracted for transaction)
func handleKeepFilesWork(wsPath, branch string, keepFiles []string, repoRoot string, workspacePath string) error {
	for _, file := range keepFiles {
		// Check if file has remote changes
		hasChanges, err := git.HasRemoteChanges(wsPath, file, branch)
		if err != nil {
			return fmt.Errorf("failed to check remote changes for %s: %w", file, err)
		}

		if !hasChanges {
			continue // No remote changes, skip
		}

		// Create patch directory in .multirepos/patches/{workspace-path}/
		patchDir := filepath.Join(repoRoot, ".multirepos", "patches", workspacePath)
		if err := os.MkdirAll(patchDir, 0755); err != nil {
			return fmt.Errorf("failed to create patch directory: %w", err)
		}

		patchPath := filepath.Join(patchDir, filepath.Base(file)+".patch")

		// Interactive loop for this file
		for {
			choice, err := interactive.ResolveConflict(file, []string{
				"Update origin and reapply patch (recommended)",
				"Update origin only (discard patch)",
				"Skip (keep current state)",
				"Show diff",
			})
			if err != nil {
				return fmt.Errorf("failed to get user choice: %w", err)
			}

			switch choice {
			case 0: // Update origin and reapply patch (recommended)
				// Backup original file
				backupDir := filepath.Join(repoRoot, ".multirepos", "backup")
				if err := backup.CreateFileBackup(filepath.Join(wsPath, file), backupDir, repoRoot); err != nil {
					return fmt.Errorf("backup failed for %s: %w", file, err)
				}

				// Create patch from current local changes
				if err := patch.Create(wsPath, file, patchPath); err != nil {
					fmt.Printf("  ⚠ Failed to create patch: %v\n", err)
					continue
				}

				// Backup patch file
				if err := backup.CreatePatchBackup(patchPath, backupDir); err != nil {
					fmt.Printf("  ⚠ Patch backup failed: %v\n", err)
				}

				// Reset file to remote version
				if err := git.ResetFile(wsPath, file, branch); err != nil {
					fmt.Printf("  ⚠ Failed to reset file: %v\n", err)
					continue
				}

				// Check patch for conflicts before applying
				hasConflicts, err := patch.Check(wsPath, patchPath)
				if err != nil {
					fmt.Printf("  ⚠ Failed to check patch: %v\n", err)
					fmt.Printf("  ℹ Original backed up, patch saved to: %s\n", patchPath)
					continue
				}
				if hasConflicts {
					fmt.Printf("  ⚠ Patch has conflicts\n")
					fmt.Printf("  ℹ Original backed up, patch saved to: %s\n", patchPath)
					continue
				}

				// Apply patch
				if err := patch.Apply(wsPath, patchPath); err != nil {
					fmt.Printf("  ⚠ Failed to apply patch: %v\n", err)
					fmt.Printf("  ℹ Original backed up\n")
				} else {
					fmt.Printf("  ✓ Updated %s and reapplied local changes\n", file)
					// Clean up successful patch
					os.Remove(patchPath)
				}
				return nil

			case 1: // Update origin only (discard patch)
				// Reset file to remote version
				if err := git.ResetFile(wsPath, file, branch); err != nil {
					fmt.Printf("  ⚠ Failed to reset file: %v\n", err)
					continue
				}
				fmt.Printf("  ✓ Updated %s to remote version (local changes discarded)\n", file)
				return nil

			case 2: // Skip (keep current state)
				fmt.Printf("  ⏭ Skipped %s (keeping current state)\n", file)
				return nil

			case 3: // Show diff
				diff, err := git.GetFileDiff(wsPath, file, branch)
				if err != nil {
					fmt.Printf("  ⚠ Failed to get diff: %v\n", err)
					continue
				}
				if err := interactive.ShowDiff(diff); err != nil {
					fmt.Printf("  ⚠ Failed to show diff: %v\n", err)
				}
				// Continue loop to show menu again
				continue

			default:
				return fmt.Errorf("invalid choice: %d", choice)
			}
		}
	}

	return nil
}
