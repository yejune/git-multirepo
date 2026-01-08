// Package hooks handles git hooks installation
package hooks

import (
	"os"
	"path/filepath"
)

const postCheckoutHook = `#!/bin/sh
# git-sub post-checkout hook
# Automatically syncs subs after checkout

if command -v git-sub >/dev/null 2>&1; then
    git-sub sync --recursive
fi
`

const postCommitHook = `#!/bin/sh
# git-sub post-commit hook for sub repositories
# Automatically updates parent's .workspaces after commit

# Find parent repository (look for .workspaces)
find_parent() {
    local dir="$1"
    while [ "$dir" != "/" ] && [ "$dir" != "." ]; do
        dir=$(dirname "$dir")
        if [ -f "$dir/.workspaces" ]; then
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

# Check if git-sub is available
if ! command -v git-sub >/dev/null 2>&1; then
    exit 0
fi

# Get relative path of sub from parent
SUB_PATH=$(realpath --relative-to="$PARENT_ROOT" "$SUB_ROOT" 2>/dev/null || \
           python3 -c "import os.path; print(os.path.relpath('$SUB_ROOT', '$PARENT_ROOT'))" 2>/dev/null)

if [ -z "$SUB_PATH" ]; then
    exit 0
fi

# Update parent's .workspaces (only if pushed)
cd "$PARENT_ROOT" && git-sub sync --update-manifest-only --quiet 2>/dev/null || true
`

// Install installs git hooks in the repository
func Install(repoRoot string) error {
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")

	// Ensure hooks directory exists
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	// Install post-checkout hook
	hookPath := filepath.Join(hooksDir, "post-checkout")
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

// InstallSubHook installs post-commit hook in a sub repository
func InstallSubHook(subPath string) error {
	hooksDir := filepath.Join(subPath, ".git", "hooks")

	// Ensure hooks directory exists
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	// Install post-commit hook
	hookPath := filepath.Join(hooksDir, "post-commit")
	return os.WriteFile(hookPath, []byte(postCommitHook), 0755)
}

// IsSubHookInstalled checks if the sub hook is installed
func IsSubHookInstalled(subPath string) bool {
	hookPath := filepath.Join(subPath, ".git", "hooks", "post-commit")
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return string(content) == postCommitHook
}
