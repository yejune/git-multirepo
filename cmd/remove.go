package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-multirepo/internal/common"
	"github.com/yejune/git-multirepo/internal/git"
	"github.com/yejune/git-multirepo/internal/interactive"
	"github.com/yejune/git-multirepo/internal/manifest"
)

var removeForce bool
var removeKeepFiles bool

var removeCmd = &cobra.Command{
	Use:     "remove <path>",
	Aliases: []string{"rm"},
	Short:   "Remove a repository",
	Long: `Remove a repository from the manifest and optionally delete its files.

By default, prompts before deleting files. Use --force to skip confirmation.
Use --keep-files to only remove from manifest without deleting files.

Examples:
  git multirepo remove packages/lib
  git multirepo rm packages/lib --force
  git multirepo rm packages/lib --keep-files`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Skip confirmation")
	removeCmd.Flags().BoolVar(&removeKeepFiles, "keep-files", false, "Keep files, only remove from manifest")
}

func runRemove(cmd *cobra.Command, args []string) error {
	path := args[0]

	ctx, err := common.LoadWorkspaceContext()
	if err != nil {
		return err
	}

	if !ctx.Manifest.Exists(path) {
		return fmt.Errorf("repository not found: %s", path)
	}

	fullPath := filepath.Join(ctx.RepoRoot, path)

	// Check for uncommitted changes
	if git.IsRepo(fullPath) {
		hasChanges, _ := git.HasChanges(fullPath)
		if hasChanges && !removeForce {
			return fmt.Errorf("repository has uncommitted changes. Use --force to remove anyway")
		}
	}

	// NEW: Modified files warning
	if !removeKeepFiles && git.IsRepo(fullPath) {
		// Get workspace from manifest for keep files
		var ws *manifest.WorkspaceEntry
		for i := range ctx.Manifest.Workspaces {
			if ctx.Manifest.Workspaces[i].Path == path {
				ws = &ctx.Manifest.Workspaces[i]
				break
			}
		}

		// Get workspace status using unified pattern
		if ws != nil {
			status, err := git.GetWorkspaceStatus(fullPath, ws.Keep)
			if err == nil && len(status.ModifiedFiles) > 0 {
				fmt.Printf("⚠️  WARNING: %d modified files will be deleted:\n", len(status.ModifiedFiles))
				for i, f := range status.ModifiedFiles {
					if i < 5 {
						fmt.Printf("    - %s\n", f)
					}
				}
				if len(status.ModifiedFiles) > 5 {
					fmt.Printf("    ... and %d more\n", len(status.ModifiedFiles)-5)
				}
				fmt.Println()
			}
		}
	}

	// NEW: Backup option suggestion
	if !removeKeepFiles && !removeForce {
		fmt.Printf("💡 Tip: Use '--keep-files' to keep files\n\n")
	}

	// Confirm deletion using unified prompt
	if !removeKeepFiles && !removeForce {
		confirmed, err := interactive.ConfirmYN(fmt.Sprintf("Remove repository '%s' and delete its files? [y/N] ", path))
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Remove from manifest
	// Note: ctx.Manifest.Remove always succeeds if ctx.Manifest.Exists returned true
	ctx.Manifest.Remove(path)

	if err := ctx.SaveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	// Remove from .gitignore
	if err := git.RemoveFromGitignore(ctx.RepoRoot, path); err != nil {
		fmt.Printf("⚠ Failed to update .gitignore: %v\n", err)
	}

	// Delete files
	if !removeKeepFiles {
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("failed to delete files: %w", err)
		}
		fmt.Printf("✓ Removed repository: %s (files deleted)\n", path)
	} else {
		fmt.Printf("✓ Removed repository: %s (files kept)\n", path)
	}

	return nil
}
