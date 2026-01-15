// Package hooks handles git hooks installation
package hooks

import (
	"os"
	"path/filepath"
)

const postCheckoutHook = `#!/bin/sh
# git-multirepo post-checkout hook
# Automatically syncs subs after checkout

if command -v git-multirepo >/dev/null 2>&1; then
    git-multirepo sync
fi
`

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
// If an existing hook is found that's not ours, it backs it up
func Install(repoRoot string) error {
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")

	// Ensure hooks directory exists
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	hookPath := filepath.Join(hooksDir, "post-checkout")

	// Check if hook already exists
	existingContent, err := os.ReadFile(hookPath)
	if err == nil {
		// Hook exists - check if it's ours
		if string(existingContent) != postCheckoutHook {
			// Different hook - backup first
			backupPath := hookPath + ".bak"
			if err := os.WriteFile(backupPath, existingContent, 0755); err != nil {
				return err
			}
			// Note: caller should display backup warning message
		}
	}

	// Install post-checkout hook (overwrites if exists)
	return os.WriteFile(hookPath, []byte(postCheckoutHook), 0755)
}

// Uninstall removes git hooks from the repository
func Uninstall(repoRoot string) error {
	hookPath := filepath.Join(repoRoot, ".git", "hooks", "post-checkout")

	// Read current hook
	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Only remove if it's our hook
	if string(content) == postCheckoutHook {
		return os.Remove(hookPath)
	}

	return nil
}

// IsInstalled checks if the hook is installed
func IsInstalled(repoRoot string) bool {
	hookPath := filepath.Join(repoRoot, ".git", "hooks", "post-checkout")
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return string(content) == postCheckoutHook
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
