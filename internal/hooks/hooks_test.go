package hooks

import (
	"os"
	"path/filepath"
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
		if string(content) != postCheckoutHook {
			t.Error("hook content mismatch")
		}

		// Check executable
		info, _ := os.Stat(hookPath)
		if info.Mode().Perm()&0111 == 0 {
			t.Error("hook should be executable")
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

	t.Run("uninstall non-existent hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0755)

		err := Uninstall(dir)
		if err != nil {
			t.Errorf("should not error on non-existent hook: %v", err)
		}
	})

	t.Run("preserve custom hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Write custom hook
		hookPath := filepath.Join(hooksDir, "post-checkout")
		customContent := "#!/bin/sh\necho custom hook"
		os.WriteFile(hookPath, []byte(customContent), 0755)

		err := Uninstall(dir)
		if err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}

		// Custom hook should remain
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatal("custom hook should still exist")
		}
		if string(content) != customContent {
			t.Error("custom hook content should be preserved")
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
		os.Chmod(hooksDir, 0555)
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
}
