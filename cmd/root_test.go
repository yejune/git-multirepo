package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL with .git",
			url:      "https://github.com/user/repo.git",
			expected: "repo",
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/user/repo",
			expected: "repo",
		},
		{
			name:     "SSH URL with .git",
			url:      "git@github.com:user/repo.git",
			expected: "repo",
		},
		{
			name:     "SSH URL without .git",
			url:      "git@github.com:user/repo",
			expected: "repo",
		},
		{
			name:     "GitLab HTTPS",
			url:      "https://gitlab.com/group/subgroup/project.git",
			expected: "project",
		},
		{
			name:     "GitLab SSH",
			url:      "git@gitlab.com:group/subgroup/project.git",
			expected: "project",
		},
		{
			name:     "Simple name",
			url:      "myrepo",
			expected: "myrepo",
		},
		{
			name:     "Local path",
			url:      "/path/to/repo",
			expected: "repo",
		},
		{
			name:     "Bitbucket SSH",
			url:      "git@bitbucket.org:team/repo.git",
			expected: "repo",
		},
		{
			name:     "SSH with nested path",
			url:      "git@github.com:org/team/repo.git",
			expected: "repo",
		},
		{
			name:     "SSH colon only",
			url:      "host:repo",
			expected: "repo",
		},
		{
			name:     "Empty string",
			url:      "",
			expected: "",
		},
		{
			name:     "Just .git",
			url:      ".git",
			expected: "",
		},
		{
			name:     "Trailing slash",
			url:      "https://github.com/user/repo/",
			expected: "",
		},
		{
			name:     "Multiple colons SSH",
			url:      "git@host:org:team:repo",
			expected: "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRepoName(tt.url)
			if result != tt.expected {
				t.Errorf("extractRepoName(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestExecuteWithMock(t *testing.T) {
	// Save original osExit
	originalOsExit := osExit
	defer func() { osExit = originalOsExit }()

	t.Run("Execute success in git repo", func(t *testing.T) {
		// Create temp git repo
		dir := t.TempDir()
		exec.Command("git", "-C", dir, "init").Run()
		exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
		exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()

		// Create initial commit
		readme := filepath.Join(dir, "README.md")
		os.WriteFile(readme, []byte("# Test"), 0644)
		exec.Command("git", "-C", dir, "add", ".").Run()
		exec.Command("git", "-C", dir, "commit", "-m", "Initial commit").Run()

		// Change to test directory
		originalDir, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(originalDir)

		// Mock osExit to track if it was called
		exitCalled := false
		exitCode := 0
		osExit = func(code int) {
			exitCalled = true
			exitCode = code
		}

		// Set args to show help (no URL)
		rootCmd.SetArgs([]string{})

		Execute()

		if exitCalled {
			t.Errorf("Execute() called os.Exit with code %d, want no exit", exitCode)
		}
	})

	t.Run("Execute error exits with code 1", func(t *testing.T) {
		// Create temp non-git directory
		dir := t.TempDir()

		// Change to test directory
		originalDir, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(originalDir)

		// Mock osExit
		exitCalled := false
		exitCode := 0
		osExit = func(code int) {
			exitCalled = true
			exitCode = code
		}

		// Set args to trigger an error (trying to list in non-git repo)
		rootCmd.SetArgs([]string{"list"})

		Execute()

		if !exitCalled {
			t.Error("Execute() should have called os.Exit")
		}
		if exitCode != 1 {
			t.Errorf("Execute() exit code = %d, want 1", exitCode)
		}
	})
}
