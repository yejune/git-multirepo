package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/manifest"
)

var removeForce bool
var removeKeepFiles bool

var removeCmd = &cobra.Command{
	Use:     "remove <path>",
	Aliases: []string{"rm"},
	Short:   "Remove a workspace",
	Long: `Remove a workspace from the manifest and optionally delete its files.

By default, prompts before deleting files. Use --force to skip confirmation.
Use --keep-files to only remove from manifest without deleting files.

Examples:
  git workspace remove packages/lib
  git workspace rm packages/lib --force
  git workspace rm packages/lib --keep-files`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Skip confirmation")
	removeCmd.Flags().BoolVar(&removeKeepFiles, "keep-files", false, "Keep files, only remove from manifest")
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	path := args[0]

	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if !m.Exists(path) {
		return fmt.Errorf("workspace not found: %s", path)
	}

	fullPath := filepath.Join(repoRoot, path)

	// Check for uncommitted changes
	if git.IsRepo(fullPath) {
		hasChanges, _ := git.HasChanges(fullPath)
		if hasChanges && !removeForce {
			return fmt.Errorf("workspace has uncommitted changes. Use --force to remove anyway")
		}
	}

	// NEW: Modified files warning
	if !removeKeepFiles && git.IsRepo(fullPath) {
		modified, _ := git.GetModifiedFiles(fullPath)
		if len(modified) > 0 {
			fmt.Printf("⚠️  WARNING: %d modified files will be deleted:\n", len(modified))
			for i, f := range modified {
				if i < 5 {
					fmt.Printf("    - %s\n", f)
				}
			}
			if len(modified) > 5 {
				fmt.Printf("    ... and %d more\n", len(modified)-5)
			}
			fmt.Println()
		}
	}

	// NEW: Backup option suggestion
	if !removeKeepFiles && !removeForce {
		fmt.Printf("💡 Tip: Use '--keep-files' to keep files\n\n")
	}

	// Confirm deletion
	if !removeKeepFiles && !removeForce {
		fmt.Printf("Remove workspace '%s' and delete its files? [y/N] ", path)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Remove from manifest
	// Note: m.Remove always succeeds if m.Exists returned true
	m.Remove(path)

	if err := manifest.Save(repoRoot, m); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	// Remove from .gitignore
	if err := git.RemoveFromGitignore(repoRoot, path); err != nil {
		fmt.Printf("⚠ Failed to update .gitignore: %v\n", err)
	}

	// Delete files
	if !removeKeepFiles {
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("failed to delete files: %w", err)
		}
		fmt.Printf("✓ Removed workspace: %s (files deleted)\n", path)
	} else {
		fmt.Printf("✓ Removed workspace: %s (files kept)\n", path)
	}

	return nil
}
