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
	Short: "Clone missing subs and apply configurations",
	Long: `Sync all subs from .gitsubs manifest:
  - Clone missing subs automatically
  - Install git hooks if not present
  - Apply ignore patterns to .gitignore
  - Apply skip-worktree to specified files
  - Verify .gitignore entries for subs

Examples:
  git sub sync`,
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
			// Check if directory has files (parent is tracking source)
			entries, err := os.ReadDir(fullPath)
			if err == nil && len(entries) > 0 {
				// Directory exists with files - init git in place
				fmt.Printf("    → Initializing .git (source files already present)\n")

				if err := git.InitRepo(fullPath, sc.Repo, sc.Branch, sc.Commit); err != nil {
					fmt.Printf("    ✗ Failed to initialize: %v\n", err)
					issues++
					continue
				}

				// Add to .gitignore
				if err := git.AddToGitignore(repoRoot, sc.Path); err != nil {
					fmt.Printf("    ⚠ Failed to update .gitignore: %v\n", err)
				}

				fmt.Printf("    ✓ Initialized .git directory\n")
				continue
			}

			// Directory empty or doesn't exist - clone normally
			fmt.Printf("    → Cloning from %s\n", sc.Repo)

			// Create parent directory if needed
			parentDir := filepath.Dir(fullPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				fmt.Printf("    ✗ Failed to create directory: %v\n", err)
				issues++
				continue
			}

			// Clone the repository
			if err := git.Clone(sc.Repo, fullPath, sc.Branch); err != nil {
				fmt.Printf("    ✗ Clone failed: %v\n", err)
				issues++
				continue
			}

			// Add to .gitignore
			if err := git.AddToGitignore(repoRoot, sc.Path); err != nil {
				fmt.Printf("    ⚠ Failed to update .gitignore: %v\n", err)
			}

			fmt.Printf("    ✓ Cloned successfully\n")
			continue
		}

		// Auto-update commit hash in .gitsubs
		commit, err := git.GetCurrentCommit(fullPath)
		if err == nil && commit != sc.Commit {
			// Check if pushed
			hasUnpushed, checkErr := git.HasUnpushedCommits(fullPath)
			if checkErr == nil {
				if hasUnpushed {
					fmt.Printf("    ⚠ Has unpushed commits (%s)\n", commit[:7])
					fmt.Printf("      Push first: cd %s && git push\n", sc.Path)
				} else {
					// Update .gitsubs with pushed commit
					oldCommit := "none"
					if sc.Commit != "" {
						oldCommit = sc.Commit[:7]
					}
					m.UpdateCommit(sc.Path, commit)
					fmt.Printf("    ✓ Updated commit: %s → %s\n", oldCommit, commit[:7])
				}
			}
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

		// Install/update post-commit hook in sub
		if !hooks.IsSubHookInstalled(fullPath) {
			fmt.Printf("    → Installing post-commit hook\n")
			if err := hooks.InstallSubHook(fullPath); err != nil {
				fmt.Printf("    ⚠ Failed to install hook: %v\n", err)
			} else {
				fmt.Printf("    ✓ Hook installed\n")
			}
		}
	}

	// Save manifest if any commits were updated
	if err := manifest.Save(repoRoot, m); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
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
