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

		// Should show "Hook:" line for each repository
		if !strings.Contains(output, "Hook:") {
			t.Errorf("output should show hook status per repository, got: %s", output)
		}

		// Should show ✗ for root (post-checkout not installed)
		if !strings.Contains(output, "✗") {
			t.Errorf("output should show ✗ for uninstalled hooks, got: %s", output)
		}

		// Should show ✓ for workspace (post-commit auto-installed during clone)
		if !strings.Contains(output, "✓") {
			t.Errorf("output should show ✓ for workspace post-commit hook, got: %s", output)
		}

		// Should show "post-commit" for workspace hook
		if !strings.Contains(output, "post-commit") {
			t.Errorf("output should show post-commit hook type, got: %s", output)
		}

		// Should show summary with 1/2 installed (workspace only)
		if !strings.Contains(output, "Summary:") || !strings.Contains(output, "1/2") {
			t.Errorf("output should show summary with 1/2 hooks installed, got: %s", output)
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

		// Should show "Hook:" line for each repository
		if !strings.Contains(output, "Hook:") {
			t.Errorf("output should show hook status per repository, got: %s", output)
		}

		// Should show ✓ for installed hooks
		if !strings.Contains(output, "✓") {
			t.Errorf("output should show ✓ for installed hooks, got: %s", output)
		}

		// Should show "post-checkout" for root
		if !strings.Contains(output, "post-checkout") {
			t.Errorf("output should show post-checkout hook type, got: %s", output)
		}

		// Should show "post-commit" for workspace
		if !strings.Contains(output, "post-commit") {
			t.Errorf("output should show post-commit hook type, got: %s", output)
		}

		// Should show summary with all hooks installed (1 root, 1 workspace)
		if !strings.Contains(output, "All 2 hooks installed (1 root, 1 workspaces)") {
			t.Errorf("output should show 'All 2 hooks installed (1 root, 1 workspaces)', got: %s", output)
		}

		// When all hooks installed, should NOT show install-hook suggestion
		if strings.Contains(output, "git multirepo install-hook") {
			t.Errorf("output should NOT suggest install-hook when all hooks installed, got: %s", output)
		}
	})
}

// TestStatusHookInfo_WithWorkspaces tests hook status with multiple workspaces
func TestStatusHookInfo_WithWorkspaces(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create multiple workspaces
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "apps/frontend"})
	runClone(cloneCmd, []string{remoteRepo, "apps/backend"})

	t.Run("status shows root without hook, workspaces with post-commit", func(t *testing.T) {
		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show "Hook:" line for each repository
		hookLines := strings.Count(output, "Hook:")
		if hookLines < 3 { // root + 2 workspaces
			t.Errorf("output should show hook status for all repos (expected at least 3), got %d lines with 'Hook:'", hookLines)
		}

		// Should show ✗ for root (post-checkout not installed)
		if !strings.Contains(output, "✗") {
			t.Errorf("output should show ✗ for root without post-checkout, got: %s", output)
		}

		// Should show ✓ for workspaces (post-commit auto-installed)
		checkMarks := strings.Count(output, "✓")
		if checkMarks < 2 {
			t.Errorf("output should show ✓ for 2 workspaces with post-commit, got: %s", output)
		}

		// Should show "post-commit" for workspace hooks
		if !strings.Contains(output, "post-commit") {
			t.Errorf("output should show post-commit hook type for workspaces, got: %s", output)
		}

		// Should show install suggestion (root still needs post-checkout)
		if !strings.Contains(output, "git multirepo install-hook") {
			t.Errorf("output should suggest install-hook command, got: %s", output)
		}

		// Summary should show 2/3 installed (2 workspaces)
		if !strings.Contains(output, "2/3") {
			t.Errorf("output should show 2/3 hooks installed, got: %s", output)
		}
	})

	t.Run("status after install-hook completes all hooks", func(t *testing.T) {
		// Install hook in root (workspaces already have post-commit)
		runInstallHook(installHookCmd, []string{})

		output := captureOutput(func() {
			runStatus(statusCmd, []string{})
		})

		// Should show "Hook:" line for each repository
		hookLines := strings.Count(output, "Hook:")
		if hookLines < 3 { // root + 2 workspaces
			t.Errorf("output should show hook status for all repos, got %d lines with 'Hook:'", hookLines)
		}

		// Should show ✓ for all repos (root post-checkout + workspace post-commit)
		checkMarks := strings.Count(output, "✓")
		if checkMarks < 3 {
			t.Errorf("output should show ✓ for all 3 repos, got %d checkmarks", checkMarks)
		}

		// Should show both hook types
		if !strings.Contains(output, "post-checkout") {
			t.Errorf("output should show post-checkout for root, got: %s", output)
		}
		if !strings.Contains(output, "post-commit") {
			t.Errorf("output should show post-commit for workspaces, got: %s", output)
		}

		// Should show "All hooks installed" with breakdown
		if !strings.Contains(output, "All 3 hooks installed (1 root, 2 workspaces)") {
			t.Errorf("output should show 'All 3 hooks installed (1 root, 2 workspaces)', got: %s", output)
		}

		// Should NOT show install suggestion when all hooks installed
		if strings.Contains(output, "git multirepo install-hook") {
			t.Errorf("output should NOT suggest install-hook when all hooks installed, got: %s", output)
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

		// Should show "Hook:" line for each repository (root + 4 workspaces = 5)
		hookLines := strings.Count(output, "Hook:")
		if hookLines < 5 {
			t.Errorf("output should show hook status for all repos (expected 5), got %d lines with 'Hook:'", hookLines)
		}

		// Should show correct counts
		// installed = 2 (workspace1 + workspace2)
		// total = 5 (root + 4 workspaces)
		// merged = 1 (workspace2)
		// otherOnly = 1 (workspace3)

		// Check markers for each state
		// 1. ✓ for git-multirepo only (workspace1)
		if !strings.Contains(output, "✓") {
			t.Errorf("output should show ✓ for git-multirepo only hook, got: %s", output)
		}

		// 2. ⚠️ for merged (workspace2) and other-only (workspace3)
		if strings.Count(output, "⚠️") < 2 {
			t.Errorf("output should show ⚠️ for merged and other-only hooks, got: %s", output)
		}

		// 3. Check specific descriptions
		if !strings.Contains(output, "Merged with other hook") {
			t.Errorf("output should show 'Merged with other hook' for workspace2, got: %s", output)
		}

		if !strings.Contains(output, "Other hook only") {
			t.Errorf("output should show 'Other hook only' for workspace3, got: %s", output)
		}

		// 4. ✗ for no hook (workspace4 and root)
		if !strings.Contains(output, "✗") {
			t.Errorf("output should show ✗ for no hook, got: %s", output)
		}

		// Check summary shows correct installation message
		if !strings.Contains(output, "Summary:") {
			t.Errorf("output should show summary, got: %s", output)
		}

		if !strings.Contains(output, "need installation") {
			t.Errorf("output should show 'need installation' for otherOnly repos, got: %s", output)
		}

		if !strings.Contains(output, "git multirepo install-hook") {
			t.Errorf("output should suggest install-hook command, got: %s", output)
		}
	})
}
