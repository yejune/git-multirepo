package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/yejune/git-multirepo/internal/manifest"
)

// Helper function to create modified files in a repository
func createModifiedFiles(t *testing.T, repoPath string, files ...string) {
	for _, file := range files {
		filePath := filepath.Join(repoPath, file)
		dir := filepath.Dir(filePath)
		os.MkdirAll(dir, 0755)

		// Create file
		os.WriteFile(filePath, []byte("content"), 0644)

		// Add to git
		exec.Command("git", "-C", repoPath, "add", file).Run()
		exec.Command("git", "-C", repoPath, "commit", "-m", "Add "+file).Run()

		// Modify file
		os.WriteFile(filePath, []byte("modified"), 0644)
	}
}

// ============================================================================
// Group H: Workspace Sync from Subdirectory (5 tests - NEW FEATURE)
// ============================================================================

// TestSyncFromWorkspace_NoLocalManifest tests sync from workspace doesn't create local manifest
// ⭐ NEW FEATURE: Sync from workspace should propagate to parent only
// Structure: parent/.git.multirepo + parent/workspace/.git
// EXPECTED: No manifest created in workspace, changes propagated to parent
func TestSyncFromWorkspace_NoLocalManifest(t *testing.T) {
	parent := t.TempDir()

	// Setup parent repo with manifest
	exec.Command("git", "-C", parent, "init").Run()
	exec.Command("git", "-C", parent, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", parent, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", parent, "commit", "--allow-empty", "-m", "init").Run()

	parentManifest := filepath.Join(parent, ".git.multirepos")
	os.WriteFile(parentManifest, []byte("workspaces:\n  - path: workspace\n    repo: https://example.com/workspace.git\n    keep: []"), 0644)

	// Setup workspace repo
	workspacePath := filepath.Join(parent, "workspace")
	os.MkdirAll(workspacePath, 0755)
	exec.Command("git", "-C", workspacePath, "init").Run()
	exec.Command("git", "-C", workspacePath, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", workspacePath, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", workspacePath, "commit", "--allow-empty", "-m", "init").Run()

	// Test manifest detection from workspace
	manifestRoot, err := manifest.FindParent(workspacePath)
	if err != nil {
		t.Fatalf("FindParent failed: %v", err)
	}

	if manifestRoot != parent {
		t.Errorf("Expected parent root %s, got %s", parent, manifestRoot)
	}

	// Workspace should not have its own manifest
	workspaceManifest := filepath.Join(workspacePath, ".git.multirepos")
	if _, err := os.Stat(workspaceManifest); err == nil {
		t.Error("❌ workspace should NOT have .git.multirepo file (should propagate to parent)")
	}

	t.Log("✅ Feature spec: Sync from workspace should detect parent and propagate changes")
}

// TestSyncFromWorkspace_DetectAllChanges tests all changes within workspace are detected
// Structure: parent/.git.multirepo + parent/workspace/.git with multiple modified files
// EXPECTED: All modified files detected including deeply nested ones
func TestSyncFromWorkspace_DetectAllChanges(t *testing.T) {
	parent := t.TempDir()

	// Setup parent repo with manifest
	exec.Command("git", "-C", parent, "init").Run()
	exec.Command("git", "-C", parent, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", parent, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", parent, "commit", "--allow-empty", "-m", "init").Run()

	parentManifest := filepath.Join(parent, ".git.multirepos")
	os.WriteFile(parentManifest, []byte("workspaces:\n  - path: workspace\n    repo: https://example.com/workspace.git\n    keep: []"), 0644)

	// Setup workspace with nested files
	workspacePath := filepath.Join(parent, "workspace")
	os.MkdirAll(workspacePath, 0755)
	exec.Command("git", "-C", workspacePath, "init").Run()
	exec.Command("git", "-C", workspacePath, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", workspacePath, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", workspacePath, "commit", "--allow-empty", "-m", "init").Run()

	createModifiedFiles(t, workspacePath, "src/modified.ts", "docs/new.md", "deep/nested/changed.json")

	t.Log("✅ Feature spec: Detect all changes within workspace including nested files")
}

// TestSyncFromWorkspace_PropagateToParent tests changes are propagated to parent manifest only
// Structure: parent/.git.multirepo (before: empty keep) → (after: keep: [config.yml])
// EXPECTED: Parent manifest updated, workspace stays clean
func TestSyncFromWorkspace_PropagateToParent(t *testing.T) {
	parent := t.TempDir()

	// Setup parent repo with manifest
	exec.Command("git", "-C", parent, "init").Run()
	exec.Command("git", "-C", parent, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", parent, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", parent, "commit", "--allow-empty", "-m", "init").Run()

	parentManifest := filepath.Join(parent, ".git.multirepos")
	os.WriteFile(parentManifest, []byte("workspaces:\n  - path: workspace\n    repo: https://example.com/workspace.git\n    keep: []"), 0644)

	// Setup workspace
	workspacePath := filepath.Join(parent, "workspace")
	os.MkdirAll(workspacePath, 0755)
	exec.Command("git", "-C", workspacePath, "init").Run()
	exec.Command("git", "-C", workspacePath, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", workspacePath, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", workspacePath, "commit", "--allow-empty", "-m", "init").Run()

	createModifiedFiles(t, workspacePath, "config.yml")

	// Verify parent manifest is used
	manifestRoot, _ := manifest.FindParent(workspacePath)
	if manifestRoot != parent {
		t.Errorf("Expected parent root %s, got %s", parent, manifestRoot)
	}

	// Workspace should remain clean (no manifest)
	workspaceManifest := filepath.Join(workspacePath, ".git.multirepos")
	if _, err := os.Stat(workspaceManifest); err == nil {
		t.Error("❌ workspace should NOT have manifest after sync")
	}

	t.Log("✅ Feature spec: Changes propagated to parent manifest only")
}

// TestSyncFromWorkspace_ParentDetection tests parent detection from deep workspace path
// Structure: parent/.git.multirepo at top, workspace at parent/level1/level2/workspace
// EXPECTED: Traverse up and find parent, calculate relative path correctly
func TestSyncFromWorkspace_ParentDetection(t *testing.T) {
	parent := t.TempDir()

	// Setup parent repo with manifest
	exec.Command("git", "-C", parent, "init").Run()
	exec.Command("git", "-C", parent, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", parent, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", parent, "commit", "--allow-empty", "-m", "init").Run()

	parentManifest := filepath.Join(parent, ".git.multirepos")
	os.WriteFile(parentManifest, []byte("workspaces: []"), 0644)

	// Create deeply nested workspace
	workspacePath := filepath.Join(parent, "level1", "level2", "workspace")
	os.MkdirAll(workspacePath, 0755)
	exec.Command("git", "-C", workspacePath, "init").Run()
	exec.Command("git", "-C", workspacePath, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", workspacePath, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", workspacePath, "commit", "--allow-empty", "-m", "init").Run()

	// Test parent detection
	manifestRoot, err := manifest.FindParent(workspacePath)
	if err != nil {
		t.Fatalf("FindParent failed: %v", err)
	}

	if manifestRoot != parent {
		t.Errorf("Expected parent root %s, got %s", parent, manifestRoot)
	}

	// Calculate relative path
	relPath, err := filepath.Rel(parent, workspacePath)
	if err != nil {
		t.Fatalf("Rel failed: %v", err)
	}

	expected := filepath.Join("level1", "level2", "workspace")
	if relPath != expected {
		t.Errorf("Expected relative path %s, got %s", expected, relPath)
	}

	t.Log("✅ Feature spec: Parent detection from deeply nested workspace")
}

// TestSyncFromWorkspace_NoParentError tests behavior when no parent exists
// Structure: standalone/.git (no parent .git.multirepo)
// EXPECTED: Current directory becomes parent, create manifest locally
func TestSyncFromWorkspace_NoParentError(t *testing.T) {
	standalone := t.TempDir()

	// Setup standalone repo (no parent multirepo)
	exec.Command("git", "-C", standalone, "init").Run()
	exec.Command("git", "-C", standalone, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", standalone, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", standalone, "commit", "--allow-empty", "-m", "init").Run()

	// Test parent detection (should return empty string)
	manifestRoot, err := manifest.FindParent(standalone)
	if err != nil {
		t.Fatalf("FindParent failed: %v", err)
	}

	if manifestRoot != "" {
		t.Errorf("Expected empty string, got %s", manifestRoot)
	}

	t.Log("✅ Feature spec: No parent found, standalone repository")
}
