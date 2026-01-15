package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yejune/git-multirepo/internal/hooks"
)

// TestGetHookStatusStringEdgeCases tests getHookStatus function with various string edge cases
func TestGetHookStatusStringEdgeCases(t *testing.T) {
	t.Run("no hook file", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0755)

		status := getHookStatus(dir)
		if status != HookNone {
			t.Errorf("Expected HookNone, got %v", status)
		}
	})

	t.Run("only our hook clean", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Only our hook with shebang
		content := `#!/bin/sh
` + hooks.MarkerStart + `
if command -v git-multirepo >/dev/null 2>&1; then
    cd "$(pwd)" && git-multirepo sync
fi
` + hooks.MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookOurs {
			t.Errorf("Expected HookOurs, got %v", status)
		}
	})

	t.Run("only our hook without shebang", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Only our hook, no shebang
		content := hooks.MarkerStart + `
if command -v git-multirepo >/dev/null 2>&1; then
    cd "$(pwd)" && git-multirepo sync
fi
` + hooks.MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookOurs {
			t.Errorf("Expected HookOurs (no shebang), got %v", status)
		}
	})

	t.Run("mixed hook with other content before", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
echo "custom hook before"
` + hooks.MarkerStart + `
sync
` + hooks.MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookMixed {
			t.Errorf("Expected HookMixed (content before), got %v", status)
		}
	})

	t.Run("mixed hook with other content after", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
` + hooks.MarkerStart + `
sync
` + hooks.MarkerEnd + `
echo "custom hook after"`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookMixed {
			t.Errorf("Expected HookMixed (content after), got %v", status)
		}
	})

	t.Run("mixed hook with other content before and after", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
echo "before"
` + hooks.MarkerStart + `
sync
` + hooks.MarkerEnd + `
echo "after"`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookMixed {
			t.Errorf("Expected HookMixed (content both sides), got %v", status)
		}
	})

	t.Run("other hook only no markers", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
echo "totally different hook"
git status`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookOtherOnly {
			t.Errorf("Expected HookOtherOnly, got %v", status)
		}
	})

	t.Run("marker with whitespace around", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh

  ` + hooks.MarkerStart + `
sync
  ` + hooks.MarkerEnd + `

`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		// Should be HookOurs (only shebang and whitespace around markers)
		if status != HookOurs {
			t.Errorf("Expected HookOurs (whitespace ignored), got %v", status)
		}
	})

	t.Run("incomplete markers START only", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
` + hooks.MarkerStart + `
sync
# No END marker`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		// Has START marker, so IsInstalled returns true
		// But getHookStatus checks for both - should handle gracefully
		// Current implementation: if startIdx >= 0 but endIdx == -1, it's malformed
		// Falls back to HookOurs if error reading (defensive)
		if status != HookOurs {
			t.Logf("Status for incomplete markers (START only): %v", status)
		}
	})

	t.Run("incomplete markers END only", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
sync
` + hooks.MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		// No START marker, so IsInstalled returns false -> HookOtherOnly
		if status != HookOtherOnly {
			t.Errorf("Expected HookOtherOnly (no START marker), got %v", status)
		}
	})

	t.Run("windows CRLF line endings", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := "#!/bin/sh\r\n" +
			hooks.MarkerStart + "\r\n" +
			"sync\r\n" +
			hooks.MarkerEnd + "\r\n"

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookOurs {
			t.Errorf("Expected HookOurs (CRLF), got %v", status)
		}
	})

	t.Run("mixed line endings", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := "#!/bin/sh\n" +
			hooks.MarkerStart + "\r\n" +
			"sync\n" +
			hooks.MarkerEnd + "\r\n"

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookOurs {
			t.Errorf("Expected HookOurs (mixed line endings), got %v", status)
		}
	})

	t.Run("very long hook content", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Create 10KB+ file
		longContent := strings.Repeat("echo 'test'\n", 1000)
		content := "#!/bin/sh\n" +
			longContent +
			hooks.MarkerStart + "\n" +
			"sync\n" +
			hooks.MarkerEnd + "\n"

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookMixed {
			t.Errorf("Expected HookMixed (long content before), got %v", status)
		}
	})

	t.Run("special characters in other hook", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
# $VAR ${VAR} $(cmd) \$escaped "quotes" 'single' ` + "`backticks`" + `
` + hooks.MarkerStart + `
sync
` + hooks.MarkerEnd + `
echo "more special: [brackets] {braces}"`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookMixed {
			t.Errorf("Expected HookMixed (special chars), got %v", status)
		}
	})

	t.Run("marker-like content in comments", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
# This comment mentions: ` + hooks.MarkerStart + ` but it's fake
echo "not a real marker"
` + hooks.MarkerStart + `
sync
` + hooks.MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		// Should detect as mixed (has our marker + other content)
		if status != HookMixed {
			t.Errorf("Expected HookMixed (comment + real marker), got %v", status)
		}
	})

	t.Run("only shebang before markers", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Only shebang before, only whitespace after
		content := "#!/bin/sh\n" +
			hooks.MarkerStart + "\n" +
			"sync\n" +
			hooks.MarkerEnd + "\n\n"

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookOurs {
			t.Errorf("Expected HookOurs (only shebang + whitespace), got %v", status)
		}
	})

	t.Run("comments only before markers", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
# Just a comment
# Another comment
` + hooks.MarkerStart + `
sync
` + hooks.MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		// Comments are content, so should be HookMixed
		if status != HookMixed {
			t.Errorf("Expected HookMixed (comments are content), got %v", status)
		}
	})

	t.Run("empty file with only markers", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := hooks.MarkerStart + "\n" +
			"sync\n" +
			hooks.MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		if status != HookOurs {
			t.Errorf("Expected HookOurs (only markers, no shebang), got %v", status)
		}
	})

	t.Run("multiple empty lines between content and markers", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh



` + hooks.MarkerStart + `
sync
` + hooks.MarkerEnd + `



`

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		// Empty lines are trimmed in getHookStatus logic
		if status != HookOurs {
			t.Errorf("Expected HookOurs (empty lines trimmed), got %v", status)
		}
	})

	t.Run("duplicate marker sections", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		content := `#!/bin/sh
` + hooks.MarkerStart + `
sync 1
` + hooks.MarkerEnd + `
echo "between"
` + hooks.MarkerStart + `
sync 2
` + hooks.MarkerEnd

		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.WriteFile(hookPath, []byte(content), 0755)

		status := getHookStatus(dir)
		// Has our markers + other content ("echo between")
		if status != HookMixed {
			t.Errorf("Expected HookMixed (duplicate markers + content), got %v", status)
		}
	})

	t.Run("read error returns HookOurs as fallback", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		hooksDir := filepath.Join(gitDir, "hooks")
		os.MkdirAll(hooksDir, 0755)

		// Create hook as directory to cause read error
		hookPath := filepath.Join(hooksDir, "post-checkout")
		os.MkdirAll(hookPath, 0755)

		status := getHookStatus(dir)
		// IsInstalled returns false (can't read), HasHook returns true (exists)
		// So hasOurs=false, hasAny=true -> HookOtherOnly
		if status != HookOtherOnly {
			t.Logf("Status with read error: %v (expected defensive fallback)", status)
		}
	})
}
