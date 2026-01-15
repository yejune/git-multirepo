package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-multirepo/internal/hooks"
	"github.com/yejune/git-multirepo/internal/i18n"
)

var installHookCmd = &cobra.Command{
	Use:   "install-hook",
	Short: i18n.T("install_hook_usage"),
	Long: `Install a post-checkout hook that automatically runs 'git multirepo sync' after checkout.

The hook runs from the current directory, respecting the hierarchy:
  - If executed from the repository root → syncs all workspaces
  - If executed from a subdirectory → syncs only workspaces under that directory

Examples:
  git multirepo install-hook`,
	RunE: runInstallHook,
}

var uninstallHookCmd = &cobra.Command{
	Use:   "uninstall-hook",
	Short: i18n.T("uninstall_hook_usage"),
	Long: `Remove the post-checkout hook installed by git-multirepo.

If a previous hook was backed up during installation, it will be restored.

Examples:
  git multirepo uninstall-hook`,
	RunE: runUninstallHook,
}

func init() {
	// Commands registered in root.go init() in workflow order
}

// findAllGitRoots finds all .git directories under the given path
func findAllGitRoots(startPath string) ([]string, error) {
	var gitRoots []string

	err := filepath.Walk(startPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() && info.Name() == ".git" {
			// Found a .git directory
			repoRoot := filepath.Dir(path)
			gitRoots = append(gitRoots, repoRoot)
			return filepath.SkipDir // Don't recurse into .git
		}

		// Skip common directories that shouldn't contain repositories
		if info.IsDir() && shouldSkipDir(info.Name()) {
			return filepath.SkipDir
		}

		return nil
	})

	return gitRoots, err
}

func shouldSkipDir(name string) bool {
	skipDirs := []string{
		"node_modules", "vendor", ".build", "SourcePackages",
		"Carthage", "Pods", "target", "dist", "build",
	}
	for _, skip := range skipDirs {
		if name == skip {
			return true
		}
	}
	return false
}

func runInstallHook(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Find all git repositories under current directory
	gitRoots, err := findAllGitRoots(cwd)
	if err != nil {
		return err
	}

	if len(gitRoots) == 0 {
		return fmt.Errorf("no git repositories found")
	}

	fmt.Printf(i18n.T("installing_hooks")+" (%d repositories)\n", len(gitRoots))

	installed := 0
	skipped := 0

	for _, repoRoot := range gitRoots {
		if hooks.IsInstalled(repoRoot) {
			skipped++
			continue
		}

		if err := hooks.Install(repoRoot); err != nil {
			fmt.Printf("  ✗ %s: %v\n", repoRoot, err)
			continue
		}

		installed++
		fmt.Printf("  ✓ %s\n", repoRoot)
	}

	fmt.Printf("\nInstalled: %d, Skipped: %d (already installed)\n", installed, skipped)
	return nil
}

func runUninstallHook(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Find all git repositories under current directory
	gitRoots, err := findAllGitRoots(cwd)
	if err != nil {
		return err
	}

	if len(gitRoots) == 0 {
		return fmt.Errorf("no git repositories found")
	}

	fmt.Printf(i18n.T("removing_hooks")+" (%d repositories)\n", len(gitRoots))

	removed := 0
	skipped := 0

	for _, repoRoot := range gitRoots {
		if !hooks.IsInstalled(repoRoot) {
			skipped++
			continue
		}

		if err := hooks.Uninstall(repoRoot); err != nil {
			fmt.Printf("  ✗ %s: %v\n", repoRoot, err)
			continue
		}

		removed++
		fmt.Printf("  ✓ %s\n", repoRoot)
	}

	fmt.Printf("\nRemoved: %d, Skipped: %d (not installed)\n", removed, skipped)
	return nil
}
