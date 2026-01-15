package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall(t *testing.T) {
	t.Run("install fails when cannot create hooks directory", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		// Create .git as a file, not a directory - prevents MkdirAll from creating hooks
		os.WriteFile(gitDir, []byte("gitdir: /some/path"), 0644)

		err := Install(dir)
		if err == nil {
			t.Error("Install should fail when hooks directory cannot be created")
		}
	})

	t.Run("install fails when cannot write hook file", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)
		// Create post-checkout as a directory to prevent WriteFile
		os.MkdirAll(filepath.Join(hooksDir, "post-checkout"), 0755)

		err := Install(dir)
		if err == nil {
			t.Error("Install should fail when hook file cannot be written")
		}
	})

	t.Run("install to new repo", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0755)

		err := Install(dir)
		if err != nil {
			t.Fatalf("Install failed: %v", err)
		}

		hookPath := filepath.Join(gitDir, "hooks", "post-checkout")
		if _, err := os.Stat(hookPath); os.IsNotExist(err) {
			t.Error("hook file should exist")
		}

		content, _ := os.ReadFile(hookPath)
		if !strings.Contains(string(content), hookMarkerStart) {
			t.Error("hook content should contain marker")
		}

		// Check executable
		info, _ := os.Stat(hookPath)
		if info.Mode().Perm()&0111 == 0 {
			t.Error("hook should be executable")
		}
	})

	t.Run("install merges with existing hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Write existing custom hook
		hookPath := filepath.Join(hooksDir, "post-checkout")
		existingContent := "#!/bin/sh\necho custom hook"
		os.WriteFile(hookPath, []byte(existingContent), 0755)

		// Install our hook
		err := Install(dir)
		if err != nil {
			t.Fatalf("Install failed: %v", err)
		}

		// Both hooks should exist
		content, _ := os.ReadFile(hookPath)
		contentStr := string(content)
		if !strings.Contains(contentStr, "echo custom hook") {
			t.Error("existing hook should be preserved")
		}
		if !strings.Contains(contentStr, hookMarkerStart) {
			t.Error("our hook should be added")
		}
	})

	t.Run("install skips if already installed", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Install first time
		err := Install(dir)
		if err != nil {
			t.Fatalf("First install failed: %v", err)
		}

		hookPath := filepath.Join(hooksDir, "post-checkout")
		firstContent, _ := os.ReadFile(hookPath)

		// Install second time
		err = Install(dir)
		if err != nil {
			t.Fatalf("Second install failed: %v", err)
		}

		secondContent, _ := os.ReadFile(hookPath)
		if string(firstContent) != string(secondContent) {
			t.Error("hook should not be modified when already installed")
		}
	})

	t.Run("install creates hooks directory", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0755)

		err := Install(dir)
		if err != nil {
			t.Fatalf("Install failed: %v", err)
		}

		hooksDir := filepath.Join(gitDir, "hooks")
		info, err := os.Stat(hooksDir)
		if err != nil || !info.IsDir() {
			t.Error("hooks directory should be created")
		}
	})
}

func TestUninstall(t *testing.T) {
	t.Run("uninstall existing hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Install first
		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(postCheckoutHook), 0755)

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Error("hook file should be removed")
		}
	})

	t.Run("uninstall removes only our part", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Create merged hook (existing + ours)
		hookPath := filepath.Join(hooksDir, "post-checkout")
		mergedContent := "#!/bin/sh\necho custom hook\n\n" + postCheckoutHook
		os.WriteFile(hookPath, []byte(mergedContent), 0755)

		// Uninstall
		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		// Existing hook should remain
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Error("hook file should still exist")
		}
		contentStr := string(content)
		if !strings.Contains(contentStr, "echo custom hook") {
			t.Error("existing hook should be preserved")
		}
		if strings.Contains(contentStr, hookMarkerStart) {
			t.Error("our hook should be removed")
		}
	})

	t.Run("uninstall removes file if only our hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Install only our hook
		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(postCheckoutHook), 0755)

		// Uninstall
		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		// File should be deleted
		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Error("hook file should be removed when only our hook exists")
		}
	})

	t.Run("uninstall non-existent hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0755)

		err := Uninstall(dir)
		if err != nil {
			t.Errorf("should not error on non-existent hook: %v", err)
		}
	})

	t.Run("uninstall fails when hook is a directory", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Create post-checkout as a directory - ReadFile will fail with non-NotExist error
		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.MkdirAll(hookPath, 0755)

		err := Uninstall(dir)
		if err == nil {
			t.Error("Uninstall should fail when hook path is a directory")
		}
	})

	t.Run("uninstall fails when cannot remove hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Install our hook first
		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(postCheckoutHook), 0755)

		// Make hooks directory read-only to prevent removal
		if err := os.Chmod(hooksDir, 0555); err != nil {
			t.Skipf("cannot change directory permissions: %v", err)
		}
		defer os.Chmod(hooksDir, 0755) // Cleanup

		err := Uninstall(dir)
		if err == nil {
			t.Error("Uninstall should fail when hook file cannot be removed")
		}
	})
}

func TestIsInstalled(t *testing.T) {
	t.Run("installed", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(postCheckoutHook), 0755)

		if !IsInstalled(dir) {
			t.Error("should return true when hook is installed")
		}
	})

	t.Run("not installed", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0755)

		if IsInstalled(dir) {
			t.Error("should return false when hook is not installed")
		}
	})

	t.Run("custom hook installed", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte("custom"), 0755)

		if IsInstalled(dir) {
			t.Error("should return false when different hook is installed")
		}
	})

	t.Run("merged hook contains our marker", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		hookPath := filepath.Join(hooksDir, "post-checkout")
		mergedContent := "#!/bin/sh\necho custom\n\n" + postCheckoutHook
		os.WriteFile(hookPath, []byte(mergedContent), 0755)

		if !IsInstalled(dir) {
			t.Error("should return true when hook contains our marker")
		}
	})
}

func TestHasHook(t *testing.T) {
	t.Run("hook exists", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte("some hook"), 0755)

		if !HasHook(dir) {
			t.Error("should return true when hook file exists")
		}
	})

	t.Run("hook does not exist", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0755)

		if HasHook(dir) {
			t.Error("should return false when hook file does not exist")
		}
	})
}

// TestHookStringEdgeCases tests string parsing edge cases for hook markers
func TestHookStringEdgeCases(t *testing.T) {
	t.Run("marker with leading/trailing whitespace", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Create hook with markers surrounded by whitespace
		content := `#!/bin/sh
echo "before"

  ` + MarkerStart + `
if command -v git-multirepo >/dev/null 2>&1; then
    cd "$(pwd)" && git-multirepo sync
fi
  ` + MarkerEnd + `

echo "after"`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		// IsInstalled should still find it despite whitespace
		if !IsInstalled(dir) {
			t.Error("IsInstalled should detect marker with surrounding whitespace")
		}

		// Uninstall should remove it cleanly
		if err := Uninstall(dir); err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		resultStr := string(result)

		if strings.Contains(resultStr, MarkerStart) {
			t.Error("Uninstall should remove marker section with whitespace")
		}

		if !strings.Contains(resultStr, `echo "before"`) {
			t.Error("Uninstall should preserve content before marker")
		}

		if !strings.Contains(resultStr, `echo "after"`) {
			t.Error("Uninstall should preserve content after marker")
		}
	})

	t.Run("marker without shebang", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Hook content without shebang
		content := MarkerStart + `
if command -v git-multirepo >/dev/null 2>&1; then
    cd "$(pwd)" && git-multirepo sync
fi
` + MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		// Should still be detected as installed
		if !IsInstalled(dir) {
			t.Error("IsInstalled should work without shebang")
		}

		// Uninstall should remove file completely
		if err := Uninstall(dir); err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Error("Uninstall should remove file when only our hook (no shebang)")
		}
	})

	t.Run("only START marker without END", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
` + MarkerStart + `
if command -v git-multirepo >/dev/null 2>&1; then
    cd "$(pwd)" && git-multirepo sync
fi
# END marker is missing!
echo "other hook"`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		// IsInstalled should still return true (contains START marker)
		// This is current behavior - marker presence check is simple
		if !IsInstalled(dir) {
			t.Error("IsInstalled should detect START marker even without END")
		}

		// Uninstall should handle gracefully (won't find matching pair)
		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall should handle incomplete markers gracefully: %v", err)
		}

		// Content should remain unchanged (no valid marker pair to remove)
		result, _ := os.ReadFile(hookPath)
		if !strings.Contains(string(result), MarkerStart) {
			t.Error("Content should remain unchanged when marker pair is incomplete")
		}
	})

	t.Run("only END marker without START", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
if command -v git-multirepo >/dev/null 2>&1; then
    cd "$(pwd)" && git-multirepo sync
fi
` + MarkerEnd + `
echo "other hook"`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		// IsInstalled should return false (no START marker)
		if IsInstalled(dir) {
			t.Error("IsInstalled should return false without START marker")
		}

		// Uninstall should be no-op
		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall should handle gracefully: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		if !strings.Contains(string(result), MarkerEnd) {
			t.Error("Content should remain unchanged when no START marker")
		}
	})

	t.Run("duplicate markers", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
` + MarkerStart + `
sync 1
` + MarkerEnd + `
echo "middle"
` + MarkerStart + `
sync 2
` + MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		// IsInstalled should detect presence
		if !IsInstalled(dir) {
			t.Error("IsInstalled should detect duplicate markers")
		}

		// Uninstall removes first occurrence
		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		resultStr := string(result)

		// First marker pair should be removed
		firstStart := strings.Index(resultStr, MarkerStart)
		if firstStart != -1 {
			// There's still a marker - check if it's the second one
			beforeFirst := resultStr[:firstStart]
			if !strings.Contains(beforeFirst, "middle") {
				t.Error("First marker pair should be removed, second remains")
			}
		}
	})

	t.Run("windows line endings (CRLF)", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := "#!/bin/sh\r\n" +
			MarkerStart + "\r\n" +
			"sync\r\n" +
			MarkerEnd + "\r\n"

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		// Should work with Windows line endings
		if !IsInstalled(dir) {
			t.Error("IsInstalled should work with CRLF line endings")
		}

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall should handle CRLF: %v", err)
		}

		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Error("Should remove file when only our hook with CRLF")
		}
	})

	t.Run("mixed line endings", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := "#!/bin/sh\n" +
			MarkerStart + "\r\n" +
			"sync\n" +
			MarkerEnd + "\r\n"

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		if !IsInstalled(dir) {
			t.Error("IsInstalled should work with mixed line endings")
		}

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall should handle mixed line endings: %v", err)
		}
	})

	t.Run("very long hook content", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Create 10KB+ of content
		longContent := strings.Repeat("echo 'test'\n", 1000)
		content := "#!/bin/sh\n" +
			longContent + "\n" +
			MarkerStart + "\n" +
			"sync\n" +
			MarkerEnd + "\n" +
			longContent

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		// Should handle large files efficiently
		if !IsInstalled(dir) {
			t.Error("IsInstalled should work with large files")
		}

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall should handle large files: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		if strings.Contains(string(result), MarkerStart) {
			t.Error("Should remove marker section from large file")
		}

		if !strings.Contains(string(result), "echo 'test'") {
			t.Error("Should preserve surrounding content in large file")
		}
	})

	t.Run("special characters in other hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
# Special chars: $VAR ${VAR} $(cmd) \$escaped "quotes" 'single'
echo "test with $SPECIAL_CHARS"
` + MarkerStart + `
sync
` + MarkerEnd + `
# More special: ` + "`backticks`" + ` [brackets] {braces}`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		if !IsInstalled(dir) {
			t.Error("IsInstalled should work with special characters")
		}

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall should handle special characters: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		resultStr := string(result)

		if strings.Contains(resultStr, MarkerStart) {
			t.Error("Should remove marker section")
		}

		if !strings.Contains(resultStr, "$SPECIAL_CHARS") {
			t.Error("Should preserve special characters")
		}

		if !strings.Contains(resultStr, "`backticks`") {
			t.Error("Should preserve backticks")
		}
	})

	t.Run("marker-like content in other hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
# This is not a marker: fake start marker text
echo "fake marker"
` + MarkerStart + `
sync
` + MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		if !IsInstalled(dir) {
			t.Error("IsInstalled should detect real marker")
		}

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		resultStr := string(result)

		// Comment should be preserved
		if !strings.Contains(resultStr, "# This is not a marker") {
			t.Error("Should preserve comment")
		}

		// Real marker should be removed completely
		if strings.Contains(resultStr, MarkerStart) || strings.Contains(resultStr, MarkerEnd) {
			t.Error("Real markers should be completely removed")
		}

		// Other content should be preserved
		if !strings.Contains(resultStr, "echo \"fake marker\"") {
			t.Error("Should preserve other hook content")
		}
	})

	t.Run("empty lines around markers", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh


` + MarkerStart + `


sync


` + MarkerEnd + `


echo "after"
`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		if !IsInstalled(dir) {
			t.Error("IsInstalled should work with many blank lines")
		}

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall should handle blank lines: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		resultStr := string(result)

		if strings.Contains(resultStr, MarkerStart) {
			t.Error("Should remove marker section with blank lines")
		}

		if !strings.Contains(resultStr, `echo "after"`) {
			t.Error("Should preserve content after blank lines")
		}
	})

	t.Run("no newline at end of file", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
` + MarkerStart + `
sync
` + MarkerEnd // No trailing newline

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		if !IsInstalled(dir) {
			t.Error("IsInstalled should work without trailing newline")
		}

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall should handle missing trailing newline: %v", err)
		}

		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Error("Should remove file when only our hook (no trailing newline)")
		}
	})
}

// TestInstallUninstallRoundtrip tests that install+uninstall preserves the original hook
func TestInstallUninstallRoundtrip(t *testing.T) {
	t.Run("roundtrip preserves original hook content", func(t *testing.T) {
		original := `#!/bin/sh
# Original hook
echo "before checkout"
npm install
echo "after checkout"
`

		tmpDir := t.TempDir()
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "post-checkout")
		os.MkdirAll(filepath.Dir(hookPath), 0755)
		os.WriteFile(hookPath, []byte(original), 0755)

		// Install
		if err := Install(tmpDir); err != nil {
			t.Fatalf("Install failed: %v", err)
		}

		// Verify merged
		content, _ := os.ReadFile(hookPath)
		if !strings.Contains(string(content), "npm install") {
			t.Error("Original content lost after install")
		}
		if !strings.Contains(string(content), MarkerStart) {
			t.Error("Our hook not installed")
		}

		// Uninstall
		if err := Uninstall(tmpDir); err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		// Verify original content lines are preserved
		restored, _ := os.ReadFile(hookPath)
		restoredStr := string(restored)

		// Check all meaningful lines are preserved
		originalLines := []string{
			"# Original hook",
			`echo "before checkout"`,
			"npm install",
			`echo "after checkout"`,
		}

		for _, line := range originalLines {
			if !strings.Contains(restoredStr, line) {
				t.Errorf("Original line lost: %s", line)
			}
		}

		// Verify our hook is completely removed
		if strings.Contains(restoredStr, MarkerStart) {
			t.Error("Our marker should be removed")
		}
		if strings.Contains(restoredStr, "git-multirepo") {
			t.Error("Our hook content should be removed")
		}

		// Shebang should still be present
		if !strings.HasPrefix(strings.TrimSpace(restoredStr), "#!/bin/sh") {
			t.Error("Shebang should be preserved")
		}
	})

	t.Run("roundtrip with multiple blank lines", func(t *testing.T) {
		original := `#!/bin/sh


echo "test1"


echo "test2"


`

		tmpDir := t.TempDir()
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "post-checkout")
		os.MkdirAll(filepath.Dir(hookPath), 0755)
		os.WriteFile(hookPath, []byte(original), 0755)

		// Install + Uninstall
		Install(tmpDir)
		Uninstall(tmpDir)

		// Check if blank lines are preserved (may be normalized)
		restored, _ := os.ReadFile(hookPath)
		originalLines := strings.Split(strings.TrimSpace(original), "\n")

		// At minimum, all content lines should be preserved
		for _, line := range originalLines {
			if line != "" && !strings.Contains(string(restored), line) {
				t.Errorf("Line lost: %s", line)
			}
		}
	})

	t.Run("roundtrip with trailing whitespace on lines", func(t *testing.T) {
		original := "#!/bin/sh\necho \"test\"  \n  \necho \"test2\"\n"

		tmpDir := t.TempDir()
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "post-checkout")
		os.MkdirAll(filepath.Dir(hookPath), 0755)
		os.WriteFile(hookPath, []byte(original), 0755)

		Install(tmpDir)
		Uninstall(tmpDir)

		// Lines should be preserved (whitespace may be normalized)
		restored, _ := os.ReadFile(hookPath)
		if !strings.Contains(string(restored), "echo \"test\"") {
			t.Error("Content lost")
		}
	})
}

// TestInstallSpacing tests proper spacing between hooks
func TestInstallSpacing(t *testing.T) {
	t.Run("adds proper spacing between hooks", func(t *testing.T) {
		original := "#!/bin/sh\necho test"

		tmpDir := t.TempDir()
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "post-checkout")
		os.MkdirAll(filepath.Dir(hookPath), 0755)
		os.WriteFile(hookPath, []byte(original), 0755)

		Install(tmpDir)

		content, _ := os.ReadFile(hookPath)
		lines := strings.Split(string(content), "\n")

		// Should have blank line separator
		foundBlank := false
		for i, line := range lines {
			if i > 0 && line == "" {
				foundBlank = true
				break
			}
		}

		if !foundBlank {
			t.Error("No blank line separator between hooks")
		}
	})

	t.Run("handles hook ending with newline", func(t *testing.T) {
		original := "#!/bin/sh\necho test\n"

		tmpDir := t.TempDir()
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "post-checkout")
		os.MkdirAll(filepath.Dir(hookPath), 0755)
		os.WriteFile(hookPath, []byte(original), 0755)

		Install(tmpDir)

		content, _ := os.ReadFile(hookPath)
		// Should not have excessive blank lines (triple newline)
		if strings.Contains(string(content), "\n\n\n") {
			t.Error("Too many blank lines")
		}
	})

	t.Run("handles hook not ending with newline", func(t *testing.T) {
		original := "#!/bin/sh\necho test" // No trailing newline

		tmpDir := t.TempDir()
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "post-checkout")
		os.MkdirAll(filepath.Dir(hookPath), 0755)
		os.WriteFile(hookPath, []byte(original), 0755)

		Install(tmpDir)

		content, _ := os.ReadFile(hookPath)
		// Original content should be preserved
		if !strings.Contains(string(content), "echo test") {
			t.Error("Original content lost")
		}
	})
}

// TestUninstallPreservesBlankLines tests blank line handling during uninstall
func TestUninstallPreservesBlankLines(t *testing.T) {
	t.Run("preserves blank lines before marker", func(t *testing.T) {
		content := `#!/bin/sh
echo "before"


` + MarkerStart + `
sync
` + MarkerEnd

		tmpDir := t.TempDir()
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "post-checkout")
		os.MkdirAll(filepath.Dir(hookPath), 0755)
		os.WriteFile(hookPath, []byte(content), 0755)

		Uninstall(tmpDir)

		result, _ := os.ReadFile(hookPath)
		// May normalize, but content should be preserved
		if !strings.Contains(string(result), "echo \"before\"") {
			t.Error("Content before marker lost")
		}
	})

	t.Run("preserves blank lines after marker", func(t *testing.T) {
		content := MarkerStart + `
sync
` + MarkerEnd + `


echo "after"`

		tmpDir := t.TempDir()
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "post-checkout")
		os.MkdirAll(filepath.Dir(hookPath), 0755)
		os.WriteFile(hookPath, []byte(content), 0755)

		Uninstall(tmpDir)

		result, _ := os.ReadFile(hookPath)
		if !strings.Contains(string(result), "echo \"after\"") {
			t.Error("Content after marker lost")
		}
	})
}

// TestMultipleInstallUninstallCycles tests repeated cycles
func TestMultipleInstallUninstallCycles(t *testing.T) {
	t.Run("multiple cycles preserve original", func(t *testing.T) {
		original := `#!/bin/sh
echo "original hook"
npm install
`

		tmpDir := t.TempDir()
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "post-checkout")
		os.MkdirAll(filepath.Dir(hookPath), 0755)
		os.WriteFile(hookPath, []byte(original), 0755)

		// Cycle 3 times
		for i := 0; i < 3; i++ {
			Install(tmpDir)
			Uninstall(tmpDir)
		}

		result, _ := os.ReadFile(hookPath)
		if !strings.Contains(string(result), "npm install") {
			t.Error("Original lost after multiple cycles")
		}
		if strings.Contains(string(result), MarkerStart) {
			t.Error("Marker not removed")
		}
	})
}

// TestUninstallStringHandling tests precise string manipulation during uninstall
func TestUninstallStringHandling(t *testing.T) {
	t.Run("removes only exact marker section", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		before := `#!/bin/sh
echo "before hook"
`
		middle := MarkerStart + `
if command -v git-multirepo >/dev/null 2>&1; then
    cd "$(pwd)" && git-multirepo sync
fi
` + MarkerEnd

		after := `
echo "after hook"
`

		content := before + middle + after

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		// Uninstall
		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		actual := string(result)

		// Verify marker is completely removed
		if strings.Contains(actual, MarkerStart) || strings.Contains(actual, MarkerEnd) {
			t.Error("Markers should be completely removed")
		}

		// Verify both before and after content preserved
		if !strings.Contains(actual, `echo "before hook"`) {
			t.Error("Before content should be preserved")
		}

		if !strings.Contains(actual, `echo "after hook"`) {
			t.Error("After content should be preserved")
		}

		// Verify shebang is preserved
		if !strings.Contains(actual, "#!/bin/sh") {
			t.Error("Shebang should be preserved")
		}

		// Note: Uninstall uses TrimSpace, so exact blank line preservation
		// is not guaranteed. The important thing is that all non-blank
		// content is preserved correctly.
	})

	t.Run("preserves exact indentation", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
if [ "$condition" = "true" ]; then
    echo "indented line 1"
        echo "extra indented"
    echo "indented line 2"
fi
` + MarkerStart + `
sync
` + MarkerEnd + `
for i in 1 2 3; do
    echo "loop $i"
done`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		resultStr := string(result)

		// Check exact indentation is preserved
		if !strings.Contains(resultStr, "    echo \"indented line 1\"") {
			t.Error("Should preserve 4-space indentation")
		}

		if !strings.Contains(resultStr, "        echo \"extra indented\"") {
			t.Error("Should preserve 8-space indentation")
		}

		if !strings.Contains(resultStr, "    echo \"loop $i\"") {
			t.Error("Should preserve indentation after marker removal")
		}
	})

	t.Run("handles consecutive newlines correctly", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh

echo "line1"


echo "line2"

` + MarkerStart + `
sync
` + MarkerEnd + `

echo "line3"


echo "line4"`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		resultStr := string(result)

		// Marker section should be gone
		if strings.Contains(resultStr, MarkerStart) {
			t.Error("Marker should be removed")
		}

		// Content should be preserved (trimmed due to TrimSpace in Uninstall)
		if !strings.Contains(resultStr, "line1") || !strings.Contains(resultStr, "line4") {
			t.Error("All content lines should be preserved")
		}
	})
}

// TestInstallStringHandling tests string handling during install
func TestInstallStringHandling(t *testing.T) {
	t.Run("appends with proper spacing", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Existing hook without trailing newline
		existing := `#!/bin/sh
echo "existing hook"`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(existing), 0755)

		err := Install(dir)
		if err != nil {
			t.Fatalf("Install failed: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		resultStr := string(result)

		// Check for proper separation
		lines := strings.Split(resultStr, "\n")

		// Should have existing content
		if !strings.Contains(resultStr, "existing hook") {
			t.Error("Should preserve existing hook")
		}

		// Should have our marker
		if !strings.Contains(resultStr, MarkerStart) {
			t.Error("Should add our marker")
		}

		// Check there's reasonable spacing (at least one blank line between)
		existingIdx := -1
		markerIdx := -1
		for i, line := range lines {
			if strings.Contains(line, "existing hook") {
				existingIdx = i
			}
			if strings.Contains(line, MarkerStart) {
				markerIdx = i
			}
		}

		if markerIdx <= existingIdx {
			t.Error("Our hook should come after existing hook")
		}

		if markerIdx-existingIdx < 2 {
			t.Error("Should have at least one blank line between hooks")
		}
	})

	t.Run("preserves existing hook format", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		existing := `#!/bin/bash
# Custom hook
# Multiple comments

function my_hook() {
    echo "custom logic"
}

my_hook
`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(existing), 0755)

		err := Install(dir)
		if err != nil {
			t.Fatalf("Install failed: %v", err)
		}

		result, _ := os.ReadFile(hookPath)
		resultStr := string(result)

		// All original content should be preserved exactly
		if !strings.Contains(resultStr, "#!/bin/bash") {
			t.Error("Should preserve bash shebang")
		}

		if !strings.Contains(resultStr, "# Custom hook") {
			t.Error("Should preserve comments")
		}

		if !strings.Contains(resultStr, "function my_hook()") {
			t.Error("Should preserve function definition")
		}

		if !strings.Contains(resultStr, "my_hook") {
			t.Error("Should preserve function call")
		}

		// Our hook should be appended
		if !strings.Contains(resultStr, MarkerStart) {
			t.Error("Should append our hook")
		}
	})
}
