package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/hooks"
	"github.com/yejune/git-workspace/internal/i18n"
	"github.com/yejune/git-workspace/internal/manifest"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Clone missing subs and apply configurations",
	Long: `Sync all subs from .workspaces manifest:
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

	// Load manifest early to get language setting
	m, err := manifest.Load(repoRoot)
	if err == nil {
		i18n.SetLanguage(m.GetLanguage())
	}

	fmt.Println(i18n.T("syncing"))

	// 1. Auto-install hooks
	if !hooks.IsInstalled(repoRoot) {
		fmt.Println(i18n.T("installing_hooks"))
		if err := hooks.Install(repoRoot); err != nil {
			fmt.Printf("  %s\n", i18n.T("hooks_failed", err))
		} else {
			fmt.Printf("  %s\n", i18n.T("hooks_installed"))
		}
	}

	// 2. Load manifest
	if err != nil || len(m.Subclones) == 0 {
		// No manifest or empty - scan directories for existing subs
		fmt.Println(i18n.T("no_gitsubs_found"))
		discovered, scanErr := scanForSubs(repoRoot)
		if scanErr != nil {
			return fmt.Errorf(i18n.T("failed_scan"), scanErr)
		}

		if len(discovered) == 0 {
			fmt.Println(i18n.T("no_subs_found"))
			fmt.Println(i18n.T("to_add_sub"))
			fmt.Println(i18n.T("cmd_git_sub_clone"))
			return nil
		}

		// Create manifest from discovered subs
		m = &manifest.Manifest{
			Subclones: discovered,
		}

		if err := manifest.Save(repoRoot, m); err != nil {
			return fmt.Errorf("failed to save manifest: %w", err)
		}

		fmt.Printf(i18n.T("created_gitsubs", len(discovered)))
		for _, sc := range discovered {
			fmt.Printf("  - %s (%s)\n", sc.Path, sc.Repo)
		}
	}

	// 3. Apply ignore patterns to mother repo
	if len(m.Ignore) > 0 {
		fmt.Println(i18n.T("applying_ignore"))
		if err := git.AddIgnorePatternsToGitignore(repoRoot, m.Ignore); err != nil {
			fmt.Printf("  %s\n", i18n.T("hooks_failed", err))
		} else {
			fmt.Printf("  %s\n", i18n.T("applied_patterns", len(m.Ignore)))
		}
	}

	// 4. Apply skip-worktree to mother repo
	if len(m.Skip) > 0 {
		fmt.Println(i18n.T("applying_skip_mother"))
		if err := git.ApplySkipWorktree(repoRoot, m.Skip); err != nil {
			fmt.Printf("  %s\n", i18n.T("hooks_failed", err))
		} else {
			fmt.Printf("  %s\n", i18n.T("applied_files", len(m.Skip)))
		}
	}

	if len(m.Subclones) == 0 {
		fmt.Println(i18n.T("no_subclones"))
		return nil
	}

	// 5. Process each subclone
	fmt.Println(i18n.T("processing_subclones"))
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
				fmt.Printf("    %s\n", i18n.T("initializing_git"))

				if err := git.InitRepo(fullPath, sc.Repo, sc.Branch, sc.Commit); err != nil {
					fmt.Printf("    %s\n", i18n.T("failed_initialize", err))
					issues++
					continue
				}

				// Add to .gitignore
				if err := git.AddToGitignore(repoRoot, sc.Path); err != nil {
					fmt.Printf("    %s\n", i18n.T("failed_update_gitignore", err))
				}

				fmt.Printf("    %s\n", i18n.T("initialized_git"))
				continue
			}

			// Directory empty or doesn't exist - clone normally
			fmt.Printf("    %s\n", i18n.T("cloning_from", sc.Repo))

			// Create parent directory if needed
			parentDir := filepath.Dir(fullPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				fmt.Printf("    %s\n", i18n.T("failed_create_dir", err))
				issues++
				continue
			}

			// Clone the repository
			if err := git.Clone(sc.Repo, fullPath, sc.Branch); err != nil {
				fmt.Printf("    %s\n", i18n.T("clone_failed", err))
				issues++
				continue
			}

			// Add to .gitignore
			if err := git.AddToGitignore(repoRoot, sc.Path); err != nil {
				fmt.Printf("    %s\n", i18n.T("failed_update_gitignore", err))
			}

			fmt.Printf("    %s\n", i18n.T("cloned_successfully"))
			continue
		}

		// Auto-update commit hash in .workspaces
		commit, err := git.GetCurrentCommit(fullPath)
		if err == nil && commit != sc.Commit {
			// Check if pushed
			hasUnpushed, checkErr := git.HasUnpushedCommits(fullPath)
			if checkErr == nil {
				if hasUnpushed {
					fmt.Printf("    %s\n", i18n.T("has_unpushed", commit[:7]))
					fmt.Printf("      %s\n", i18n.T("push_first", sc.Path))
				} else {
					// Update .workspaces with pushed commit
					oldCommit := "none"
					if sc.Commit != "" {
						oldCommit = sc.Commit[:7]
					}
					m.UpdateCommit(sc.Path, commit)
					fmt.Printf("    %s\n", i18n.T("updated_commit", oldCommit, commit[:7]))
				}
			}
		}

		// Verify and fix .gitignore entry
		if !hasGitignoreEntry(repoRoot, sc.Path) {
			fmt.Printf("    %s\n", i18n.T("adding_to_gitignore"))
			if err := git.AddToGitignore(repoRoot, sc.Path); err != nil {
				fmt.Printf("    %s\n", i18n.T("hooks_failed", err))
				issues++
			} else {
				fmt.Printf("    %s\n", i18n.T("added_to_gitignore"))
			}
		}

		// Apply skip-worktree for this subclone
		if len(sc.Skip) > 0 {
			fmt.Printf("    %s\n", i18n.T("applying_skip_sub", len(sc.Skip)))
			if err := git.ApplySkipWorktree(fullPath, sc.Skip); err != nil {
				fmt.Printf("    %s\n", i18n.T("hooks_failed", err))
				issues++
			} else {
				fmt.Printf("    %s\n", i18n.T("skip_applied"))
			}
		} else {
			fmt.Printf("    %s\n", i18n.T("no_skip_config"))
		}

		// Install/update post-commit hook in sub
		if !hooks.IsSubHookInstalled(fullPath) {
			fmt.Printf("    %s\n", i18n.T("installing_hook"))
			if err := hooks.InstallSubHook(fullPath); err != nil {
				fmt.Printf("    %s\n", i18n.T("hook_failed", err))
			} else {
				fmt.Printf("    %s\n", i18n.T("hook_installed"))
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
		fmt.Println(i18n.T("completed_issues", issues))
	} else {
		fmt.Println(i18n.T("all_success"))
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

// scanForSubs recursively scans directories for git repositories
func scanForSubs(repoRoot string) ([]manifest.Subclone, error) {
	var subs []manifest.Subclone

	// Walk the directory tree
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip parent's .git directory
		if path == filepath.Join(repoRoot, ".git") {
			return filepath.SkipDir
		}

		// Check if this is a .git directory
		if !info.IsDir() || info.Name() != ".git" {
			return nil
		}

		// Get the repository path (parent of .git)
		subPath := filepath.Dir(path)

		// Skip if it's the parent repo itself
		if subPath == repoRoot {
			return filepath.SkipDir
		}

		// Get relative path from parent
		relPath, err := filepath.Rel(repoRoot, subPath)
		if err != nil {
			return nil
		}

		// Extract git info
		repo, err := git.GetRemoteURL(subPath)
		if err != nil {
			fmt.Println(i18n.T("failed_get_remote", relPath, err))
			return filepath.SkipDir
		}

		branch, err := git.GetCurrentBranch(subPath)
		if err != nil {
			branch = ""
		}

		commit, err := git.GetCurrentCommit(subPath)
		if err != nil {
			fmt.Println(i18n.T("failed_get_commit", relPath, err))
			return filepath.SkipDir
		}

		fmt.Printf("  %s\n", i18n.T("found_sub", relPath))

		subs = append(subs, manifest.Subclone{
			Path:   relPath,
			Repo:   repo,
			Branch: branch,
			Commit: commit,
		})

		// Skip descending into this sub's subdirectories
		return filepath.SkipDir
	})

	return subs, err
}
