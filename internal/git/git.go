// Package git provides git command wrappers
package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Clone clones a repository to the specified path
func Clone(repo, path, branch string) error {
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, repo, path)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// InitRepo initializes a git repository in an existing directory with source files
// This is used when source files are already tracked by parent but .git is missing
func InitRepo(path, repo, branch string) error {
	// Create a temporary directory for bare clone
	tempDir, err := os.MkdirTemp("", "git-multirepo-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone as bare to temp location (only .git contents)
	tempGit := filepath.Join(tempDir, "temp.git")
	args := []string{"clone", "--bare"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, repo, tempGit)

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone: %w", err)
	}

	// Move .git directory to target path
	targetGit := filepath.Join(path, ".git")
	if err := os.Rename(tempGit, targetGit); err != nil {
		return fmt.Errorf("failed to move .git: %w", err)
	}

	// Convert from bare to normal repository
	cmd = exec.Command("git", "-C", path, "config", "--bool", "core.bare", "false")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure: %w", err)
	}

	// Reset index to match HEAD (don't touch working tree files)
	cmd = exec.Command("git", "-C", path, "reset", "--mixed", "HEAD")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reset: %w", err)
	}

	return nil
}

// Pull pulls the latest changes in the specified directory
func Pull(path string) error {
	cmd := exec.Command("git", "-C", path, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Push pushes changes in the specified directory
func Push(path string) error {
	cmd := exec.Command("git", "-C", path, "push")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// IsRepo checks if the given path is a git repository
func IsRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetRepoRoot returns the root directory of the git repository
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

// AddToGitignore adds a path's .git directory to .gitignore
// This allows the workspace's files to be tracked by the parent repo
// while keeping the nested .git separate
func AddToGitignore(repoRoot, path string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	// Read existing content
	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Only ignore the .git directory, not the files
	entry := path + "/.git/"

	// Check if already ignored
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return nil // Already exists
		}
	}

	// Build the string to append
	var toWrite string
	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		toWrite = "\n" + entry + "\n"
	} else {
		toWrite = entry + "\n"
	}

	// Append new entry
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(toWrite)
	return err
}

// AddIgnorePatternsToGitignore adds multiple patterns to .gitignore
func AddIgnorePatternsToGitignore(repoRoot string, patterns []string) error {
	if len(patterns) == 0 {
		return nil
	}

	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	// Read existing content
	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	existingLines := make(map[string]bool)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		existingLines[strings.TrimSpace(line)] = true
	}

	// Collect new patterns
	var newPatterns []string
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}
		if !existingLines[pattern] {
			newPatterns = append(newPatterns, pattern)
		}
	}

	if len(newPatterns) == 0 {
		return nil // All patterns already exist
	}

	// Append new patterns
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	// Add header comment
	if _, err := f.WriteString("\n# git-multirepo ignore\n"); err != nil {
		return err
	}

	// Add patterns
	for _, pattern := range newPatterns {
		if _, err := f.WriteString(pattern + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// RemoveIgnorePatternsFromGitignore removes git-multirepo ignore section from .gitignore
func RemoveIgnorePatternsFromGitignore(repoRoot string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	inIgnoreSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of ignore section
		if trimmed == "# git-multirepo ignore" {
			inIgnoreSection = true
			continue
		}

		// End of ignore section (empty line or next section)
		if inIgnoreSection {
			if trimmed == "" || (strings.HasPrefix(trimmed, "#") && !strings.Contains(trimmed, "git-multirepo")) {
				inIgnoreSection = false
			} else {
				// Skip lines in ignore section
				continue
			}
		}

		newLines = append(newLines, line)
	}

	return os.WriteFile(gitignorePath, []byte(strings.Join(newLines, "\n")), 0644)
}

// RemoveFromGitignore removes a path's .git directory from .gitignore
func RemoveFromGitignore(repoRoot, path string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(content), "\n")
	entry := path + "/.git/"
	var newLines []string

	for _, line := range lines {
		if strings.TrimSpace(line) != entry {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(gitignorePath, []byte(strings.Join(newLines, "\n")), 0644)
}


// HasChanges checks if there are uncommitted changes
func HasChanges(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetCurrentCommit returns the current HEAD commit hash
func GetCurrentCommit(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// HasUnpushedCommits checks if there are commits not pushed to remote
func HasUnpushedCommits(path string) (bool, error) {
	// Get current branch
	cmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	branch := strings.TrimSpace(string(out))

	// Check if branch has upstream
	cmd = exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	if err := cmd.Run(); err != nil {
		// No upstream configured - consider as unpushed
		return true, nil
	}

	// Compare with upstream
	cmd = exec.Command("git", "-C", path, "rev-list", "--count", branch+"@{upstream}.."+branch)
	out, err = cmd.Output()
	if err != nil {
		return false, err
	}

	count := strings.TrimSpace(string(out))
	return count != "0", nil
}

// GetRemoteURL returns the remote origin URL
func GetRemoteURL(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ApplySkipWorktree applies skip-worktree to files
func ApplySkipWorktree(repoPath string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	var failed []string

	for _, file := range files {
		// Check if already skip-worktree
		cmd := exec.Command("git", "-C", repoPath, "ls-files", "-v", file)
		out, err := cmd.Output()
		if err == nil && len(out) > 0 && out[0] == 'S' {
			// Already skip-worktree, skip
			continue
		}

		// Apply skip-worktree - let git tell us if file doesn't exist or isn't tracked
		cmd = exec.Command("git", "-C", repoPath, "update-index", "--skip-worktree", file)
		if err := cmd.Run(); err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", file, err))
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to apply skip-worktree to %d file(s):\n  - %s", len(failed), strings.Join(failed, "\n  - "))
	}

	return nil
}

// UnapplySkipWorktree removes skip-worktree from files
func UnapplySkipWorktree(repoPath string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	var failed []string

	for _, file := range files {
		cmd := exec.Command("git", "-C", repoPath, "update-index", "--no-skip-worktree", file)
		if err := cmd.Run(); err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", file, err))
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to unapply skip-worktree from %d file(s):\n  - %s", len(failed), strings.Join(failed, "\n  - "))
	}

	return nil
}

// ListSkipWorktree lists all files with skip-worktree set
func ListSkipWorktree(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "ls-files", "-v")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if len(line) > 2 && line[0] == 'S' {
			// Format: "S filename"
			files = append(files, strings.TrimSpace(line[2:]))
		}
	}

	return files, nil
}

// HasLocalChanges checks if there are uncommitted changes (including untracked files)
func HasLocalChanges(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

// CountChangedFiles counts the number of changed files
func CountChangedFiles(path string) (int, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0, nil
	}
	return len(lines), nil
}

// Stash stashes all local changes
func Stash(path string) error {
	cmd := exec.Command("git", "-C", path, "stash", "push", "-m", "git-multirepo auto-stash")
	return cmd.Run()
}

// StashPop applies and removes the most recent stash
func StashPop(path string) error {
	cmd := exec.Command("git", "-C", path, "stash", "pop")
	return cmd.Run()
}

// GetModifiedFiles returns list of modified files
func GetModifiedFiles(path string) ([]string, error) {
	cmd := exec.Command("git", "-C", path, "diff", "--name-only", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}
	return files, nil
}

// GetUntrackedFiles returns list of untracked files
func GetUntrackedFiles(path string) ([]string, error) {
	cmd := exec.Command("git", "-C", path, "ls-files", "--others", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}
	return files, nil
}

// GetStagedFiles returns list of staged files
func GetStagedFiles(path string) ([]string, error) {
	cmd := exec.Command("git", "-C", path, "diff", "--name-only", "--cached")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}
	return files, nil
}

// FetchTimeout is the timeout for fetch operations
const FetchTimeout = 10 * time.Second

// ErrFetchTimeout indicates fetch operation timed out
var ErrFetchTimeout = fmt.Errorf("fetch timed out after %v", FetchTimeout)

// Fetch fetches from remote with timeout
func Fetch(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), FetchTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", path, "fetch", "origin")
	cmd.Stderr = nil // Suppress stderr
	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return ErrFetchTimeout
	}
	return err
}

// GetBehindCount returns number of commits behind remote
func GetBehindCount(path, branch string) (int, error) {
	// Check if remote branch exists
	cmd := exec.Command("git", "-C", path, "rev-parse", "--verify", "origin/"+branch)
	if err := cmd.Run(); err != nil {
		return 0, nil // Remote branch doesn't exist
	}

	cmd = exec.Command("git", "-C", path, "rev-list", "--count", branch+"..origin/"+branch)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var count int
	_, err = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &count)
	return count, err
}

// GetAheadCount returns number of commits ahead of remote
func GetAheadCount(path, branch string) (int, error) {
	// Check if remote branch exists
	cmd := exec.Command("git", "-C", path, "rev-parse", "--verify", "origin/"+branch)
	if err := cmd.Run(); err != nil {
		return 0, nil // Remote branch doesn't exist
	}

	cmd = exec.Command("git", "-C", path, "rev-list", "--count", "origin/"+branch+".."+branch)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var count int
	_, err = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &count)
	return count, err
}

// GetSkipFileRemoteChanges returns diff of skip-worktree file between local and remote
func GetSkipFileRemoteChanges(path, file string) (string, error) {
	branch, err := GetCurrentBranch(path)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", "-C", path, "diff", "HEAD", "origin/"+branch, "--", file)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// HasRemoteChanges checks if a file has changes between HEAD and remote
func HasRemoteChanges(path, file, branch string) (bool, error) {
	// Check if remote branch exists
	cmd := exec.Command("git", "-C", path, "rev-parse", "--verify", "origin/"+branch)
	if err := cmd.Run(); err != nil {
		return false, nil // Remote branch doesn't exist
	}

	// Check for differences
	cmd = exec.Command("git", "-C", path, "diff", "--quiet", "HEAD", "origin/"+branch, "--", file)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return true, nil // Differences found
		}
		return false, err // Other error
	}
	return false, nil // No differences
}

// GetFileDiff returns the diff of a file between HEAD and remote
func GetFileDiff(path, file, branch string) (string, error) {
	cmd := exec.Command("git", "-C", path, "diff", "HEAD", "origin/"+branch, "--", file)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ResetFile resets a file to match remote version
func ResetFile(path, file, branch string) error {
	cmd := exec.Command("git", "-C", path, "checkout", "origin/"+branch, "--", file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
