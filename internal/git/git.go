// Package git provides git command wrappers
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
// This allows the subclone's files to be tracked by the parent repo
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
	if _, err := f.WriteString("\n# git-subclone ignore\n"); err != nil {
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

// RemoveIgnorePatternsFromGitignore removes git-subclone ignore section from .gitignore
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
		if trimmed == "# git-subclone ignore" {
			inIgnoreSection = true
			continue
		}

		// End of ignore section (empty line or next section)
		if inIgnoreSection {
			if trimmed == "" || (strings.HasPrefix(trimmed, "#") && !strings.Contains(trimmed, "git-subclone")) {
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

// InitRepo initializes a git repo and adds remote
func InitRepo(path, repo, branch string) error {
	// git init
	cmd := exec.Command("git", "-C", path, "init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// git remote add origin
	cmd = exec.Command("git", "-C", path, "remote", "add", "origin", repo)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// git fetch
	cmd = exec.Command("git", "-C", path, "fetch", "origin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Set upstream branch
	targetBranch := branch
	if targetBranch == "" {
		targetBranch = "main"
	}

	cmd = exec.Command("git", "-C", path, "branch", "--set-upstream-to=origin/"+targetBranch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // Ignore error if branch doesn't exist yet

	return nil
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

	for _, file := range files {
		// Check if file exists
		fullPath := filepath.Join(repoPath, file)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// File doesn't exist, skip
			continue
		}

		// Check if already skip-worktree
		cmd := exec.Command("git", "-C", repoPath, "ls-files", "-v", file)
		out, err := cmd.Output()
		if err != nil {
			continue
		}

		// If starts with 'S', already skip-worktree
		if len(out) > 0 && out[0] == 'S' {
			continue
		}

		// Apply skip-worktree
		cmd = exec.Command("git", "-C", repoPath, "update-index", "--skip-worktree", file)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to apply skip-worktree to %s: %v\n", file, err)
		}
	}

	return nil
}

// UnapplySkipWorktree removes skip-worktree from files
func UnapplySkipWorktree(repoPath string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	for _, file := range files {
		cmd := exec.Command("git", "-C", repoPath, "update-index", "--no-skip-worktree", file)
		if err := cmd.Run(); err != nil {
			// Ignore errors (file might not be tracked)
			continue
		}
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
