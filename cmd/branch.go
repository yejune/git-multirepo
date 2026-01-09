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
	Use:   "branch [workspace-path]",
	Short: "Show branch information for workspaces",
	Long: `Display current branch for all workspaces or a specific workspace.

Shows:
  - Workspace path
  - Repository URL
  - Current branch

Examples:
  git-workspace branch                 # Show all workspaces
  git-workspace branch packages/lib    # Show specific workspace`,
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

	if len(m.Workspaces) == 0 {
		fmt.Println("No workspaces registered.")
		return nil
	}

	// Show specific workspace
	if len(args) == 1 {
		path := args[0]
		ws := m.Find(path)
		if ws == nil {
			return fmt.Errorf("workspace not found: %s", path)
		}

		return showBranchInfo(repoRoot, ws)
	}

	// Show all workspaces
	fmt.Println("Workspaces:")
	for _, ws := range m.Workspaces {
		if err := showBranchInfo(repoRoot, &ws); err != nil {
			fmt.Printf("  %s: %v\n", ws.Path, err)
		}
	}

	return nil
}

func showBranchInfo(repoRoot string, ws *manifest.WorkspaceEntry) error {
	fullPath := filepath.Join(repoRoot, ws.Path)

	if !git.IsRepo(fullPath) {
		fmt.Printf("  %s: not cloned\n", ws.Path)
		return nil
	}

	// Get current branch
	cmdBranch := exec.Command("git", "-C", fullPath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmdBranch.Output()
	if err != nil {
		fmt.Printf("  %s: failed to get branch\n", ws.Path)
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

	fmt.Printf("  %s\n", ws.Path)
	fmt.Printf("    Repo:   %s\n", ws.Repo)
	fmt.Printf("    Branch: %s", branch)
	if tracking != "" {
		fmt.Printf(" → %s", tracking)
	}
	fmt.Println()

	return nil
}
