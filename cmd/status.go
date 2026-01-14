package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yejune/git-multirepo/internal/common"
	"github.com/yejune/git-multirepo/internal/git"
	"github.com/yejune/git-multirepo/internal/i18n"
)

var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Show detailed status of repositories",
	Long: `Display comprehensive status information for each repository:

Examples:
  git multirepo status              # Show status for all repositories
  git multirepo status apps/admin   # Show status for specific repository

For each repository, shows:
  1. Local Status (modified, untracked, staged files)
  2. Remote Status (commits behind/ahead)
  3. How to resolve (step-by-step commands)`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Define color printers
	// Use Fprintf to always print to the correct stdout
	var (
		printCyan   = func(format string, a ...interface{}) { color.New(color.FgCyan, color.Bold).Fprintf(os.Stdout, format, a...) }
		printBlue   = func(format string, a ...interface{}) { color.New(color.FgBlue, color.Bold).Fprintf(os.Stdout, format, a...) }
		printGreen  = func(format string, a ...interface{}) { color.New(color.FgGreen).Fprintf(os.Stdout, format, a...) }
		printYellow = func(format string, a ...interface{}) { color.New(color.FgYellow).Fprintf(os.Stdout, format, a...) }
		printRed    = func(format string, a ...interface{}) { color.New(color.FgRed, color.Bold).Fprintf(os.Stdout, format, a...) }
		printGray   = func(format string, a ...interface{}) { color.New(color.Faint).Fprintf(os.Stdout, format, a...) }
	)

	ctx, err := common.LoadWorkspaceContext()
	if err != nil {
		return err
	}

	if len(ctx.Manifest.Workspaces) == 0 {
		fmt.Println(i18n.T("no_subs_registered"))
		return nil
	}

	// Filter workspaces if path argument provided
	workspacesToProcess, err := ctx.FilterWorkspaces(args)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("sub_not_found", args[0]))
	}

	for idx, ws := range workspacesToProcess {
		if idx > 0 {
			// Add separator between workspaces
			printGray("%s\n", strings.Repeat("─", 80))
			fmt.Println()
		}

		fullPath := filepath.Join(ctx.RepoRoot, ws.Path)

		// Workspace header
		printCyan("%s", ws.Path)

		if !git.IsRepo(fullPath) {
			printRed(" %s\n", i18n.T("not_cloned"))
			fmt.Println()
			printBlue("  %s\n", i18n.T("how_to_resolve"))
			printGray("    git multirepo sync\n")
			fmt.Println()
			continue
		}

		// Get current branch
		branch, err := git.GetCurrentBranch(fullPath)
		if err != nil {
			branch = "unknown"
		}
		printGray(" (%s)\n", branch)
		fmt.Println()

		// Section 1: Local Status
		printBlue("  %s\n", i18n.T("local_status"))

		// Get workspace status using unified pattern
		status, err := git.GetWorkspaceStatus(fullPath, ws.Keep)
		hasLocalChanges := false
		if err != nil {
			printRed("    Failed to get status: %v\n", err)
		} else {
			if len(status.ModifiedFiles) > 0 {
				hasLocalChanges = true
				printYellow("    %s\n", i18n.T("files_modified", len(status.ModifiedFiles)))
				for _, file := range status.ModifiedFiles {
					printGray("      - %s\n", file)
				}
			}

			if len(status.UntrackedFiles) > 0 {
				hasLocalChanges = true
				printYellow("    %s\n", i18n.T("files_untracked", len(status.UntrackedFiles)))
				for _, file := range status.UntrackedFiles {
					printGray("      - %s\n", file)
				}
			}

			if len(status.StagedFiles) > 0 {
				hasLocalChanges = true
				printYellow("    %s\n", i18n.T("files_staged", len(status.StagedFiles)))
				for _, file := range status.StagedFiles {
					printGray("      - %s\n", file)
				}
			}

			if !hasLocalChanges {
				printGreen("    %s\n", i18n.T("clean_working_tree"))
			}
		}
		fmt.Println()

		// Section 2: Remote Status
		printBlue("  %s\n", i18n.T("remote_status"))

		// Fetch from remote (suppress errors)
		_ = git.Fetch(fullPath)

		behindCount, _ := git.GetBehindCount(fullPath, branch)
		aheadCount, _ := git.GetAheadCount(fullPath, branch)

		if behindCount > 0 {
			printYellow("    %s\n", i18n.T("commits_behind", behindCount, branch))
		}

		if aheadCount > 0 {
			printYellow("    %s\n", i18n.T("commits_ahead", aheadCount))
		}

		if behindCount == 0 && aheadCount == 0 {
			printGreen("    %s\n", i18n.T("up_to_date"))
		}

		// Check if remote branch exists
		if behindCount == 0 && aheadCount == 0 {
			// Try to verify remote branch exists
			if err := git.Fetch(fullPath); err != nil {
				printRed("    %s\n", i18n.T("cannot_fetch"))
			}
		}
		fmt.Println()

		// Section 3: How to resolve
		needsResolution := hasLocalChanges || behindCount > 0 || aheadCount > 0

		if needsResolution {
			printBlue("  %s\n", i18n.T("how_to_resolve"))
			fmt.Println()

			if hasLocalChanges {
				printYellow("    %s\n", i18n.T("resolve_commit"))
				printGray("       cd %s\n", ws.Path)
				if len(status.StagedFiles) > 0 || len(status.ModifiedFiles) > 0 {
					printGray("       git add .\n")
					printGray("       git commit -m \"your message\"\n")
				}
				if len(status.UntrackedFiles) > 0 {
					printGray("       %s\n", i18n.T("resolve_or_gitignore"))
				}
				fmt.Println()
			}

			if behindCount > 0 {
				printYellow("    %s\n", i18n.T("resolve_pull"))
				printGray("       git multirepo pull %s\n", ws.Path)
				fmt.Println()
			}

			if aheadCount > 0 {
				printYellow("    %s\n", i18n.T("resolve_push"))
				printGray("       cd %s\n", ws.Path)
				printGray("       git push\n")
				fmt.Println()
			}
		} else {
			printGreen("  %s\n", i18n.T("no_action_needed"))
			fmt.Println()
		}
	}

	return nil
}
