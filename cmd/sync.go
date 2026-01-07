package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/git-sub/internal/git"
	"github.com/yejune/git-sub/internal/hooks"
	"github.com/yejune/git-sub/internal/manifest"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Apply ignore and skip-worktree settings",
	Long: `Apply configuration from .gitsubs:
  - Install git hooks if not present
  - Apply ignore patterns to .gitignore
  - Apply skip-worktree to specified files
  - Verify .gitignore entries for subclones

Does NOT pull or clone repositories.

Examples:
  git-subclone sync`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	fmt.Println("Syncing configuration...")

	// 1. Auto-install hooks
	if !hooks.IsInstalled(repoRoot) {
		fmt.Println("→ Installing git hooks")
		if err := hooks.Install(repoRoot); err != nil {
			fmt.Printf("  ✗ Failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Installed")
		}
	}

	// 2. Load manifest
	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// 3. Apply ignore patterns to mother repo
	if len(m.Ignore) > 0 {
		fmt.Println("→ Applying ignore patterns")
		if err := git.AddIgnorePatternsToGitignore(repoRoot, m.Ignore); err != nil {
			fmt.Printf("  ✗ Failed: %v\n", err)
		} else {
			fmt.Printf("  ✓ Applied %d patterns\n", len(m.Ignore))
		}
	}

	// 4. Apply skip-worktree to mother repo
	if len(m.Skip) > 0 {
		fmt.Println("→ Applying skip-worktree to mother repo")
		if err := git.ApplySkipWorktree(repoRoot, m.Skip); err != nil {
			fmt.Printf("  ✗ Failed: %v\n", err)
		} else {
			fmt.Printf("  ✓ Applied to %d files\n", len(m.Skip))
		}
	}

	if len(m.Subclones) == 0 {
		fmt.Println("\nNo subclones registered.")
		return nil
	}

	// 5. Process each subclone
	fmt.Println("\n→ Processing subclones:")
	issues := 0

	for _, sc := range m.Subclones {
		fullPath := filepath.Join(repoRoot, sc.Path)
		fmt.Printf("\n  %s\n", sc.Path)

		// Check if subclone exists
		if !git.IsRepo(fullPath) {
			fmt.Printf("    ✗ Not cloned (run: git subclone %s %s)\n", sc.Repo, sc.Path)
			issues++
			continue
		}

		// Verify and fix .gitignore entry
		if !hasGitignoreEntry(repoRoot, sc.Path) {
			fmt.Printf("    → Adding to .gitignore\n")
			if err := git.AddToGitignore(repoRoot, sc.Path); err != nil {
				fmt.Printf("    ✗ Failed: %v\n", err)
				issues++
			} else {
				fmt.Printf("    ✓ Added\n")
			}
		}

		// Apply skip-worktree for this subclone
		if len(sc.Skip) > 0 {
			fmt.Printf("    → Applying skip-worktree (%d files)\n", len(sc.Skip))
			if err := git.ApplySkipWorktree(fullPath, sc.Skip); err != nil {
				fmt.Printf("    ✗ Failed: %v\n", err)
				issues++
			} else {
				fmt.Printf("    ✓ Applied\n")
			}
		} else {
			fmt.Printf("    ✓ No skip-worktree config\n")
		}
	}

	// Summary
	fmt.Println()
	if issues > 0 {
		fmt.Printf("⚠ Completed with %d issue(s)\n", issues)
	} else {
		fmt.Println("✓ All configurations applied successfully")
	}

	return nil
}

func hasGitignoreEntry(repoRoot, path string) bool {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return false
	}

	expected := path + "/.git/"
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == expected {
			return true
		}
	}
	return false
}
