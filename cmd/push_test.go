package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yejune/git-workspace/internal/config"
	"github.com/yejune/git-workspace/internal/git"
)

// ============================================================================
// Helper Functions
// ============================================================================

// setupPushTestEnv creates a test environment with config
func setupPushTestEnv(t *testing.T, org, prefix, suffix string) (string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".git.workspaces")

	// Set up config
	if org != "" {
		cmd := exec.Command("git", "config", "-f", configPath,
			"workspace.organization", org)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set organization: %v", err)
		}
	}

	if prefix != "" {
		cmd := exec.Command("git", "config", "-f", configPath,
			"workspace.stripPrefix", prefix)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set stripPrefix: %v", err)
		}
	}

	if suffix != "" {
		cmd := exec.Command("git", "config", "-f", configPath,
			"workspace.stripSuffix", suffix)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set stripSuffix: %v", err)
		}
	}

	// Override HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	cleanup := func() {
		os.Setenv("HOME", oldHome)
	}

	return tempDir, cleanup
}

// setupGitRepo creates a simple git repository for testing
func setupGitRepo(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Initialize git repo
	exec.Command("git", "-C", path, "init").Run()
	exec.Command("git", "-C", path, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", path, "config", "user.name", "Test User").Run()

	// Create initial commit
	readme := filepath.Join(path, "README.md")
	os.WriteFile(readme, []byte("# Test Repo"), 0644)
	exec.Command("git", "-C", path, "add", ".").Run()
	exec.Command("git", "-C", path, "commit", "-m", "Initial commit").Run()
}

// ============================================================================
// Test Cases: shouldEnablePushCommand
// ============================================================================

func TestShouldEnablePushCommand_WithConfig(t *testing.T) {
	t.Run("config exists with organization", func(t *testing.T) {
		_, cleanup := setupPushTestEnv(t, "https://github.com/test-org", "", "")
		defer cleanup()

		if !shouldEnablePushCommand() {
			t.Error("shouldEnablePushCommand() should return true when config exists with organization")
		}
	})
}

func TestShouldEnablePushCommand_WithoutConfig(t *testing.T) {
	t.Run("config file does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", oldHome)

		if shouldEnablePushCommand() {
			t.Error("shouldEnablePushCommand() should return false when config does not exist")
		}
	})
}

func TestShouldEnablePushCommand_NoOrganization(t *testing.T) {
	t.Run("config exists but no organization", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, ".git.workspaces")

		// Create empty config file
		os.WriteFile(configPath, []byte(""), 0644)

		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", oldHome)

		if shouldEnablePushCommand() {
			t.Error("shouldEnablePushCommand() should return false when organization is not set")
		}
	})
}

// ============================================================================
// Test Cases: determineWorkspacePath
// ============================================================================

func TestDetermineWorkspacePath_FromArgs(t *testing.T) {
	t.Run("absolute path from args", func(t *testing.T) {
		testPath := "/tmp/test-workspace"
		path, err := determineWorkspacePath([]string{testPath})
		if err != nil {
			t.Fatalf("determineWorkspacePath() unexpected error: %v", err)
		}
		if path != testPath {
			t.Errorf("determineWorkspacePath() = %v, want %v", path, testPath)
		}
	})

	t.Run("relative path from args", func(t *testing.T) {
		cwd, _ := os.Getwd()
		relativePath := "test-workspace"
		expectedPath := filepath.Join(cwd, relativePath)

		path, err := determineWorkspacePath([]string{relativePath})
		if err != nil {
			t.Fatalf("determineWorkspacePath() unexpected error: %v", err)
		}
		if path != expectedPath {
			t.Errorf("determineWorkspacePath() = %v, want %v", path, expectedPath)
		}
	})
}

func TestDetermineWorkspacePath_FromCurrentDir(t *testing.T) {
	t.Run("no args uses current directory", func(t *testing.T) {
		cwd, _ := os.Getwd()

		path, err := determineWorkspacePath([]string{})
		if err != nil {
			t.Fatalf("determineWorkspacePath() unexpected error: %v", err)
		}
		if path != cwd {
			t.Errorf("determineWorkspacePath() = %v, want %v", path, cwd)
		}
	})
}

func TestDetermineWorkspacePath_NotGitRepo(t *testing.T) {
	t.Run("path is not a git repository", func(t *testing.T) {
		tempDir := t.TempDir()

		// Test that we can determine the path even if it's not a git repo
		// (the git repo check happens later in runPush)
		path, err := determineWorkspacePath([]string{tempDir})
		if err != nil {
			t.Fatalf("determineWorkspacePath() unexpected error: %v", err)
		}
		if path != tempDir {
			t.Errorf("determineWorkspacePath() = %v, want %v", path, tempDir)
		}
	})
}

// ============================================================================
// Test Cases: setupRemote
// ============================================================================

func TestSetupRemote_AddNewRemote(t *testing.T) {
	t.Run("add new remote when none exists", func(t *testing.T) {
		tempDir := t.TempDir()
		repoPath := filepath.Join(tempDir, "test-repo")
		setupGitRepo(t, repoPath)

		repoURL := "https://github.com/test-org/test-repo.git"
		err := setupRemote(repoPath, repoURL)
		if err != nil {
			t.Fatalf("setupRemote() unexpected error: %v", err)
		}

		// Verify remote was added
		remoteURL, err := git.GetRemoteURL(repoPath)
		if err != nil {
			t.Fatalf("Failed to get remote URL: %v", err)
		}
		if remoteURL != repoURL {
			t.Errorf("Remote URL = %v, want %v", remoteURL, repoURL)
		}
	})
}

func TestSetupRemote_ExistingSameURL(t *testing.T) {
	t.Run("remote exists with same URL", func(t *testing.T) {
		tempDir := t.TempDir()
		repoPath := filepath.Join(tempDir, "test-repo")
		setupGitRepo(t, repoPath)

		repoURL := "https://github.com/test-org/test-repo.git"

		// Add remote first
		exec.Command("git", "-C", repoPath, "remote", "add", "origin", repoURL).Run()

		// Try to setup again - should succeed
		err := setupRemote(repoPath, repoURL)
		if err != nil {
			t.Fatalf("setupRemote() unexpected error: %v", err)
		}
	})
}

func TestSetupRemote_ExistingDifferentURL(t *testing.T) {
	t.Run("remote exists with different URL", func(t *testing.T) {
		tempDir := t.TempDir()
		repoPath := filepath.Join(tempDir, "test-repo")
		setupGitRepo(t, repoPath)

		existingURL := "https://github.com/old-org/old-repo.git"
		newURL := "https://github.com/test-org/test-repo.git"

		// Add remote with different URL
		exec.Command("git", "-C", repoPath, "remote", "add", "origin", existingURL).Run()

		// Try to setup - should fail with clear error
		err := setupRemote(repoPath, newURL)
		if err == nil {
			t.Error("setupRemote() expected error when remote exists with different URL")
		}
		if !strings.Contains(err.Error(), "different URL") {
			t.Errorf("setupRemote() error should mention 'different URL', got: %v", err)
		}
		if !strings.Contains(err.Error(), existingURL) {
			t.Errorf("setupRemote() error should show existing URL, got: %v", err)
		}
		if !strings.Contains(err.Error(), newURL) {
			t.Errorf("setupRemote() error should show expected URL, got: %v", err)
		}
	})
}

// ============================================================================
// Test Cases: runPush - Basic Validation
// ============================================================================

func TestRunPush_NotGitRepo(t *testing.T) {
	t.Run("error when path is not a git repository", func(t *testing.T) {
		_, cleanup := setupPushTestEnv(t, "https://github.com/test-org", "", "")
		defer cleanup()

		tempDir := t.TempDir()

		// Save and restore working directory
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)
		os.Chdir(tempDir)

		err := runPush(pushCmd, []string{})
		if err == nil {
			t.Error("runPush() expected error when not in git repository")
		}
		if !strings.Contains(err.Error(), "not a git repository") {
			t.Errorf("runPush() error should mention 'not a git repository', got: %v", err)
		}
	})
}

func TestRunPush_NoOrganizationConfigured(t *testing.T) {
	t.Run("error when organization not configured", func(t *testing.T) {
		tempDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", oldHome)

		repoPath := filepath.Join(tempDir, "test-repo")
		setupGitRepo(t, repoPath)

		err := runPush(pushCmd, []string{repoPath})
		if err == nil {
			t.Error("runPush() expected error when organization not configured")
		}
		if !strings.Contains(err.Error(), "organization not configured") {
			t.Errorf("runPush() error should mention organization, got: %v", err)
		}
	})
}

// ============================================================================
// Test Cases: Repository Name Normalization
// ============================================================================

func TestRunPush_RepoNameNormalization(t *testing.T) {
	tests := []struct {
		name       string
		folderName string
		prefix     string
		suffix     string
		wantName   string
	}{
		{
			name:       "strip workspace suffix",
			folderName: "my-project.workspace",
			prefix:     "",
			suffix:     ".workspace",
			wantName:   "my-project",
		},
		{
			name:       "strip tmp prefix",
			folderName: "tmp-my-project",
			prefix:     "tmp-",
			suffix:     "",
			wantName:   "my-project",
		},
		{
			name:       "strip both prefix and suffix",
			folderName: "tmp-my-project.workspace",
			prefix:     "tmp-",
			suffix:     ".workspace",
			wantName:   "my-project",
		},
		{
			name:       "no stripping",
			folderName: "my-project",
			prefix:     "",
			suffix:     "",
			wantName:   "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupPushTestEnv(t, "https://github.com/test-org", tt.prefix, tt.suffix)
			defer cleanup()

			// Test NormalizeRepoName directly
			normalized, err := config.NormalizeRepoName(tt.folderName)
			if err != nil {
				t.Fatalf("NormalizeRepoName() unexpected error: %v", err)
			}
			if normalized != tt.wantName {
				t.Errorf("NormalizeRepoName() = %v, want %v", normalized, tt.wantName)
			}
		})
	}
}

// ============================================================================
// Test Cases: Integration Tests (require mocking or real API)
// ============================================================================

func TestRunPush_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("full push workflow", func(t *testing.T) {
		// This test would require:
		// 1. Valid GitHub authentication
		// 2. Mocking GitHub API calls
		// 3. Interactive prompt mocking
		// Skip for now as it requires extensive setup
		t.Skip("Requires GitHub API mocking or real credentials")
	})
}

// ============================================================================
// Test Cases: Error Handling
// ============================================================================

func TestRunPush_EmptyRepoName(t *testing.T) {
	t.Run("error when normalized repo name is empty", func(t *testing.T) {
		_, cleanup := setupPushTestEnv(t, "https://github.com/test-org", "", "")
		defer cleanup()

		tempDir := t.TempDir()
		repoPath := filepath.Join(tempDir, "")
		setupGitRepo(t, repoPath)

		// This test would need to trigger the empty name check
		// which happens after normalization - requires more complex setup
		t.Skip("Requires interactive prompt mocking")
	})
}

// ============================================================================
// Test Cases: Git Operations
// ============================================================================

func TestGitOperations(t *testing.T) {
	t.Run("verify git remote operations", func(t *testing.T) {
		tempDir := t.TempDir()
		repoPath := filepath.Join(tempDir, "test-repo")
		setupGitRepo(t, repoPath)

		// Test adding remote
		testURL := "https://github.com/test-org/test-repo.git"
		cmd := exec.Command("git", "-C", repoPath, "remote", "add", "origin", testURL)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to add remote: %v", err)
		}

		// Verify remote was added
		remoteURL, err := git.GetRemoteURL(repoPath)
		if err != nil {
			t.Fatalf("Failed to get remote URL: %v", err)
		}
		if remoteURL != testURL {
			t.Errorf("Remote URL = %v, want %v", remoteURL, testURL)
		}
	})
}
