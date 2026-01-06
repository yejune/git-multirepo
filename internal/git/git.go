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
