package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-sub/internal/git"
	"github.com/yejune/git-sub/internal/manifest"
)

var pushAll bool

var pushCmd = &cobra.Command{
	Use:   "push [path]",
	Short: "Push changes in subclones to their remotes",
	Long: `Push changes in a specific subclone or all modified subclones.

Each subclone pushes to its own remote repository, not the parent.

Examples:
  git-subclone push packages/lib    # Push specific subclone
  git-subclone push --all           # Push all modified subclones`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPush,
}

func init() {
	pushCmd.Flags().BoolVarP(&pushAll, "all", "a", false, "Push all modified subclones")
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if len(m.Subclones) == 0 {
		return fmt.Errorf("no subclones registered")
	}

	// Push specific subclone
	if len(args) == 1 {
		path := args[0]
		sc := m.Find(path)
		if sc == nil {
			return fmt.Errorf("subclone not found: %s", path)
		}

		fullPath := filepath.Join(repoRoot, path)
		return pushSubclone(fullPath, path)
	}

	// Push all or require --all flag
	if !pushAll {
		return fmt.Errorf("specify a subclone path or use --all to push all modified subclones")
	}

	// Push all modified subclones
	pushedCount := 0
	for _, sc := range m.Subclones {
		fullPath := filepath.Join(repoRoot, sc.Path)

		if !git.IsRepo(fullPath) {
			continue
		}

		hasChanges, err := git.HasChanges(fullPath)
		if err != nil {
			fmt.Printf("⚠ %s: failed to check status: %v\n", sc.Path, err)
			continue
		}

		if !hasChanges {
			// Check if there are commits to push
			// For simplicity, we'll try to push and see if it succeeds
		}

		if err := pushSubclone(fullPath, sc.Path); err != nil {
			fmt.Printf("✗ %s: %v\n", sc.Path, err)
		} else {
			pushedCount++
		}
	}

	if pushedCount == 0 {
		fmt.Println("No subclones needed pushing.")
	}

	return nil
}

func pushSubclone(fullPath, displayPath string) error {
	if !git.IsRepo(fullPath) {
		return fmt.Errorf("not cloned")
	}

	fmt.Printf("↑ Pushing %s...\n", displayPath)
	if err := git.Push(fullPath); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	fmt.Printf("  ✓ Pushed successfully\n")
	return nil
}
