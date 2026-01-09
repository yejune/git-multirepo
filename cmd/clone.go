// Package cmd implements the CLI commands for git-workspace
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/hooks"
	"github.com/yejune/git-workspace/internal/manifest"
)

var (
	// Clone command flags
	cloneBranch string
	clonePath   string
)

var cloneCmd = &cobra.Command{
	Use:   "clone <repository> [path]",
	Short: "Clone a new workspace repository",
	Long: `Clone a new workspace repository and add it to the parent project.

Each workspace maintains its own .git directory and can push to its own remote,
while the parent project tracks the source files (but not .git).

Examples:
  git workspace clone https://github.com/user/repo.git           # Clone to ./repo
  git workspace clone https://github.com/user/repo.git lib/repo  # Clone to lib/repo
  git workspace clone -b develop https://github.com/user/repo.git`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runClone,
}

func init() {
	cloneCmd.Flags().StringVarP(&cloneBranch, "branch", "b", "", "Branch to clone")
	cloneCmd.Flags().StringVarP(&clonePath, "path", "p", "", "Destination path")
	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) error {
	// No args = show help
	if len(args) == 0 {
		return cmd.Help()
	}

	repo := args[0]

	// Determine path
	var path string
	if len(args) >= 2 {
		path = args[1]
	} else if clonePath != "" {
		path = clonePath
	} else {
		// Extract repo name from URL
		path = extractRepoName(repo)
	}

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
		return fmt.Errorf("workspace already exists at %s", path)
	}

	// Create parent directory if needed
	fullPath := filepath.Join(repoRoot, path)
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Clone the repository
	fmt.Printf("Cloning %s into %s...\n", repo, path)
	if err := git.Clone(repo, fullPath, cloneBranch); err != nil {
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

	// Install post-commit hook in workspace
	if err := hooks.InstallWorkspaceHook(fullPath); err != nil {
		fmt.Printf("⚠ Failed to install hook: %v\n", err)
	}

	fmt.Printf("✓ Added workspace: %s\n", path)
	fmt.Printf("  Repository: %s\n", repo)

	return nil
}

// extractRepoName extracts repository name from URL
// https://github.com/user/repo.git -> repo
// git@github.com:user/repo.git -> repo
func extractRepoName(url string) string {
	// Remove trailing .git
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH format first: git@host:path/to/repo
	// The colon separates host from path, so we need to extract path first
	if strings.Contains(url, ":") && !strings.Contains(url, "://") {
		// Split by colon to get the path part
		parts := strings.Split(url, ":")
		pathPart := parts[len(parts)-1]
		// Now extract the last component from the path
		pathParts := strings.Split(pathPart, "/")
		return pathParts[len(pathParts)-1]
	}

	// Handle HTTPS or local path format: just get last component after /
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
