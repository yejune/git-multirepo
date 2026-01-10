package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConfigExists checks if ~/.git.workspaces exists
func ConfigExists() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	configPath := filepath.Join(home, ".git.workspaces")
	_, err = os.Stat(configPath)
	return err == nil
}

// GetOrganization reads workspace.organization from config
// Returns: "https://github.com/git-workspaces", error
func GetOrganization() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(home, ".git.workspaces")

	cmd := exec.Command("git", "config", "-f", configPath,
		"--get", "workspace.organization")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("organization not configured in ~/.git.workspaces")
	}

	org := strings.TrimSpace(string(out))
	if org == "" {
		return "", fmt.Errorf("organization is empty")
	}

	return org, nil
}

// GetStripPrefix reads workspace.stripPrefix from config (optional)
// Returns: "tmp-", nil if not set
func GetStripPrefix() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(home, ".git.workspaces")

	cmd := exec.Command("git", "config", "-f", configPath,
		"--get", "workspace.stripPrefix")
	out, err := cmd.Output()
	if err != nil {
		return "", nil // Not set - no error
	}

	return strings.TrimSpace(string(out)), nil
}

// GetStripSuffix reads workspace.stripSuffix from config (optional)
// Returns: ".workspace", nil if not set
func GetStripSuffix() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(home, ".git.workspaces")

	cmd := exec.Command("git", "config", "-f", configPath,
		"--get", "workspace.stripSuffix")
	out, err := cmd.Output()
	if err != nil {
		return "", nil // Not set - no error
	}

	return strings.TrimSpace(string(out)), nil
}

// NormalizeRepoName removes prefix/suffix based on config
// Order: Remove prefix first, then suffix
func NormalizeRepoName(name string) (string, error) {
	// 1. Remove prefix if configured
	prefix, err := GetStripPrefix()
	if err != nil {
		return "", err
	}
	if prefix != "" {
		name = strings.TrimPrefix(name, prefix)
	}

	// 2. Remove suffix if configured
	suffix, err := GetStripSuffix()
	if err != nil {
		return "", err
	}
	if suffix != "" {
		name = strings.TrimSuffix(name, suffix)
	}

	return name, nil
}
