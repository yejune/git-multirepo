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

func TestHookCommandsDoesNotAffectParent(t *testing.T) {
	// Create directory structure:
	// tmpDir/
	//   .git/           <- parent (should NOT be affected)
	//   child/
	//     .git/         <- run install-hook from here
	//     subdir/
	//       .git/       <- child (should be affected)

	tmpDir := t.TempDir()

	// Create parent git repo
	parentGitDir := filepath.Join(tmpDir, ".git")
	parentHooksDir := filepath.Join(parentGitDir, "hooks")
	if err := os.MkdirAll(parentHooksDir, 0755); err != nil {
		t.Fatalf("Failed to create parent .git/hooks directory: %v", err)
	}

	// Create child directory with git repo
	childDir := filepath.Join(tmpDir, "child")
	childGitDir := filepath.Join(childDir, ".git")
	childHooksDir := filepath.Join(childGitDir, "hooks")
	if err := os.MkdirAll(childHooksDir, 0755); err != nil {
		t.Fatalf("Failed to create child .git/hooks directory: %v", err)
	}

	// Create subdir under child with git repo
	subdirDir := filepath.Join(childDir, "subdir")
	subdirGitDir := filepath.Join(subdirDir, ".git")
	subdirHooksDir := filepath.Join(subdirGitDir, "hooks")
	if err := os.MkdirAll(subdirHooksDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir .git/hooks directory: %v", err)
	}

	// Save original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origWd)

	t.Run("install-hook from child does not affect parent", func(t *testing.T) {
		// Change to child directory
		if err := os.Chdir(childDir); err != nil {
			t.Fatalf("Failed to change to child directory: %v", err)
		}

		// Run install-hook from child
		if err := runInstallHook(installHookCmd, []string{}); err != nil {
			t.Fatalf("install-hook failed: %v", err)
		}

		// Check parent does NOT have hook
		if hooks.IsInstalled(tmpDir) {
			t.Error("Parent hook should NOT be installed")
		}

		// Check child has hook
		if !hooks.IsInstalled(childDir) {
			t.Error("Child hook should be installed")
		}

		// Check subdir has hook
		if !hooks.IsInstalled(subdirDir) {
			t.Error("Subdir hook should be installed")
		}
	})

	t.Run("uninstall-hook from child does not affect parent", func(t *testing.T) {
		// Manually install hook in parent for testing
		parentHook := filepath.Join(parentHooksDir, "post-checkout")
		testHookContent := []byte("#!/bin/sh\n# test hook\n")
		if err := os.WriteFile(parentHook, testHookContent, 0755); err != nil {
			t.Fatalf("Failed to create test hook in parent: %v", err)
		}

		// Change to child directory
		if err := os.Chdir(childDir); err != nil {
			t.Fatalf("Failed to change to child directory: %v", err)
		}

		// Run uninstall-hook from child
		if err := runUninstallHook(uninstallHookCmd, []string{}); err != nil {
			t.Fatalf("uninstall-hook failed: %v", err)
		}

		// Check parent hook still exists (file should not be deleted)
		if _, err := os.Stat(parentHook); os.IsNotExist(err) {
			t.Error("Parent hook file should still exist (not affected)")
		} else if err != nil {
			t.Errorf("Failed to check parent hook: %v", err)
		}

		// Verify parent hook content is unchanged
		content, err := os.ReadFile(parentHook)
		if err != nil {
			t.Errorf("Failed to read parent hook: %v", err)
		} else if string(content) != string(testHookContent) {
			t.Error("Parent hook content was modified")
		}

		// Check child does NOT have hook
		if hooks.IsInstalled(childDir) {
			t.Error("Child hook should be removed")
		}

		// Check subdir does NOT have hook
		if hooks.IsInstalled(subdirDir) {
			t.Error("Subdir hook should be removed")
		}
	})
}

func TestHookCommandsDeepNesting(t *testing.T) {
	// Create deeply nested structure (5 levels):
	// root/
	//   level1/.git              ← 1단계
	//   level1/level2/.git       ← 2단계
	//   level1/level2/level3/.git           ← 3단계
	//   level1/level2/level3/level4/.git    ← 4단계
	//   level1/level2/level3/level4/level5/.git  ← 5단계

	tmpDir := t.TempDir()

	// Build nested structure
	levels := []string{
		"level1",
		filepath.Join("level1", "level2"),
		filepath.Join("level1", "level2", "level3"),
		filepath.Join("level1", "level2", "level3", "level4"),
		filepath.Join("level1", "level2", "level3", "level4", "level5"),
	}

	levelPaths := make([]string, 0, len(levels))

	for _, levelPath := range levels {
		fullPath := filepath.Join(tmpDir, levelPath)
		gitDir := filepath.Join(fullPath, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")

		if err := os.MkdirAll(hooksDir, 0755); err != nil {
			t.Fatalf("Failed to create %s: %v", hooksDir, err)
		}

		levelPaths = append(levelPaths, fullPath)
	}

	// Save original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origWd)

	t.Run("install-hook works with deeply nested structure", func(t *testing.T) {
		// Change to root directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to tmpDir: %v", err)
		}

		// Run install-hook
		if err := runInstallHook(installHookCmd, []string{}); err != nil {
			t.Fatalf("install-hook failed: %v", err)
		}

		// Verify all hooks are installed
		for i, levelPath := range levelPaths {
			if !hooks.IsInstalled(levelPath) {
				t.Errorf("Hook not installed at level %d (%s)", i+1, levelPath)
			}
		}
	})

	t.Run("uninstall-hook works with deeply nested structure", func(t *testing.T) {
		// Change to root directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to tmpDir: %v", err)
		}

		// Run uninstall-hook
		if err := runUninstallHook(uninstallHookCmd, []string{}); err != nil {
			t.Fatalf("uninstall-hook failed: %v", err)
		}

		// Verify all hooks are removed
		for i, levelPath := range levelPaths {
			if hooks.IsInstalled(levelPath) {
				t.Errorf("Hook still exists at level %d (%s)", i+1, levelPath)
			}
		}
	})

	t.Run("install-hook from middle level only affects descendants", func(t *testing.T) {
		// Install from level3 (index 2)
		level3Dir := levelPaths[2]

		if err := os.Chdir(level3Dir); err != nil {
			t.Fatalf("Failed to change to level3 directory: %v", err)
		}

		// Run install-hook from level3
		if err := runInstallHook(installHookCmd, []string{}); err != nil {
			t.Fatalf("install-hook failed: %v", err)
		}

		// Check level1, level2: should NOT have hooks (ancestors)
		ancestorPaths := levelPaths[:2]
		for i, levelPath := range ancestorPaths {
			if hooks.IsInstalled(levelPath) {
				t.Errorf("Ancestor hook should NOT exist at level %d (%s)", i+1, levelPath)
			}
		}

		// Check level3, level4, level5: should HAVE hooks (self + descendants)
		descendantPaths := levelPaths[2:]
		for i, levelPath := range descendantPaths {
			if !hooks.IsInstalled(levelPath) {
				t.Errorf("Hook should exist at level %d (%s)", i+3, levelPath)
			}
		}
	})
}
