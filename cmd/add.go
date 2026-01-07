package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-subclone/internal/git"
	"github.com/yejune/git-subclone/internal/manifest"
)

var addBranch string

var addCmd = &cobra.Command{
	Use:   "add <repo> <path>",
	Short: "Add a new subclone",
	Long: `Clone a repository as a subclone and register it in .subclones.yaml.

The subclone's source files will be tracked by the parent repo,
but its .git directory will be ignored (added to .gitignore).

Examples:
  git-subclone add https://github.com/user/lib.git packages/lib
  git-subclone add git@github.com:user/lib.git packages/lib -b develop`,
	Args: cobra.ExactArgs(2),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVarP(&addBranch, "branch", "b", "", "Branch to clone")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	repo := args[0]
	path := args[1]

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

	// Check if already exists
	if m.Exists(path) {
		return fmt.Errorf("subclone already exists at %s", path)
	}

	// Create parent directory if needed
	fullPath := filepath.Join(repoRoot, path)
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Clone the repository
	fmt.Printf("Cloning %s into %s...\n", repo, path)
	if err := git.Clone(repo, fullPath, addBranch); err != nil {
		return fmt.Errorf("failed to clone: %w", err)
	}

	// Add to manifest
	m.Add(path, repo)
	if err := manifest.Save(repoRoot, m); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	// Add .git directory to parent's .gitignore
	if err := git.AddToGitignore(repoRoot, path); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
	}

	fmt.Printf("✓ Added subclone: %s\n", path)
	fmt.Printf("  Repository: %s\n", repo)

	return nil
}
