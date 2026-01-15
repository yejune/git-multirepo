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

	// First check if startPath itself is a git repository
	if _, err := os.Stat(filepath.Join(startPath, ".git")); err == nil {
		gitRoots = append(gitRoots, startPath)
	}

	err := filepath.Walk(startPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip the root .git directory (already added above)
		if path == filepath.Join(startPath, ".git") {
			return filepath.SkipDir
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
	rootCount := 0
	workspaceCount := 0

	for _, repoRoot := range gitRoots {
		// Determine if this is a root repository or workspace
		isRoot := isRootRepository(repoRoot)

		var isInstalled bool
		var installErr error
		var hookType string

		if isRoot {
			// Root repository: install post-checkout hook
			isInstalled = hooks.IsInstalled(repoRoot)
			hookType = "post-checkout"
			if !isInstalled {
				installErr = hooks.Install(repoRoot)
				if installErr == nil {
					rootCount++
				}
			}
		} else {
			// Workspace repository: install post-commit hook
			isInstalled = hooks.IsWorkspaceHookInstalled(repoRoot)
			hookType = "post-commit"
			if !isInstalled {
				installErr = hooks.InstallWorkspaceHook(repoRoot)
				if installErr == nil {
					workspaceCount++
				}
			}
		}

		if isInstalled {
			skipped++
			continue
		}

		if installErr != nil {
			fmt.Printf("  ✗ %s (%s): %v\n", repoRoot, hookType, installErr)
			continue
		}

		installed++
		fmt.Printf("  ✓ %s (%s)\n", repoRoot, hookType)
	}

	fmt.Printf("\n")
	if rootCount > 0 && workspaceCount > 0 {
		fmt.Printf("Installed: %d (%d root, %d workspaces), Skipped: %d (already installed)\n",
			installed, rootCount, workspaceCount, skipped)
	} else if rootCount > 0 {
		fmt.Printf("Installed: %d (root only), Skipped: %d (already installed)\n", installed, skipped)
	} else {
		fmt.Printf("Installed: %d (workspaces only), Skipped: %d (already installed)\n", installed, skipped)
	}

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
	rootCount := 0
	workspaceCount := 0

	for _, repoRoot := range gitRoots {
		// Determine if this is a root repository or workspace
		isRoot := isRootRepository(repoRoot)

		var isInstalled bool
		var uninstallErr error
		var hookType string

		if isRoot {
			// Root repository: check and remove post-checkout hook
			isInstalled = hooks.IsInstalled(repoRoot)
			hookType = "post-checkout"
			if isInstalled {
				uninstallErr = hooks.Uninstall(repoRoot)
				rootCount++
			}
		} else {
			// Workspace repository: check and remove post-commit hook
			isInstalled = hooks.IsWorkspaceHookInstalled(repoRoot)
			hookType = "post-commit"
			if isInstalled {
				uninstallErr = hooks.UninstallWorkspaceHook(repoRoot)
				workspaceCount++
			}
		}

		if !isInstalled {
			skipped++
			continue
		}

		if uninstallErr != nil {
			fmt.Printf("  ✗ %s (%s): %v\n", repoRoot, hookType, uninstallErr)
			continue
		}

		removed++
		fmt.Printf("  ✓ %s (%s)\n", repoRoot, hookType)
	}

	fmt.Printf("\nRemoved: %d (%d root, %d workspaces), Skipped: %d (not installed)\n",
		removed, rootCount, workspaceCount, skipped)
	return nil
}

// isRootRepository checks if the given repository is a multirepo root
// by checking if .git.multirepos exists in its directory
func isRootRepository(repoRoot string) bool {
	manifestPath := filepath.Join(repoRoot, ".git.multirepos")
	_, err := os.Stat(manifestPath)
	return err == nil
}
