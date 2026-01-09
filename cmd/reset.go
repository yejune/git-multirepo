package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/manifest"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset ignore patterns and skip-worktree files",
	Long: `Reset both ignore patterns and skip-worktree files.

This will:
  - Remove git-workspace section from .gitignore
  - Remove all skip-worktree flags
  - Reapply from .workspaces

NOTE: This does NOT modify .workspaces

Examples:
  git workspace reset           # Reset both
  git workspace reset ignore    # Reset ignore only
  git workspace reset skip      # Reset skip only`,
	RunE: runReset,
}

var resetIgnoreCmd = &cobra.Command{
	Use:   "ignore",
	Short: "Reset .gitignore patterns only",
	RunE:  runResetIgnore,
}

var resetSkipCmd = &cobra.Command{
	Use:   "skip",
	Short: "Reset skip-worktree files only",
	RunE:  runResetSkip,
}

func init() {
	resetCmd.AddCommand(resetIgnoreCmd)
	resetCmd.AddCommand(resetSkipCmd)
	rootCmd.AddCommand(resetCmd)
}

func runReset(cmd *cobra.Command, args []string) error {
	if err := runResetIgnore(cmd, args); err != nil {
		return err
	}
	return runResetSkip(cmd, args)
}

func runResetIgnore(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Remove existing patterns
	if err := git.RemoveIgnorePatternsFromGitignore(repoRoot); err != nil {
		return fmt.Errorf("failed to remove ignore patterns: %w", err)
	}

	// Reapply from manifest
	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if len(m.Ignore) > 0 {
		if err := git.AddIgnorePatternsToGitignore(repoRoot, m.Ignore); err != nil {
			return fmt.Errorf("failed to apply ignore patterns: %w", err)
		}
	}

	fmt.Println("✓ Reset ignore patterns from .workspaces")
	return nil
}

func runResetSkip(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Reset mother repository
	activeSkip, err := git.ListSkipWorktree(repoRoot)
	if err == nil && len(activeSkip) > 0 {
		if err := git.UnapplySkipWorktree(repoRoot, activeSkip); err != nil {
			fmt.Printf("⚠ Warning: failed to remove skip-worktree: %v\n", err)
		}
	}

	// Reset each workspace
	for _, ws := range m.Workspaces {
		fullPath := filepath.Join(repoRoot, ws.Path)
		if !git.IsRepo(fullPath) {
			continue
		}

		// Handle keep files: restore from origin but keep local modifications
		keepFiles := ws.Keep
		if len(keepFiles) > 0 {
			// 1. Unapply skip-worktree for keep files
			if err := git.UnapplySkipWorktree(fullPath, keepFiles); err != nil {
				fmt.Printf("⚠ Warning: failed to unapply skip-worktree for keep files in %s: %v\n", ws.Path, err)
			}

			// 2. Restore original files from HEAD (git checkout HEAD -- file)
			// Note: This does NOT delete .workspaces/patches/ or backup files
			for _, file := range keepFiles {
				cmd := exec.Command("git", "-C", fullPath, "checkout", "HEAD", "--", file)
				if err := cmd.Run(); err != nil {
					// File might not exist in HEAD, that's okay
					continue
				}
			}

			// 3. Reapply skip-worktree (patches are preserved in .workspaces/patches/)
			if err := git.ApplySkipWorktree(fullPath, keepFiles); err != nil {
				fmt.Printf("⚠ Warning: failed to reapply skip-worktree for keep files in %s: %v\n", ws.Path, err)
			}
		}

		// Pull from current branch (ignore branch setting)
		if err := git.Pull(fullPath); err != nil {
			fmt.Printf("⚠ Warning: failed to pull in %s: %v\n", ws.Path, err)
		}
	}

	fmt.Println("✓ Reset skip-worktree from .workspaces")
	return nil
}
