package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ============================================================================
// Helper Functions
// ============================================================================

// setupTempConfig creates a temporary config file for testing
func setupTempConfig(t *testing.T, org, prefix, suffix string) (string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".git.workspaces")

	// Initialize git config file
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

	// Override HOME to use temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	cleanup := func() {
		os.Setenv("HOME", oldHome)
	}

	return configPath, cleanup
}

// ============================================================================
// Test Cases: ConfigExists
// ============================================================================

func TestConfigExists(t *testing.T) {
	t.Run("config file exists", func(t *testing.T) {
		_, cleanup := setupTempConfig(t, "https://github.com/test-org", "", "")
		defer cleanup()

		if !ConfigExists() {
			t.Error("ConfigExists() should return true when config file exists")
		}
	})

	t.Run("config file does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", oldHome)

		if ConfigExists() {
			t.Error("ConfigExists() should return false when config file does not exist")
		}
	})

	t.Run("home directory error", func(t *testing.T) {
		// This test is tricky - we can't easily force os.UserHomeDir() to fail
		// But we verify it handles the error gracefully by returning false
		// Skip this test as it requires OS-level manipulation
		t.Skip("Cannot reliably test UserHomeDir() error case")
	})
}

// ============================================================================
// Test Cases: GetOrganization
// ============================================================================

func TestGetOrganization(t *testing.T) {
	t.Run("get organization success", func(t *testing.T) {
		expectedOrg := "https://github.com/git-workspaces"
		_, cleanup := setupTempConfig(t, expectedOrg, "", "")
		defer cleanup()

		org, err := GetOrganization()
		if err != nil {
			t.Fatalf("GetOrganization() unexpected error: %v", err)
		}
		if org != expectedOrg {
			t.Errorf("GetOrganization() = %v, want %v", org, expectedOrg)
		}
	})

	t.Run("organization not set", func(t *testing.T) {
		_, cleanup := setupTempConfig(t, "", "", "") // No organization
		defer cleanup()

		_, err := GetOrganization()
		if err == nil {
			t.Error("GetOrganization() expected error when organization not set")
		}
		if err != nil && err.Error() != "organization not configured in ~/.git.workspaces" {
			t.Errorf("GetOrganization() unexpected error message: %v", err)
		}
	})

	t.Run("config file does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", oldHome)

		_, err := GetOrganization()
		if err == nil {
			t.Error("GetOrganization() expected error when config file does not exist")
		}
	})

	t.Run("organization is empty string", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, ".git.workspaces")

		// Create config with empty organization
		cmd := exec.Command("git", "config", "-f", configPath,
			"workspace.organization", "")
		cmd.Run()

		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", oldHome)

		_, err := GetOrganization()
		if err == nil {
			t.Error("GetOrganization() expected error when organization is empty")
		}
	})
}

// ============================================================================
// Test Cases: GetStripPrefix
// ============================================================================

func TestGetStripPrefix(t *testing.T) {
	t.Run("get strip prefix success", func(t *testing.T) {
		expectedPrefix := "tmp-"
		_, cleanup := setupTempConfig(t, "https://github.com/test-org", expectedPrefix, "")
		defer cleanup()

		prefix, err := GetStripPrefix()
		if err != nil {
			t.Fatalf("GetStripPrefix() unexpected error: %v", err)
		}
		if prefix != expectedPrefix {
			t.Errorf("GetStripPrefix() = %v, want %v", prefix, expectedPrefix)
		}
	})

	t.Run("strip prefix not set", func(t *testing.T) {
		_, cleanup := setupTempConfig(t, "https://github.com/test-org", "", "")
		defer cleanup()

		prefix, err := GetStripPrefix()
		if err != nil {
			t.Fatalf("GetStripPrefix() unexpected error: %v", err)
		}
		if prefix != "" {
			t.Errorf("GetStripPrefix() = %v, want empty string", prefix)
		}
	})

	t.Run("config file does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", oldHome)

		prefix, err := GetStripPrefix()
		if err != nil {
			t.Fatalf("GetStripPrefix() unexpected error: %v", err)
		}
		if prefix != "" {
			t.Errorf("GetStripPrefix() should return empty string when config does not exist")
		}
	})
}

// ============================================================================
// Test Cases: GetStripSuffix
// ============================================================================

func TestGetStripSuffix(t *testing.T) {
	t.Run("get strip suffix success", func(t *testing.T) {
		expectedSuffix := ".workspace"
		_, cleanup := setupTempConfig(t, "https://github.com/test-org", "", expectedSuffix)
		defer cleanup()

		suffix, err := GetStripSuffix()
		if err != nil {
			t.Fatalf("GetStripSuffix() unexpected error: %v", err)
		}
		if suffix != expectedSuffix {
			t.Errorf("GetStripSuffix() = %v, want %v", suffix, expectedSuffix)
		}
	})

	t.Run("strip suffix not set", func(t *testing.T) {
		_, cleanup := setupTempConfig(t, "https://github.com/test-org", "", "")
		defer cleanup()

		suffix, err := GetStripSuffix()
		if err != nil {
			t.Fatalf("GetStripSuffix() unexpected error: %v", err)
		}
		if suffix != "" {
			t.Errorf("GetStripSuffix() = %v, want empty string", suffix)
		}
	})

	t.Run("config file does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", oldHome)

		suffix, err := GetStripSuffix()
		if err != nil {
			t.Fatalf("GetStripSuffix() unexpected error: %v", err)
		}
		if suffix != "" {
			t.Errorf("GetStripSuffix() should return empty string when config does not exist")
		}
	})
}

// ============================================================================
// Test Cases: NormalizeRepoName
// ============================================================================

func TestNormalizeRepoName(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		prefix   string
		suffix   string
		want     string
	}{
		{
			name:     "only suffix configured",
			repoName: "my-project.workspace",
			prefix:   "",
			suffix:   ".workspace",
			want:     "my-project",
		},
		{
			name:     "only prefix configured",
			repoName: "tmp-my-project",
			prefix:   "tmp-",
			suffix:   "",
			want:     "my-project",
		},
		{
			name:     "both prefix and suffix",
			repoName: "tmp-my-project.workspace",
			prefix:   "tmp-",
			suffix:   ".workspace",
			want:     "my-project",
		},
		{
			name:     "neither prefix nor suffix",
			repoName: "my-project",
			prefix:   "",
			suffix:   "",
			want:     "my-project",
		},
		{
			name:     "no match for prefix",
			repoName: "my-project.workspace",
			prefix:   "tmp-",
			suffix:   ".workspace",
			want:     "my-project",
		},
		{
			name:     "no match for suffix",
			repoName: "tmp-my-project",
			prefix:   "tmp-",
			suffix:   ".workspace",
			want:     "my-project",
		},
		{
			name:     "neither configured keeps original",
			repoName: "my-project.workspace",
			prefix:   "",
			suffix:   "",
			want:     "my-project.workspace",
		},
		{
			name:     "prefix only no match keeps original",
			repoName: "my-project",
			prefix:   "tmp-",
			suffix:   "",
			want:     "my-project",
		},
		{
			name:     "suffix only no match keeps original",
			repoName: "my-project",
			prefix:   "",
			suffix:   ".workspace",
			want:     "my-project",
		},
		{
			name:     "complex prefix",
			repoName: "workspace-tmp-my-project",
			prefix:   "workspace-tmp-",
			suffix:   "",
			want:     "my-project",
		},
		{
			name:     "complex suffix",
			repoName: "my-project.workspace.git",
			prefix:   "",
			suffix:   ".workspace.git",
			want:     "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupTempConfig(t, "https://github.com/test-org", tt.prefix, tt.suffix)
			defer cleanup()

			got, err := NormalizeRepoName(tt.repoName)
			if err != nil {
				t.Fatalf("NormalizeRepoName() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("NormalizeRepoName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeRepoName_PrefixBeforeSuffix(t *testing.T) {
	t.Run("verify prefix is removed before suffix", func(t *testing.T) {
		// If we have "tmp-my-project.workspace"
		// And prefix="tmp-", suffix=".workspace"
		// Order should be: "tmp-my-project.workspace" -> "my-project.workspace" -> "my-project"
		_, cleanup := setupTempConfig(t, "https://github.com/test-org", "tmp-", ".workspace")
		defer cleanup()

		result, err := NormalizeRepoName("tmp-my-project.workspace")
		if err != nil {
			t.Fatalf("NormalizeRepoName() unexpected error: %v", err)
		}
		if result != "my-project" {
			t.Errorf("NormalizeRepoName() = %v, want %v", result, "my-project")
		}
	})
}
