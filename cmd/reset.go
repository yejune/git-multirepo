package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-subclone/internal/git"
	"github.com/yejune/git-subclone/internal/manifest"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset ignore patterns and skip-worktree files",
	Long: `Reset both ignore patterns and skip-worktree files.

This will:
  - Remove git-subclone section from .gitignore
  - Remove all skip-worktree flags
  - Reapply from .subclones.yaml

NOTE: This does NOT modify .subclones.yaml

Examples:
  git-subclone reset           # Reset both
  git-subclone reset ignore    # Reset ignore only
  git-subclone reset skip      # Reset skip only`,
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

	fmt.Println("✓ Reset ignore patterns from .subclones.yaml")
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

	// Reapply from manifest
	if len(m.Skip) > 0 {
		if err := git.ApplySkipWorktree(repoRoot, m.Skip); err != nil {
			return fmt.Errorf("failed to apply skip-worktree: %w", err)
		}
	}

	// Reset each subclone
	for _, sc := range m.Subclones {
		fullPath := filepath.Join(repoRoot, sc.Path)
		if !git.IsRepo(fullPath) {
			continue
		}

		activeSkip, err := git.ListSkipWorktree(fullPath)
		if err == nil && len(activeSkip) > 0 {
			if err := git.UnapplySkipWorktree(fullPath, activeSkip); err != nil {
				fmt.Printf("⚠ Warning: failed to remove skip-worktree in %s: %v\n", sc.Path, err)
			}
		}

		if len(sc.Skip) > 0 {
			if err := git.ApplySkipWorktree(fullPath, sc.Skip); err != nil {
				fmt.Printf("⚠ Warning: failed to apply skip-worktree in %s: %v\n", sc.Path, err)
			}
		}
	}

	fmt.Println("✓ Reset skip-worktree from .subclones.yaml")
	return nil
}
