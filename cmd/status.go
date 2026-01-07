package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-subclone/internal/git"
	"github.com/yejune/git-subclone/internal/manifest"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all subclones",
	Long: `Display detailed status information:
  - Mother repository ignore/skip configuration
  - All subclones status
  - Verification results

Examples:
  git-subclone status`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Section 1: Mother Repository Config
	fmt.Println("Mother Repository Configuration:")

	if len(m.Ignore) > 0 {
		fmt.Println("  Ignore patterns:")
		for _, pattern := range m.Ignore {
			fmt.Printf("    - %s\n", pattern)
		}
	} else {
		fmt.Println("  Ignore patterns: (none)")
	}

	if len(m.Skip) > 0 {
		fmt.Println("  Skip-worktree files:")
		for _, file := range m.Skip {
			fmt.Printf("    - %s\n", file)
		}
	} else {
		fmt.Println("  Skip-worktree files: (none)")
	}

	// Active skip-worktree
	activeSkip, err := git.ListSkipWorktree(repoRoot)
	if err == nil && len(activeSkip) > 0 {
		fmt.Println("  Active skip-worktree:")
		for _, file := range activeSkip {
			inConfig := false
			for _, s := range m.Skip {
				if s == file {
					inConfig = true
					break
				}
			}
			if inConfig {
				fmt.Printf("    ✓ %s\n", file)
			} else {
				fmt.Printf("    ⚠ %s (not in config)\n", file)
			}
		}
	}

	fmt.Println()

	// Section 2: Subclones Status
	if len(m.Subclones) == 0 {
		fmt.Println("No subclones registered.")
		return nil
	}

	fmt.Printf("Subclones (%d):\n\n", len(m.Subclones))

	var issues []string

	for _, sc := range m.Subclones {
		fullPath := filepath.Join(repoRoot, sc.Path)

		fmt.Printf("  %s\n", sc.Path)
		fmt.Printf("    Remote: %s\n", sc.Repo)

		if !git.IsRepo(fullPath) {
			fmt.Printf("    Status: ○ not cloned\n")
			issues = append(issues, fmt.Sprintf("%s: not cloned", sc.Path))
			fmt.Println()
			continue
		}

		// Current branch
		branch, err := git.GetCurrentBranch(fullPath)
		if err != nil {
			fmt.Printf("    Current branch: unknown\n")
		} else {
			fmt.Printf("    Current branch: %s\n", branch)
		}

		// Check for changes
		hasChanges, err := git.HasChanges(fullPath)
		if err != nil {
			fmt.Printf("    Status: ✗ error checking status\n")
		} else if hasChanges {
			fmt.Printf("    Status: ● has uncommitted changes\n")
		} else {
			fmt.Printf("    Status: ✓ clean\n")
		}

		// Skip-worktree config
		if len(sc.Skip) > 0 {
			fmt.Println("    Skip-worktree:")
			for _, file := range sc.Skip {
				fmt.Printf("      - %s\n", file)
			}
		}

		// Verify .gitignore
		if !hasGitignoreEntry(repoRoot, sc.Path) {
			fmt.Printf("    ⚠ Missing from .gitignore\n")
			issues = append(issues, fmt.Sprintf("%s: missing from .gitignore", sc.Path))
		}

		fmt.Println()
	}

	// Section 3: Verification Summary
	if len(issues) > 0 {
		fmt.Printf("Issues found (%d):\n", len(issues))
		for _, issue := range issues {
			fmt.Printf("  ✗ %s\n", issue)
		}
		fmt.Println("\nRun 'git subclone sync' to fix automatically.")
	} else {
		fmt.Println("✓ All subclones verified successfully")
	}

	return nil
}
