package github

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetAuthToken attempts to retrieve GitHub auth token.
// Priority: 1) gh CLI token, 2) git credential helper
// Returns the token or an error with setup instructions.
func GetAuthToken() (string, error) {
	// 1. Try gh CLI first (if available)
	if token, err := getGhToken(); err == nil && token != "" {
		return token, nil
	}

	// 2. Try git credential helper (MAIN METHOD)
	if token, err := getGitCredentialToken(); err == nil && token != "" {
		return token, nil
	}

	// 3. None available - provide clear setup instructions
	return "", fmt.Errorf(
		"No GitHub authentication found.\n\n" +
			"Setup options:\n\n" +
			"1. GitHub CLI (recommended):\n" +
			"   gh auth login\n\n" +
			"2. Git credential helper:\n" +
			"   git config --global credential.helper osxkeychain\n" +
			"   # Then push to GitHub repo - it will prompt for credentials\n" +
			"   # Use Personal Access Token (classic) with 'repo' scope")
}

// getGhToken tries gh CLI as fallback.
// Returns the token from `gh auth token` if gh CLI is installed and authenticated.
func getGhToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh CLI not available or not authenticated: %w", err)
	}

	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("gh CLI returned empty token")
	}

	// Validate token format (GitHub personal access tokens start with ghp_)
	if !strings.HasPrefix(token, "ghp_") && !strings.HasPrefix(token, "github_pat_") {
		return "", fmt.Errorf("invalid gh token format")
	}

	return token, nil
}

// getGitCredentialToken uses git credential helper (MAIN METHOD).
// This retrieves credentials stored in the OS keychain via git's credential system.
// The token is stored securely when a user pushes to GitHub and enters their PAT.
func getGitCredentialToken() (string, error) {
	cmd := exec.Command("git", "credential", "fill")
	// Request credentials for GitHub HTTPS protocol
	cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\n\n")

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git credential helper failed: %w", err)
	}

	// Parse output format:
	// protocol=https
	// host=github.com
	// username=USERNAME
	// password=TOKEN
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "password=") {
			token := strings.TrimPrefix(line, "password=")
			token = strings.TrimSpace(token)

			if token == "" {
				continue
			}

			// Validate token format
			// GitHub tokens: ghp_ (classic), github_pat_ (fine-grained)
			if strings.HasPrefix(token, "ghp_") || strings.HasPrefix(token, "github_pat_") {
				return token, nil
			}
		}
	}

	return "", fmt.Errorf("no valid GitHub token found in credential helper")
}
