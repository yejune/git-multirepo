package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHookMergeIntegration tests the complete install -> uninstall flow with merging
func TestHookMergeIntegration(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")
	os.MkdirAll(hooksDir, 0755)

	hookPath := filepath.Join(hooksDir, "post-checkout")

	// Step 1: Create existing custom hook
	customHook := `#!/bin/sh
# Custom pre-existing hook
echo "Running custom hook"
exit 0
`
	if err := os.WriteFile(hookPath, []byte(customHook), 0755); err != nil {
		t.Fatalf("Failed to create custom hook: %v", err)
	}

	// Step 2: Install our hook (should merge)
	if err := Install(dir); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify both hooks exist
	content, _ := os.ReadFile(hookPath)
	contentStr := string(content)
	if !strings.Contains(contentStr, "Running custom hook") {
		t.Error("Custom hook should be preserved after install")
	}
	if !strings.Contains(contentStr, hookMarkerStart) {
		t.Error("Our hook should be added after install")
	}
	if !strings.Contains(contentStr, "git-multirepo sync") {
		t.Error("Our hook logic should be present")
	}

	// Step 3: Verify IsInstalled returns true
	if !IsInstalled(dir) {
		t.Error("IsInstalled should return true after installation")
	}

	// Step 4: Verify HasHook returns true
	if !HasHook(dir) {
		t.Error("HasHook should return true")
	}

	// Step 5: Try to install again (should skip)
	beforeContent, _ := os.ReadFile(hookPath)
	if err := Install(dir); err != nil {
		t.Fatalf("Second install failed: %v", err)
	}
	afterContent, _ := os.ReadFile(hookPath)
	if string(beforeContent) != string(afterContent) {
		t.Error("Second install should not modify the hook")
	}

	// Step 6: Uninstall (should remove only our part)
	if err := Uninstall(dir); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify custom hook remains
	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal("Hook file should still exist after uninstall")
	}
	contentStr = string(content)
	if !strings.Contains(contentStr, "Running custom hook") {
		t.Error("Custom hook should remain after uninstall")
	}
	if strings.Contains(contentStr, hookMarkerStart) {
		t.Error("Our hook should be removed after uninstall")
	}

	// Step 7: Verify IsInstalled returns false
	if IsInstalled(dir) {
		t.Error("IsInstalled should return false after uninstall")
	}

	// Step 8: Verify HasHook still returns true (custom hook exists)
	if !HasHook(dir) {
		t.Error("HasHook should still return true (custom hook exists)")
	}
}

// TestHookOnlyOurs tests install/uninstall when only our hook exists
func TestHookOnlyOurs(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")
	os.MkdirAll(hooksDir, 0755)

	hookPath := filepath.Join(hooksDir, "post-checkout")

	// Install our hook
	if err := Install(dir); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify installed
	if !IsInstalled(dir) {
		t.Error("IsInstalled should return true")
	}
	if !HasHook(dir) {
		t.Error("HasHook should return true")
	}

	// Uninstall
	if err := Uninstall(dir); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// File should be deleted
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("Hook file should be deleted when only our hook existed")
	}
	if IsInstalled(dir) {
		t.Error("IsInstalled should return false after uninstall")
	}
	if HasHook(dir) {
		t.Error("HasHook should return false after file deletion")
	}
}

// TestStatusDifferentiation tests the three hook states
func TestStatusDifferentiation(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")
	os.MkdirAll(hooksDir, 0755)

	hookPath := filepath.Join(hooksDir, "post-checkout")

	// State 1: No hook (HookNone)
	if IsInstalled(dir) {
		t.Error("Should not be installed initially")
	}
	if HasHook(dir) {
		t.Error("Should not have any hook initially")
	}

	// State 2: Other hook (HookOther)
	customHook := "#!/bin/sh\necho custom"
	os.WriteFile(hookPath, []byte(customHook), 0755)
	if IsInstalled(dir) {
		t.Error("Should not detect our hook")
	}
	if !HasHook(dir) {
		t.Error("Should detect other hook exists")
	}

	// State 3: Our hook installed (HookInstalled)
	Install(dir)
	if !IsInstalled(dir) {
		t.Error("Should detect our hook is installed")
	}
	if !HasHook(dir) {
		t.Error("Should detect hook exists")
	}
}
