package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yejune/git-multirepo/internal/hooks"
)

func TestHookCommandsMultipleWorkspaces(t *testing.T) {
	// Create temporary directory structure with multiple git repositories
	tmpDir := t.TempDir()

	// Create main git repo
	mainGitDir := filepath.Join(tmpDir, ".git")
	mainHooksDir := filepath.Join(mainGitDir, "hooks")
	if err := os.MkdirAll(mainHooksDir, 0755); err != nil {
		t.Fatalf("Failed to create main .git/hooks directory: %v", err)
	}

	// Create workspace1 git repo
	workspace1Dir := filepath.Join(tmpDir, "workspace1")
	if err := os.MkdirAll(workspace1Dir, 0755); err != nil {
		t.Fatalf("Failed to create workspace1 directory: %v", err)
	}
	ws1GitDir := filepath.Join(workspace1Dir, ".git")
	ws1HooksDir := filepath.Join(ws1GitDir, "hooks")
	if err := os.MkdirAll(ws1HooksDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace1 .git/hooks directory: %v", err)
	}

	// Create workspace2 git repo
	workspace2Dir := filepath.Join(tmpDir, "workspace2")
	if err := os.MkdirAll(workspace2Dir, 0755); err != nil {
		t.Fatalf("Failed to create workspace2 directory: %v", err)
	}
	ws2GitDir := filepath.Join(workspace2Dir, ".git")
	ws2HooksDir := filepath.Join(ws2GitDir, "hooks")
	if err := os.MkdirAll(ws2HooksDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace2 .git/hooks directory: %v", err)
	}

	// Save original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origWd)

	t.Run("install-hook finds all workspaces", func(t *testing.T) {
		// Change to root directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to tmpDir: %v", err)
		}

		// Run install-hook
		if err := runInstallHook(installHookCmd, []string{}); err != nil {
			t.Fatalf("install-hook failed: %v", err)
		}

		// Verify hooks were installed in all repositories
		if !hooks.IsInstalled(tmpDir) {
			t.Errorf("Hook was not installed at root: %s", tmpDir)
		}
		if !hooks.IsInstalled(workspace1Dir) {
			t.Errorf("Hook was not installed at workspace1: %s", workspace1Dir)
		}
		if !hooks.IsInstalled(workspace2Dir) {
			t.Errorf("Hook was not installed at workspace2: %s", workspace2Dir)
		}
	})

	t.Run("uninstall-hook removes all hooks", func(t *testing.T) {
		// Change to root directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to tmpDir: %v", err)
		}

		// Run uninstall-hook
		if err := runUninstallHook(uninstallHookCmd, []string{}); err != nil {
			t.Fatalf("uninstall-hook failed: %v", err)
		}

		// Verify hooks were removed from all repositories
		if hooks.IsInstalled(tmpDir) {
			t.Errorf("Hook is still installed at root: %s", tmpDir)
		}
		if hooks.IsInstalled(workspace1Dir) {
			t.Errorf("Hook is still installed at workspace1: %s", workspace1Dir)
		}
		if hooks.IsInstalled(workspace2Dir) {
			t.Errorf("Hook is still installed at workspace2: %s", workspace2Dir)
		}
	})

	t.Run("install-hook skips already installed hooks", func(t *testing.T) {
		// Change to root directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to tmpDir: %v", err)
		}

		// Install hooks first time
		if err := runInstallHook(installHookCmd, []string{}); err != nil {
			t.Fatalf("First install-hook failed: %v", err)
		}

		// Install hooks second time (should skip)
		if err := runInstallHook(installHookCmd, []string{}); err != nil {
			t.Fatalf("Second install-hook failed: %v", err)
		}

		// Verify hooks are still installed
		if !hooks.IsInstalled(tmpDir) {
			t.Errorf("Hook was not installed at root after second install")
		}
		if !hooks.IsInstalled(workspace1Dir) {
			t.Errorf("Hook was not installed at workspace1 after second install")
		}
		if !hooks.IsInstalled(workspace2Dir) {
			t.Errorf("Hook was not installed at workspace2 after second install")
		}
	})
}

func TestHookCommandsSingleRepository(t *testing.T) {
	// Create temporary directory with just a git repo
	tmpDir := t.TempDir()

	// Initialize git repo
	gitDir := filepath.Join(tmpDir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("Failed to create .git/hooks directory: %v", err)
	}

	// Save original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origWd)

	t.Run("install-hook works with single repository", func(t *testing.T) {
		// Change to git repo directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to tmpDir: %v", err)
		}

		// Run install-hook
		if err := runInstallHook(installHookCmd, []string{}); err != nil {
			t.Fatalf("install-hook failed: %v", err)
		}

		// Verify hook was installed
		if !hooks.IsInstalled(tmpDir) {
			t.Errorf("Hook was not installed: %s", tmpDir)
		}
	})

	t.Run("uninstall-hook removes hook", func(t *testing.T) {
		// Change to git repo directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to tmpDir: %v", err)
		}

		// Run uninstall-hook
		if err := runUninstallHook(uninstallHookCmd, []string{}); err != nil {
			t.Fatalf("uninstall-hook failed: %v", err)
		}

		// Verify hook was removed
		if hooks.IsInstalled(tmpDir) {
			t.Errorf("Hook is still installed after uninstall: %s", tmpDir)
		}
	})
}

func TestHookCommandsErrorCases(t *testing.T) {
	// Create temporary directory without git repo
	tmpDir := t.TempDir()

	// Save original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origWd)

	t.Run("install-hook fails outside git repo", func(t *testing.T) {
		// Change to non-git directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to tmpDir: %v", err)
		}

		// Run install-hook
		err := runInstallHook(installHookCmd, []string{})
		if err == nil {
			t.Error("Expected error when running install-hook outside git repo, got nil")
		}
	})

	t.Run("uninstall-hook fails outside git repo", func(t *testing.T) {
		// Change to non-git directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to tmpDir: %v", err)
		}

		// Run uninstall-hook
		err := runUninstallHook(uninstallHookCmd, []string{})
		if err == nil {
			t.Error("Expected error when running uninstall-hook outside git repo, got nil")
		}
	})
}

func TestFindAllGitRoots(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create main git repo
	mainGitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(mainGitDir, 0755); err != nil {
		t.Fatalf("Failed to create main .git directory: %v", err)
	}

	// Create nested workspace
	ws1Dir := filepath.Join(tmpDir, "workspace1")
	ws1GitDir := filepath.Join(ws1Dir, ".git")
	if err := os.MkdirAll(ws1GitDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace1 .git directory: %v", err)
	}

	// Create node_modules (should be skipped)
	nodeModulesDir := filepath.Join(tmpDir, "node_modules", "some-package", ".git")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatalf("Failed to create node_modules .git directory: %v", err)
	}

	t.Run("finds all git roots except skipped directories", func(t *testing.T) {
		roots, err := findAllGitRoots(tmpDir)
		if err != nil {
			t.Fatalf("findAllGitRoots failed: %v", err)
		}

		// Should find main and workspace1, but not node_modules
		if len(roots) != 2 {
			t.Errorf("Expected 2 git roots, got %d: %v", len(roots), roots)
		}

		found := make(map[string]bool)
		for _, root := range roots {
			found[root] = true
		}

		if !found[tmpDir] {
			t.Errorf("Main repository not found in roots: %v", roots)
		}
		if !found[ws1Dir] {
			t.Errorf("Workspace1 not found in roots: %v", roots)
		}
	})
}
