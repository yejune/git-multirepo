package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/i18n"
	"github.com/yejune/git-workspace/internal/manifest"
)

var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Show detailed status of subs",
	Long: `Display comprehensive status information for each sub:

Examples:
  git sub status              # Show status for all subs
  git sub status apps/admin   # Show status for specific sub

For each sub, shows:
  1. Local Status (modified, untracked, staged files)
  2. Remote Status (commits behind/ahead)
  3. Skip Files (remote changes detection)
  4. How to resolve (step-by-step commands)`,
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

	// Set language from manifest
	i18n.SetLanguage(m.GetLanguage())

	if len(m.Subclones) == 0 {
		fmt.Println(i18n.T("no_subs_registered"))
		return nil
	}

	// Filter subs if path argument provided
	var subsToProcess []manifest.Subclone
	if len(args) > 0 {
		targetPath := args[0]
		found := false
		for _, sub := range m.Subclones {
			if sub.Path == targetPath {
				subsToProcess = []manifest.Subclone{sub}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(i18n.T("sub_not_found", targetPath))
		}
	} else {
		subsToProcess = m.Subclones
	}

	for _, sc := range subsToProcess {
		fullPath := filepath.Join(repoRoot, sc.Path)

		fmt.Printf("%s", sc.Path)

		if !git.IsRepo(fullPath) {
			fmt.Printf(" %s\n", i18n.T("not_cloned"))
			fmt.Println()
			fmt.Printf("  %s\n", i18n.T("how_to_resolve"))
			fmt.Println("    git sub sync")
			fmt.Println()
			continue
		}

		// Get current branch
		branch, err := git.GetCurrentBranch(fullPath)
		if err != nil {
			branch = "unknown"
		}
		fmt.Printf(" (%s):\n", branch)
		fmt.Println()

		// Section 1: Local Status
		fmt.Printf("  %s\n", i18n.T("local_status"))

		modifiedFiles, _ := git.GetModifiedFiles(fullPath)
		untrackedFiles, _ := git.GetUntrackedFiles(fullPath)
		stagedFiles, _ := git.GetStagedFiles(fullPath)

		hasLocalChanges := false

		if len(modifiedFiles) > 0 {
			hasLocalChanges = true
			fmt.Printf("    %s\n", i18n.T("files_modified", len(modifiedFiles)))
			for _, file := range modifiedFiles {
				fmt.Printf("      - %s\n", file)
			}
		}

		if len(untrackedFiles) > 0 {
			hasLocalChanges = true
			fmt.Printf("    %s\n", i18n.T("files_untracked", len(untrackedFiles)))
			for _, file := range untrackedFiles {
				fmt.Printf("      - %s\n", file)
			}
		}

		if len(stagedFiles) > 0 {
			hasLocalChanges = true
			fmt.Printf("    %s\n", i18n.T("files_staged", len(stagedFiles)))
			for _, file := range stagedFiles {
				fmt.Printf("      - %s\n", file)
			}
		}

		if !hasLocalChanges {
			fmt.Printf("    %s\n", i18n.T("clean_working_tree"))
		}
		fmt.Println()

		// Section 2: Remote Status
		fmt.Printf("  %s\n", i18n.T("remote_status"))

		// Fetch from remote (suppress errors)
		_ = git.Fetch(fullPath)

		behindCount, _ := git.GetBehindCount(fullPath, branch)
		aheadCount, _ := git.GetAheadCount(fullPath, branch)

		if behindCount > 0 {
			fmt.Printf("    %s\n", i18n.T("commits_behind", behindCount, branch))
		}

		if aheadCount > 0 {
			fmt.Printf("    %s\n", i18n.T("commits_ahead", aheadCount))
		}

		if behindCount == 0 && aheadCount == 0 {
			fmt.Printf("    %s\n", i18n.T("up_to_date"))
		}

		// Check if remote branch exists
		if behindCount == 0 && aheadCount == 0 {
			// Try to verify remote branch exists
			if err := git.Fetch(fullPath); err != nil {
				fmt.Printf("    %s\n", i18n.T("cannot_fetch"))
			}
		}
		fmt.Println()

		// Section 3: Skip Files
		if len(sc.Skip) > 0 {
			fmt.Printf("  %s\n", i18n.T("skip_files"))

			hasSkipChanges := false
			for _, skipFile := range sc.Skip {
				diff, err := git.GetSkipFileRemoteChanges(fullPath, skipFile)
				if err == nil && strings.TrimSpace(diff) != "" {
					hasSkipChanges = true
					fmt.Printf("    %s\n", i18n.T("skip_file_changed", skipFile))

					// Show a simple summary of changes
					if strings.Contains(diff, "+") && !strings.Contains(diff, "-") {
						fmt.Printf("      %s\n", i18n.T("skip_remote_added"))
					} else if strings.Contains(diff, "-") && !strings.Contains(diff, "+") {
						fmt.Printf("      %s\n", i18n.T("skip_remote_removed"))
					} else if strings.Contains(diff, "+") && strings.Contains(diff, "-") {
						fmt.Printf("      %s\n", i18n.T("skip_remote_modified"))
					}
					fmt.Printf("      %s\n", i18n.T("skip_file_protected"))
				}
			}

			if !hasSkipChanges {
				fmt.Printf("    %s\n", i18n.T("no_remote_changes"))
			}
			fmt.Println()
		}

		// Section 4: How to resolve
		needsResolution := hasLocalChanges || behindCount > 0 || aheadCount > 0

		if needsResolution {
			fmt.Printf("  %s\n", i18n.T("how_to_resolve"))
			fmt.Println()

			if hasLocalChanges {
				fmt.Printf("    %s\n", i18n.T("resolve_commit"))
				fmt.Printf("       cd %s\n", sc.Path)
				if len(stagedFiles) > 0 || len(modifiedFiles) > 0 {
					fmt.Println("       git add .")
					fmt.Println("       git commit -m \"your message\"")
				}
				if len(untrackedFiles) > 0 {
					fmt.Printf("       %s\n", i18n.T("resolve_or_gitignore"))
				}
				fmt.Println()
			}

			if behindCount > 0 {
				fmt.Printf("    %s\n", i18n.T("resolve_pull"))
				fmt.Printf("       git sub pull %s\n", sc.Path)
				fmt.Println()
			}

			if aheadCount > 0 {
				fmt.Printf("    %s\n", i18n.T("resolve_push"))
				fmt.Printf("       cd %s\n", sc.Path)
				fmt.Println("       git push")
				fmt.Println()
			}

			if len(sc.Skip) > 0 {
				hasSkipChanges := false
				for _, skipFile := range sc.Skip {
					diff, err := git.GetSkipFileRemoteChanges(fullPath, skipFile)
					if err == nil && strings.TrimSpace(diff) != "" {
						hasSkipChanges = true
						break
					}
				}

				if hasSkipChanges {
					fmt.Printf("    %s\n", i18n.T("resolve_skip"))
					fmt.Printf("       cd %s\n", sc.Path)
					fmt.Println("       git update-index --no-skip-worktree <file>")
					fmt.Println("       git pull")
					fmt.Printf("       %s\n", i18n.T("resolve_review"))
					fmt.Println("       git update-index --skip-worktree <file>")
					fmt.Println()
				}
			}
		} else {
			fmt.Printf("  %s\n", i18n.T("no_action_needed"))
			fmt.Println()
		}
	}

	return nil
}
