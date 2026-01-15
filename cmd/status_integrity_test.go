package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yejune/git-multirepo/internal/manifest"
)

// TestStatusIntegrity_AllClean tests status when all integrity checks pass
func TestStatusIntegrity_AllClean(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create a clean workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "apps/frontend"})

	t.Run("status shows all clean", func(t *testing.T) {
		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show integrity check section
		if !strings.Contains(output, "Integrity Check") {
			t.Errorf("output should contain 'Integrity Check', got: %s", output)
		}

		// Note: In test environment, local paths are used as remote repos,
		// so we expect "Local path repo URL detected" warning.
		// The test verifies that the integrity check runs, not that everything is "clean".

		// Should NOT show nested manifest or unregistered workspace issues
		// (other than local path warning which is expected in test env)
		if strings.Contains(output, "Nested manifest") || strings.Contains(output, "중첩된 manifest") {
			t.Errorf("output should NOT show nested manifest issue, got: %s", output)
		}

		if strings.Contains(output, "unregistered") && !strings.Contains(output, "Local path") {
			t.Errorf("output should NOT show unregistered workspace (except local path warning), got: %s", output)
		}

		// Should show workspace status
		if !strings.Contains(output, "apps/frontend") {
			t.Errorf("output should show workspace path, got: %s", output)
		}
	})
}

// TestStatusIntegrity_NestedManifest tests detection of nested manifests
func TestStatusIntegrity_NestedManifest(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "apps/backend"})

	// Create nested manifest (CRITICAL issue)
	nestedManifestPath := filepath.Join(dir, "apps/backend", ".git.multirepos")
	os.WriteFile(nestedManifestPath, []byte("workspaces: []"), 0644)

	t.Run("status detects nested manifest", func(t *testing.T) {
		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show critical error
		if !strings.Contains(output, "CRITICAL") && !strings.Contains(output, "심각") {
			t.Errorf("output should show CRITICAL error, got: %s", output)
		}

		if !strings.Contains(output, "Nested manifest") && !strings.Contains(output, "중첩된 manifest") {
			t.Errorf("output should mention nested manifest, got: %s", output)
		}

		// Should show the path
		if !strings.Contains(output, "apps/backend") {
			t.Errorf("output should show nested manifest path, got: %s", output)
		}

		// Should show fix command
		if !strings.Contains(output, "rm") {
			t.Errorf("output should show removal command, got: %s", output)
		}
	})
}

// TestStatusIntegrity_ParentManifest tests detection of parent manifests
func TestStatusIntegrity_ParentManifest(t *testing.T) {
	// Create temp directory structure with parent manifest
	parentDir := t.TempDir()
	childDir := filepath.Join(parentDir, "child")

	// Initialize parent as git repo with manifest
	exec.Command("git", "-C", parentDir, "init").Run()
	exec.Command("git", "-C", parentDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", parentDir, "config", "user.name", "Test User").Run()

	// Create parent manifest
	parentManifest := filepath.Join(parentDir, ".git.multirepos")
	os.WriteFile(parentManifest, []byte("workspaces: []"), 0644)

	// Create initial commit in parent
	readme := filepath.Join(parentDir, "README.md")
	os.WriteFile(readme, []byte("# Parent"), 0644)
	exec.Command("git", "-C", parentDir, "add", ".").Run()
	exec.Command("git", "-C", parentDir, "commit", "-m", "Initial commit").Run()

	// Create child directory with its own git repo
	os.MkdirAll(childDir, 0755)
	exec.Command("git", "-C", childDir, "init").Run()
	exec.Command("git", "-C", childDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", childDir, "config", "user.name", "Test User").Run()

	// Create child manifest
	childManifest := filepath.Join(childDir, ".git.multirepos")
	os.WriteFile(childManifest, []byte("workspaces: []"), 0644)

	// Create initial commit in child
	childReadme := filepath.Join(childDir, "README.md")
	os.WriteFile(childReadme, []byte("# Child"), 0644)
	exec.Command("git", "-C", childDir, "add", ".").Run()
	exec.Command("git", "-C", childDir, "commit", "-m", "Initial commit").Run()

	// Save current directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to child directory
	os.Chdir(childDir)

	t.Run("status detects parent manifest", func(t *testing.T) {
		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show warning
		if !strings.Contains(output, "WARNING") && !strings.Contains(output, "경고") {
			t.Errorf("output should show WARNING, got: %s", output)
		}

		if !strings.Contains(output, "Parent manifest") && !strings.Contains(output, "부모 manifest") {
			t.Errorf("output should mention parent manifest, got: %s", output)
		}
	})
}

// TestStatusIntegrity_UnregisteredWorkspace tests detection of unregistered workspaces
func TestStatusIntegrity_UnregisteredWorkspace(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create a registered workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "apps/registered"})

	// Create an unregistered workspace (has .git but not in manifest)
	unregisteredPath := filepath.Join(dir, "apps/unregistered")
	os.MkdirAll(unregisteredPath, 0755)
	exec.Command("git", "-C", unregisteredPath, "init").Run()
	exec.Command("git", "-C", unregisteredPath, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", unregisteredPath, "config", "user.name", "Test User").Run()

	// Create initial commit
	unregReadme := filepath.Join(unregisteredPath, "README.md")
	os.WriteFile(unregReadme, []byte("# Unregistered"), 0644)
	exec.Command("git", "-C", unregisteredPath, "add", ".").Run()
	exec.Command("git", "-C", unregisteredPath, "commit", "-m", "Initial").Run()

	t.Run("status detects unregistered workspace", func(t *testing.T) {
		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show warning
		if !strings.Contains(output, "unregistered") && !strings.Contains(output, "미등록") {
			t.Errorf("output should mention unregistered workspace, got: %s", output)
		}

		// Should show the path
		if !strings.Contains(output, "apps/unregistered") {
			t.Errorf("output should show unregistered workspace path, got: %s", output)
		}

		// Should show fix command
		if !strings.Contains(output, "git multirepo sync") {
			t.Errorf("output should suggest sync command, got: %s", output)
		}
	})
}

// TestStatusIntegrity_RemoteURLMismatch tests detection of remote URL mismatches
func TestStatusIntegrity_RemoteURLMismatch(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "libs/utils"})

	// Manually change the remote URL in the workspace
	wsPath := filepath.Join(dir, "libs/utils")
	exec.Command("git", "-C", wsPath, "remote", "set-url", "origin", "https://github.com/different/repo.git").Run()

	t.Run("status detects remote URL mismatch", func(t *testing.T) {
		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show warning
		if !strings.Contains(output, "Remote URL") && !strings.Contains(output, "Remote URL") {
			t.Errorf("output should mention remote URL mismatch, got: %s", output)
		}

		// Should show workspace path
		if !strings.Contains(output, "libs/utils") {
			t.Errorf("output should show workspace with mismatch, got: %s", output)
		}

		// Should show expected and actual URLs
		if !strings.Contains(output, "Expected") && !strings.Contains(output, "예상") {
			t.Errorf("output should show expected URL, got: %s", output)
		}

		if !strings.Contains(output, "Actual") && !strings.Contains(output, "실제") {
			t.Errorf("output should show actual URL, got: %s", output)
		}
	})
}

// TestStatusIntegrity_MultipleIssues tests when multiple integrity issues exist
func TestStatusIntegrity_MultipleIssues(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create workspace with nested manifest
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "apps/web"})

	nestedManifestPath := filepath.Join(dir, "apps/web", ".git.multirepos")
	os.WriteFile(nestedManifestPath, []byte("workspaces: []"), 0644)

	// Create unregistered workspace
	unregisteredPath := filepath.Join(dir, "services/api")
	os.MkdirAll(unregisteredPath, 0755)
	exec.Command("git", "-C", unregisteredPath, "init").Run()
	exec.Command("git", "-C", unregisteredPath, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", unregisteredPath, "config", "user.name", "Test User").Run()

	unregReadme := filepath.Join(unregisteredPath, "README.md")
	os.WriteFile(unregReadme, []byte("# API"), 0644)
	exec.Command("git", "-C", unregisteredPath, "add", ".").Run()
	exec.Command("git", "-C", unregisteredPath, "commit", "-m", "Initial").Run()

	t.Run("status shows all issues", func(t *testing.T) {
		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show nested manifest critical error
		if !strings.Contains(output, "CRITICAL") && !strings.Contains(output, "심각") {
			t.Errorf("output should show CRITICAL for nested manifest, got: %s", output)
		}

		// Should show unregistered workspace warning
		if !strings.Contains(output, "unregistered") && !strings.Contains(output, "미등록") {
			t.Errorf("output should show unregistered workspace warning, got: %s", output)
		}

		// Should mention both paths
		if !strings.Contains(output, "apps/web") {
			t.Errorf("output should mention apps/web, got: %s", output)
		}

		if !strings.Contains(output, "services/api") {
			t.Errorf("output should mention services/api, got: %s", output)
		}
	})
}

// TestStatusIntegrity_EmptyManifest tests with no workspaces
func TestStatusIntegrity_EmptyManifest(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create empty manifest
	m := &manifest.Manifest{
		Workspaces: []manifest.WorkspaceEntry{},
	}
	manifest.Save(dir, m)

	t.Run("status with empty manifest", func(t *testing.T) {
		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should still show integrity check
		if !strings.Contains(output, "Integrity Check") {
			t.Errorf("output should show integrity check section, got: %s", output)
		}

		// Should show no repositories message
		if !strings.Contains(output, "No repositories") && !strings.Contains(output, "등록된 repository가 없습니다") {
			t.Errorf("output should show no repositories message, got: %s", output)
		}
	})
}

// TestStatusHookInfo tests hook installation status display
func TestStatusHookInfo(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "apps/frontend"})

	t.Run("status shows hook not installed", func(t *testing.T) {
		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show hook status
		if !strings.Contains(output, "Hooks:") {
			t.Errorf("output should show hook status, got: %s", output)
		}

		// Should show 0/N installed
		if !strings.Contains(output, "0/") {
			t.Errorf("output should show 0 hooks installed, got: %s", output)
		}

		// Should show install suggestion
		if !strings.Contains(output, "git multirepo install-hook") {
			t.Errorf("output should suggest install-hook command, got: %s", output)
		}
	})

	t.Run("status after hook installation", func(t *testing.T) {
		// Install hook
		runInstallHook(installHookCmd, []string{})

		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show hook status
		if !strings.Contains(output, "Hooks:") {
			t.Errorf("output should show hook status, got: %s", output)
		}

		// Should show all hooks installed (at least root repo)
		if !strings.Contains(output, "/1 installed") && !strings.Contains(output, "/2 installed") {
			t.Errorf("output should show hooks installed, got: %s", output)
		}

		// Should NOT show install suggestion when all hooks installed
		installedCount := 0
		totalCount := 0
		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "Hooks:") {
				// Parse "Hooks: X/Y installed"
				parts := strings.Split(line, " ")
				for i, part := range parts {
					if part == "Hooks:" && i+1 < len(parts) {
						counts := strings.Split(parts[i+1], "/")
						if len(counts) == 2 {
							installedCount = len(counts[0])
							totalCount = len(counts[1])
						}
					}
				}
			}
		}

		if installedCount == totalCount && strings.Contains(output, "git multirepo install-hook") {
			t.Errorf("output should NOT suggest install-hook when all hooks installed, got: %s", output)
		}
	})
}

// TestStatusHookInfo_WithWorkspaces tests hook status with multiple workspaces
func TestStatusHookInfo_WithWorkspaces(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create multiple workspaces
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "apps/frontend"})
	runClone(cloneCmd, []string{remoteRepo, "apps/backend"})

	t.Run("status shows all repos without hooks", func(t *testing.T) {
		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show hook status for all repos (root + 2 workspaces = 3 total)
		if !strings.Contains(output, "Hooks:") {
			t.Errorf("output should show hook status, got: %s", output)
		}

		// Should show install suggestion
		if !strings.Contains(output, "git multirepo install-hook") {
			t.Errorf("output should suggest install-hook command, got: %s", output)
		}

		// Should show ✗ markers for uninstalled hooks
		checkMarks := strings.Count(output, "✗")
		if checkMarks == 0 {
			t.Errorf("output should show ✗ for uninstalled hooks, got: %s", output)
		}
	})

	t.Run("status after partial hook installation", func(t *testing.T) {
		// Install hook only in root
		hooksPath := filepath.Join(dir, ".git", "hooks")
		os.MkdirAll(hooksPath, 0755)
		hookFile := filepath.Join(hooksPath, "post-checkout")
		os.WriteFile(hookFile, []byte("#!/bin/sh\n# git-multirepo post-checkout hook\n# Automatically syncs subs after checkout\n# Runs from current directory (respects hierarchy)\n\nif command -v git-multirepo >/dev/null 2>&1; then\n    cd \"$(pwd)\" && git-multirepo sync\nfi\n"), 0755)

		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show partial installation
		if !strings.Contains(output, "Hooks:") {
			t.Errorf("output should show hook status, got: %s", output)
		}

		// Should show both ✓ and ✗ markers
		checkMarks := strings.Count(output, "✓")
		crossMarks := strings.Count(output, "✗")
		if checkMarks == 0 || crossMarks == 0 {
			t.Errorf("output should show both ✓ and ✗ for partial installation, got: %s", output)
		}

		// Should still show install suggestion
		if !strings.Contains(output, "git multirepo install-hook") {
			t.Errorf("output should suggest install-hook command for partial installation, got: %s", output)
		}
	})
}

// TestStatusHookDifferentiation tests the 4 different hook states
func TestStatusHookDifferentiation(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create 4 workspaces for each state
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "workspace1"}) // Will have git-multirepo only
	runClone(cloneCmd, []string{remoteRepo, "workspace2"}) // Will have git-multirepo + other
	runClone(cloneCmd, []string{remoteRepo, "workspace3"}) // Will have other only
	runClone(cloneCmd, []string{remoteRepo, "workspace4"}) // Will have no hook

	t.Run("shows all 4 states correctly", func(t *testing.T) {
		// 1. Install git-multirepo hook only in workspace1
		ws1HooksPath := filepath.Join(dir, "workspace1", ".git", "hooks")
		os.MkdirAll(ws1HooksPath, 0755)
		ws1HookFile := filepath.Join(ws1HooksPath, "post-checkout")
		os.WriteFile(ws1HookFile, []byte("#!/bin/sh\n# === git-multirepo hook START ===\n# git-multirepo post-checkout hook\n# === git-multirepo hook END ===\n"), 0755)

		// 2. Install git-multirepo + other hook in workspace2
		ws2HooksPath := filepath.Join(dir, "workspace2", ".git", "hooks")
		os.MkdirAll(ws2HooksPath, 0755)
		ws2HookFile := filepath.Join(ws2HooksPath, "post-checkout")
		os.WriteFile(ws2HookFile, []byte("#!/bin/sh\n# Other hook\necho 'other'\n\n# === git-multirepo hook START ===\n# git-multirepo post-checkout hook\n# === git-multirepo hook END ===\n"), 0755)

		// 3. Install other hook only in workspace3
		ws3HooksPath := filepath.Join(dir, "workspace3", ".git", "hooks")
		os.MkdirAll(ws3HooksPath, 0755)
		ws3HookFile := filepath.Join(ws3HooksPath, "post-checkout")
		os.WriteFile(ws3HookFile, []byte("#!/bin/sh\n# Other hook only\necho 'other'\n"), 0755)

		// 4. workspace4 has no hook at all (already in this state)

		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Check output
		if !strings.Contains(output, "Hooks:") {
			t.Errorf("output should show hook status, got: %s", output)
		}

		// Should show correct counts
		// installed = 2 (workspace1 + workspace2)
		// total = 5 (root + 4 workspaces)
		// merged = 1 (workspace2)
		// otherOnly = 1 (workspace3)

		// Check markers
		if !strings.Contains(output, "✓") {
			t.Errorf("output should show ✓ for git-multirepo only hook, got: %s", output)
		}

		if strings.Count(output, "⚠️") < 2 {
			t.Errorf("output should show ⚠️ for merged and other-only hooks, got: %s", output)
		}

		if !strings.Contains(output, "(merged with other hook)") {
			t.Errorf("output should show '(merged with other hook)' for workspace2, got: %s", output)
		}

		if !strings.Contains(output, "(other hook only)") {
			t.Errorf("output should show '(other hook only)' for workspace3, got: %s", output)
		}

		if !strings.Contains(output, "✗") {
			t.Errorf("output should show ✗ for no hook, got: %s", output)
		}

		// Check installation message
		if !strings.Contains(output, "need installation") {
			t.Errorf("output should show 'need installation' for otherOnly repos, got: %s", output)
		}

		if !strings.Contains(output, "git multirepo install-hook") {
			t.Errorf("output should suggest install-hook command, got: %s", output)
		}
	})
}
