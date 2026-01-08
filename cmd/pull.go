// Package cmd implements the CLI commands for git-sub
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/i18n"
	"github.com/yejune/git-workspace/internal/manifest"
)

var pullCmd = &cobra.Command{
	Use:   "pull [path]",
	Short: "Pull latest changes for subs",
	Long: `Pull latest changes for registered subs.

Examples:
  git sub pull              # Pull all subs with confirmation
  git sub pull apps/admin   # Pull specific sub only

For each sub:
  1. Shows current branch and uncommitted files
  2. Asks for confirmation (Y/n)
  3. Pulls from remote
  4. Shows result (✓ Updated / ✗ Failed)`,
	RunE: runPull,
}

func init() {
	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) error {
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

	reader := bufio.NewReader(os.Stdin)

	for _, sub := range subsToProcess {
		fullPath := filepath.Join(repoRoot, sub.Path)

		// Check if directory exists and is a git repo
		if !git.IsRepo(fullPath) {
			fmt.Printf("%s:\n", sub.Path)
			fmt.Printf("  %s\n", i18n.T("not_git_repo"))
			fmt.Println()
			continue
		}

		// Get current branch
		branch, err := git.GetCurrentBranch(fullPath)
		if err != nil {
			fmt.Printf("%s:\n", sub.Path)
			fmt.Printf("  %s\n", i18n.T("failed_get_branch", err))
			fmt.Println()
			continue
		}

		// Count uncommitted files
		modifiedFiles, _ := git.GetModifiedFiles(fullPath)
		untrackedFiles, _ := git.GetUntrackedFiles(fullPath)
		stagedFiles, _ := git.GetStagedFiles(fullPath)
		totalUncommitted := len(modifiedFiles) + len(untrackedFiles) + len(stagedFiles)

		// Show current status
		fmt.Printf("%s (%s):\n", sub.Path, branch)
		if totalUncommitted > 0 {
			fmt.Printf("  %s\n", i18n.T("uncommitted_files", totalUncommitted))
		} else {
			fmt.Printf("  %s\n", i18n.T("clean_directory"))
		}

		// Ask for confirmation
		fmt.Print("  " + i18n.T("pull_confirm"))
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("  %s\n", i18n.T("failed_read_input", err))
			fmt.Println()
			continue
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input != "" && input != "y" && input != "yes" {
			fmt.Printf("  %s\n", i18n.T("pull_skipped"))
			fmt.Println()
			continue
		}

		// Pull from remote
		if err := git.Pull(fullPath); err != nil {
			fmt.Printf("  %s\n", i18n.T("pull_failed"))
			fmt.Printf("  %s\n", i18n.T("run_status", sub.Path))
			fmt.Println()
			continue
		}

		// Count changed files
		changedCount := 0
		if output, err := git.CountChangedFiles(fullPath); err == nil {
			changedCount = output
		}

		if changedCount > 0 {
			fmt.Printf("  %s\n", i18n.T("pull_updated", changedCount))
		} else {
			fmt.Printf("  %s\n", i18n.T("pull_already_uptodate"))
		}
		fmt.Println()
	}

	return nil
}
