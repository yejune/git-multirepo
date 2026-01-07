package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-sub/internal/git"
	"github.com/yejune/git-sub/internal/manifest"
)

var removeForce bool
var removeKeepFiles bool

var removeCmd = &cobra.Command{
	Use:     "remove <path>",
	Aliases: []string{"rm"},
	Short:   "Remove a subclone",
	Long: `Remove a subclone from the manifest and optionally delete its files.

By default, prompts before deleting files. Use --force to skip confirmation.
Use --keep-files to only remove from manifest without deleting files.

Examples:
  git-subclone remove packages/lib
  git-subclone rm packages/lib --force
  git-subclone rm packages/lib --keep-files`,
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
		return fmt.Errorf("subclone not found: %s", path)
	}

	fullPath := filepath.Join(repoRoot, path)

	// Check for uncommitted changes
	if git.IsRepo(fullPath) {
		hasChanges, _ := git.HasChanges(fullPath)
		if hasChanges && !removeForce {
			return fmt.Errorf("subclone has uncommitted changes. Use --force to remove anyway")
		}
	}

	// Confirm deletion
	if !removeKeepFiles && !removeForce {
		fmt.Printf("Remove subclone '%s' and delete its files? [y/N] ", path)
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
		fmt.Printf("✓ Removed subclone: %s (files deleted)\n", path)
	} else {
		fmt.Printf("✓ Removed subclone: %s (files kept)\n", path)
	}

	return nil
}
