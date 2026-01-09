package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/i18n"
	"github.com/yejune/git-workspace/internal/manifest"
)

var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Show detailed status of workspaces",
	Long: `Display comprehensive status information for each workspace:

Examples:
  git workspace status              # Show status for all workspaces
  git workspace status apps/admin   # Show status for specific workspace

For each workspace, shows:
  1. Local Status (modified, untracked, staged files)
  2. Remote Status (commits behind/ahead)
  3. How to resolve (step-by-step commands)`,
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

	if len(m.Workspaces) == 0 {
		fmt.Println(i18n.T("no_subs_registered"))
		return nil
	}

	// Filter workspaces if path argument provided
	var workspacesToProcess []manifest.WorkspaceEntry
	if len(args) > 0 {
		targetPath := args[0]
		found := false
		for _, workspace := range m.Workspaces {
			if workspace.Path == targetPath {
				workspacesToProcess = []manifest.WorkspaceEntry{workspace}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(i18n.T("sub_not_found", targetPath))
		}
	} else {
		workspacesToProcess = m.Workspaces
	}

	for _, ws := range workspacesToProcess {
		fullPath := filepath.Join(repoRoot, ws.Path)

		fmt.Printf("%s", ws.Path)

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

		// Section 3: How to resolve
		needsResolution := hasLocalChanges || behindCount > 0 || aheadCount > 0

		if needsResolution {
			fmt.Printf("  %s\n", i18n.T("how_to_resolve"))
			fmt.Println()

			if hasLocalChanges {
				fmt.Printf("    %s\n", i18n.T("resolve_commit"))
				fmt.Printf("       cd %s\n", ws.Path)
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
				fmt.Printf("       git workspace pull %s\n", ws.Path)
				fmt.Println()
			}

			if aheadCount > 0 {
				fmt.Printf("    %s\n", i18n.T("resolve_push"))
				fmt.Printf("       cd %s\n", ws.Path)
				fmt.Println("       git push")
				fmt.Println()
			}
		} else {
			fmt.Printf("  %s\n", i18n.T("no_action_needed"))
			fmt.Println()
		}
	}

	return nil
}
