// Package cmd implements the CLI commands for git-workspace
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/backup"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/i18n"
	"github.com/yejune/git-workspace/internal/interactive"
	"github.com/yejune/git-workspace/internal/manifest"
	"github.com/yejune/git-workspace/internal/patch"
)

var pullCmd = &cobra.Command{
	Use:   "pull [path]",
	Short: "Pull latest changes for workspaces",
	Long: `Pull latest changes for registered workspaces.

Examples:
  git workspace pull              # Pull all workspaces with confirmation
  git workspace pull apps/admin   # Pull specific workspace only

For each workspace:
  1. Shows current branch and uncommitted files
  2. Asks for confirmation (Y/n)
  3. Pulls from remote
  4. Shows result (✓ Updated / ✗ Failed)`,
	RunE: runPull,
}

func init() {
	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) error {
	// Get repository root
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load manifest
	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Set language from manifest
	i18n.SetLanguage(m.GetLanguage())

	if len(m.Workspaces) == 0 {
		fmt.Println(i18n.T("no_subs_registered"))
		return nil
	}

	// Filter workspaces if path argument provided
	var workspacesToProcess []manifest.WorkspaceEntry
	if len(args) > 0 {
		targetPath := args[0]
		found := false
		for _, workspace := range m.Workspaces {
			if workspace.Path == targetPath {
				workspacesToProcess = []manifest.WorkspaceEntry{workspace}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(i18n.T("sub_not_found", targetPath))
		}
	} else {
		workspacesToProcess = m.Workspaces
	}

	reader := bufio.NewReader(os.Stdin)

	for _, workspace := range workspacesToProcess {
		fullPath := filepath.Join(repoRoot, workspace.Path)

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

		// Count uncommitted files
		modifiedFiles, _ := git.GetModifiedFiles(fullPath)
		untrackedFiles, _ := git.GetUntrackedFiles(fullPath)
		stagedFiles, _ := git.GetStagedFiles(fullPath)
		totalUncommitted := len(modifiedFiles) + len(untrackedFiles) + len(stagedFiles)

		// Show current status
		fmt.Printf("%s (%s):\n", workspace.Path, branch)
		if totalUncommitted > 0 {
			fmt.Printf("  %s\n", i18n.T("uncommitted_files", totalUncommitted))
		} else {
			fmt.Printf("  %s\n", i18n.T("clean_directory"))
		}

		// Ask for confirmation
		fmt.Print("  " + i18n.T("pull_confirm"))
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("  %s\n", i18n.T("failed_read_input", err))
			fmt.Println()
			continue
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input != "" && input != "y" && input != "yes" {
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
			if err := handleKeepFiles(fullPath, branch, keepFiles); err != nil {
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
func handleKeepFiles(wsPath, branch string, keepFiles []string) error {
	for _, file := range keepFiles {
		// Check if file has remote changes
		hasChanges, err := git.HasRemoteChanges(wsPath, file, branch)
		if err != nil {
			return fmt.Errorf("failed to check remote changes for %s: %w", file, err)
		}

		if !hasChanges {
			continue // No remote changes, skip
		}

		// Create temporary patch directory
		patchDir := filepath.Join(wsPath, ".git", "git-workspace", "patches")
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
				backupDir := filepath.Join(filepath.Dir(wsPath), ".workspaces", "backup")
				if err := backup.CreateFileBackup(filepath.Join(wsPath, file), backupDir, filepath.Dir(wsPath)); err != nil {
					fmt.Printf("  ⚠ Backup failed: %v\n", err)
					continue
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
