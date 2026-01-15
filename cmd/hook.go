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
	rootCmd.AddCommand(installHookCmd)
	rootCmd.AddCommand(uninstallHookCmd)
}

func runInstallHook(cmd *cobra.Command, args []string) error {
	// Get repository root
	repoRoot := "."
	if cwd, err := os.Getwd(); err == nil {
		repoRoot = cwd
	}

	// Find actual git root
	gitDir := filepath.Join(repoRoot, ".git")
	for {
		if _, err := os.Stat(gitDir); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			return fmt.Errorf("not in a git repository")
		}
		repoRoot = parent
		gitDir = filepath.Join(repoRoot, ".git")
	}

	if hooks.IsInstalled(repoRoot) {
		fmt.Println(i18n.T("hook_already_installed"))
		return nil
	}

	fmt.Println(i18n.T("installing_hooks"))
	hookPath := filepath.Join(repoRoot, ".git", "hooks", "post-checkout")
	backupPath := hookPath + ".bak"

	if err := hooks.Install(repoRoot); err != nil {
		return fmt.Errorf("%s", i18n.T("hooks_failed", err))
	}

	// Check if backup was created
	if _, statErr := os.Stat(backupPath); statErr == nil {
		fmt.Printf("%s\n", i18n.T("hook_backup_warning", backupPath))
	}

	fmt.Printf("%s\n", i18n.T("hook_added"))
	return nil
}

func runUninstallHook(cmd *cobra.Command, args []string) error {
	// Get repository root
	repoRoot := "."
	if cwd, err := os.Getwd(); err == nil {
		repoRoot = cwd
	}

	// Find actual git root
	gitDir := filepath.Join(repoRoot, ".git")
	for {
		if _, err := os.Stat(gitDir); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			return fmt.Errorf("not in a git repository")
		}
		repoRoot = parent
		gitDir = filepath.Join(repoRoot, ".git")
	}

	if !hooks.IsInstalled(repoRoot) {
		fmt.Println(i18n.T("hook_not_installed"))
		return nil
	}

	fmt.Println(i18n.T("removing_hooks"))

	// Uninstall will restore backup if it exists
	backupPath := filepath.Join(repoRoot, ".git", "hooks", "post-checkout.bak")
	hasBackup := false
	if _, statErr := os.Stat(backupPath); statErr == nil {
		hasBackup = true
	}

	if err := hooks.Uninstall(repoRoot); err != nil {
		return fmt.Errorf("%s", i18n.T("hooks_failed", err))
	}

	fmt.Printf("%s\n", i18n.T("hook_removed"))
	if hasBackup {
		fmt.Printf("%s\n", i18n.T("hook_restored", backupPath))
	}

	return nil
}
