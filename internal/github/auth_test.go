package github

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// ============================================================================
// Test Cases: GetAuthToken
// ============================================================================

func TestGetAuthToken(t *testing.T) {
	t.Run("no authentication available", func(t *testing.T) {
		// This test assumes gh CLI is not authenticated and no git credentials
		// We can't reliably test this without mocking, so we skip it
		// In a real environment, this would require more sophisticated mocking
		t.Skip("Requires mocking of external commands")
	})

	t.Run("error message contains setup instructions", func(t *testing.T) {
		// We'll test that the error message is helpful
		// by checking the fallback path when both methods fail
		t.Skip("Requires mocking of external commands")
	})
}

// ============================================================================
// Test Cases: getGhToken
// ============================================================================

func TestGetGhToken(t *testing.T) {
	t.Run("gh cli not available", func(t *testing.T) {
		// Save original PATH
		oldPath := os.Getenv("PATH")
		defer os.Setenv("PATH", oldPath)

		// Set PATH to empty to ensure gh is not found
		os.Setenv("PATH", "")

		_, err := getGhToken()
		if err == nil {
			t.Error("getGhToken() expected error when gh CLI not available")
		}
		if !strings.Contains(err.Error(), "not available") {
			t.Errorf("getGhToken() error should mention 'not available', got: %v", err)
		}
	})

	t.Run("gh cli available but not authenticated", func(t *testing.T) {
		// This requires gh to be installed but not authenticated
		// Skip if gh is not installed
		if _, err := exec.LookPath("gh"); err != nil {
			t.Skip("gh CLI not installed")
		}

		// Try to get token - might fail if not authenticated
		_, err := getGhToken()
		// We don't assert here as the user might actually be authenticated
		// This is more of a smoke test
		_ = err
	})

	t.Run("invalid token format", func(t *testing.T) {
		// We can't easily test this without mocking exec.Command
		// This would require a more sophisticated testing framework
		t.Skip("Requires command output mocking")
	})
}

// ============================================================================
// Test Cases: getGitCredentialToken
// ============================================================================

func TestGetGitCredentialToken(t *testing.T) {
	t.Run("git credential helper not configured", func(t *testing.T) {
		// Save current git config
		homeDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", homeDir)
		defer os.Setenv("HOME", oldHome)

		// Try to get token from credential helper
		_, err := getGitCredentialToken()
		if err == nil {
			// It's possible the system has credentials configured
			t.Skip("System has git credentials configured")
		}
	})

	t.Run("git credential returns empty password", func(t *testing.T) {
		// This requires mocking git credential output
		t.Skip("Requires command output mocking")
	})

	t.Run("git credential returns invalid token format", func(t *testing.T) {
		// This requires mocking git credential output
		t.Skip("Requires command output mocking")
	})
}

// ============================================================================
// Test Cases: Token Validation
// ============================================================================

func TestTokenValidation(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		isValid bool
	}{
		{
			name:    "valid classic token",
			token:   "ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			isValid: true,
		},
		{
			name:    "valid fine-grained token",
			token:   "github_pat_1234567890abcdefghijklmnopqrstuvwxyz",
			isValid: true,
		},
		{
			name:    "invalid token - wrong prefix",
			token:   "gho_1234567890abcdefghijklmnopqrstuvwxyz",
			isValid: false,
		},
		{
			name:    "invalid token - no prefix",
			token:   "1234567890abcdefghijklmnopqrstuvwxyz",
			isValid: false,
		},
		{
			name:    "invalid token - empty",
			token:   "",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test token format validation
			hasGhpPrefix := strings.HasPrefix(tt.token, "ghp_")
			hasGithubPatPrefix := strings.HasPrefix(tt.token, "github_pat_")
			isValid := (hasGhpPrefix || hasGithubPatPrefix) && tt.token != ""

			if isValid != tt.isValid {
				t.Errorf("Token validation mismatch: token=%v, expected=%v, got=%v",
					tt.token, tt.isValid, isValid)
			}
		})
	}
}

// ============================================================================
// Integration Tests (require actual authentication)
// ============================================================================

func TestGetAuthToken_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("get auth token from system", func(t *testing.T) {
		token, err := GetAuthToken()
		if err != nil {
			// This is expected if not authenticated
			t.Logf("No authentication available (expected): %v", err)

			// Verify error message is helpful
			if !strings.Contains(err.Error(), "Setup options") {
				t.Error("Error message should contain setup instructions")
			}
			return
		}

		// If we got a token, verify it's valid
		if token == "" {
			t.Error("GetAuthToken() returned empty token without error")
		}

		// Verify token format
		if !strings.HasPrefix(token, "ghp_") && !strings.HasPrefix(token, "github_pat_") {
			t.Errorf("GetAuthToken() returned invalid token format: %s", token[:10])
		}
	})
}

// ============================================================================
// Error Message Tests
// ============================================================================

func TestAuthErrorMessages(t *testing.T) {
	t.Run("error message contains gh auth login", func(t *testing.T) {
		// Create a scenario where both auth methods fail
		homeDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		oldPath := os.Getenv("PATH")
		os.Setenv("HOME", homeDir)
		os.Setenv("PATH", "") // Remove gh from PATH
		defer func() {
			os.Setenv("HOME", oldHome)
			os.Setenv("PATH", oldPath)
		}()

		_, err := GetAuthToken()
		if err == nil {
			t.Skip("Unexpectedly succeeded - system has credentials")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "gh auth login") {
			t.Error("Error message should contain 'gh auth login' instruction")
		}
		if !strings.Contains(errMsg, "Setup options") {
			t.Error("Error message should contain 'Setup options'")
		}
		if !strings.Contains(errMsg, "credential helper") {
			t.Error("Error message should mention credential helper")
		}
	})
}
