// Package hooks handles git hooks installation
package hooks

import (
	"os"
	"path/filepath"
	"strings"
)

// MarkerStart is the start marker for git-multirepo hooks
const MarkerStart = "# === git-multirepo hook START ==="

// MarkerEnd is the end marker for git-multirepo hooks
const MarkerEnd = "# === git-multirepo hook END ==="

const hookMarkerStart = MarkerStart
const hookMarkerEnd = MarkerEnd

const postCheckoutHook = `#!/bin/sh
` + hookMarkerStart + `
# git-multirepo post-checkout hook
# Automatically syncs subs after checkout
# Runs from current directory (respects hierarchy)

if command -v git-multirepo >/dev/null 2>&1; then
    cd "$(pwd)" && git-multirepo sync
fi
` + hookMarkerEnd

const postCommitHook = `#!/bin/sh
# git-multirepo post-commit hook for sub repositories
# Automatically updates parent's .git.multirepos after commit

# Find parent repository (look for .git.multirepos)
find_parent() {
    local dir="$1"
    while [ "$dir" != "/" ] && [ "$dir" != "." ]; do
        dir=$(dirname "$dir")
        if [ -f "$dir/.git.multirepos" ]; then
            echo "$dir"
            return 0
        fi
    done
    return 1
}

# Get current repository root
SUB_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
if [ -z "$SUB_ROOT" ]; then
    exit 0
fi

# Find parent repository
PARENT_ROOT=$(find_parent "$SUB_ROOT")
if [ -z "$PARENT_ROOT" ]; then
    # Not a sub repository, exit silently
    exit 0
fi

# Check if git-multirepo is available
if ! command -v git-multirepo >/dev/null 2>&1; then
    exit 0
fi

# Update parent's .git.multirepos
cd "$PARENT_ROOT" && git-multirepo sync 2>/dev/null || true
`

// Install installs git hooks in the repository
// Merges with existing hooks instead of overwriting
func Install(repoRoot string) error {
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")

	// Ensure hooks directory exists
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	hookPath := filepath.Join(hooksDir, "post-checkout")

	// Read existing hook if exists
	existingContent := ""
	if content, err := os.ReadFile(hookPath); err == nil {
		existingContent = string(content)

		// Check if our hook is already installed
		if strings.Contains(existingContent, hookMarkerStart) {
			return nil // Already installed
		}
	}

	// Merge: existing + our hook
	var newContent string
	if existingContent == "" {
		newContent = postCheckoutHook
	} else {
		// Append our hook to existing
		newContent = existingContent
		if !strings.HasSuffix(newContent, "\n") {
			newContent += "\n"
		}
		newContent += "\n" + postCheckoutHook
	}

	return os.WriteFile(hookPath, []byte(newContent), 0755)
}

// Uninstall removes only our hook from the repository
// If other hooks exist, they are preserved
func Uninstall(repoRoot string) error {
	hookPath := filepath.Join(repoRoot, ".git", "hooks", "post-checkout")

	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Hook doesn't exist, nothing to do
		}
		return err
	}

	strContent := string(content)

	// Find and remove our section
	startIdx := strings.Index(strContent, hookMarkerStart)
	endIdx := strings.Index(strContent, hookMarkerEnd)

	if startIdx == -1 || endIdx == -1 {
		return nil // Our hook not found
	}

	// Remove our section (including markers and trailing newline)
	before := strContent[:startIdx]
	after := strContent[endIdx+len(hookMarkerEnd):]

	// Remove trailing newline after marker if present
	if strings.HasPrefix(after, "\n") {
		after = after[1:]
	}

	newContent := strings.TrimSpace(before + after)

	// If only shebang left or empty, delete file
	if newContent == "" || newContent == "#!/bin/sh" {
		return os.Remove(hookPath)
	}

	// Write back remaining content
	return os.WriteFile(hookPath, []byte(newContent+"\n"), 0755)
}

// IsInstalled checks if our specific hook is installed
func IsInstalled(repoRoot string) bool {
	hookPath := filepath.Join(repoRoot, ".git", "hooks", "post-checkout")
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), hookMarkerStart)
}

// HasHook checks if ANY hook exists (for status differentiation)
func HasHook(repoRoot string) bool {
	hookPath := filepath.Join(repoRoot, ".git", "hooks", "post-checkout")
	_, err := os.Stat(hookPath)
	return err == nil
}

// InstallWorkspaceHook installs post-commit hook in a workspace repository
func InstallWorkspaceHook(workspacePath string) error {
	hooksDir := filepath.Join(workspacePath, ".git", "hooks")

	// Ensure hooks directory exists
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	// Install post-commit hook
	hookPath := filepath.Join(hooksDir, "post-commit")
	return os.WriteFile(hookPath, []byte(postCommitHook), 0755)
}

// IsWorkspaceHookInstalled checks if the workspace hook is installed
func IsWorkspaceHookInstalled(workspacePath string) bool {
	hookPath := filepath.Join(workspacePath, ".git", "hooks", "post-commit")
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return string(content) == postCommitHook
}
