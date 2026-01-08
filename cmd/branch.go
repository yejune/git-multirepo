package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/manifest"
)

var branchCmd = &cobra.Command{
	Use:   "branch [sub-path]",
	Short: "Show branch information for subs",
	Long: `Display current branch for all subs or a specific sub.

Shows:
  - Sub path
  - Repository URL
  - Current branch

Examples:
  git-sub branch                 # Show all subs
  git-sub branch packages/lib    # Show specific sub`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBranch,
}

func init() {
	rootCmd.AddCommand(branchCmd)
}

func runBranch(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if len(m.Subclones) == 0 {
		fmt.Println("No subs registered.")
		return nil
	}

	// Show specific sub
	if len(args) == 1 {
		path := args[0]
		sc := m.Find(path)
		if sc == nil {
			return fmt.Errorf("sub not found: %s", path)
		}

		return showBranchInfo(repoRoot, sc)
	}

	// Show all subs
	fmt.Println("Subs:")
	for _, sc := range m.Subclones {
		if err := showBranchInfo(repoRoot, &sc); err != nil {
			fmt.Printf("  %s: %v\n", sc.Path, err)
		}
	}

	return nil
}

func showBranchInfo(repoRoot string, sc *manifest.Subclone) error {
	fullPath := filepath.Join(repoRoot, sc.Path)

	if !git.IsRepo(fullPath) {
		fmt.Printf("  %s: not cloned\n", sc.Path)
		return nil
	}

	// Get current branch
	cmdBranch := exec.Command("git", "-C", fullPath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmdBranch.Output()
	if err != nil {
		fmt.Printf("  %s: failed to get branch\n", sc.Path)
		return nil
	}

	branch := strings.TrimSpace(string(output))

	// Get remote tracking branch
	cmdTracking := exec.Command("git", "-C", fullPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	trackingOutput, err := cmdTracking.Output()
	tracking := ""
	if err == nil {
		tracking = strings.TrimSpace(string(trackingOutput))
	}

	fmt.Printf("  %s\n", sc.Path)
	fmt.Printf("    Repo:   %s\n", sc.Repo)
	fmt.Printf("    Branch: %s", branch)
	if tracking != "" {
		fmt.Printf(" → %s", tracking)
	}
	fmt.Println()

	return nil
}
