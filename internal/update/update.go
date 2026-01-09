// Package update provides self-update functionality for git-workspace
package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName    string  `json:"tag_name"`
	Assets     []Asset `json:"assets"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// HTTPClient interface for mocking HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Updater handles the self-update process
type Updater struct {
	RepoOwner      string
	RepoName       string
	CurrentVersion string
	HTTPClient     HTTPClient
	Executable     string // path to current executable, empty means auto-detect
}

// NewUpdater creates a new Updater with default settings
func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		RepoOwner:      "yejune",
		RepoName:       "git-workspace",
		CurrentVersion: currentVersion,
		HTTPClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// CheckForUpdate checks if a newer version is available
func (u *Updater) CheckForUpdate() (*GitHubRelease, bool, error) {
	release, err := u.getLatestRelease()
	if err != nil {
		return nil, false, err
	}

	latestVersion := normalizeVersion(release.TagName)
	currentVersion := normalizeVersion(u.CurrentVersion)

	if latestVersion == "" || currentVersion == "" {
		return release, false, fmt.Errorf("invalid version format")
	}

	if isNewerVersion(latestVersion, currentVersion) {
		return release, true, nil
	}

	return release, false, nil
}

// Update downloads and installs the latest version
func (u *Updater) Update(release *GitHubRelease) error {
	assetName := u.getAssetName()
	var downloadURL string

	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Get current executable path
	execPath, err := u.getExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Download to temp file
	tempFile, err := u.downloadToTemp(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tempFile)

	// Replace the executable
	if err := u.replaceExecutable(execPath, tempFile); err != nil {
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	return nil
}

// getLatestRelease fetches the latest release from GitHub API
func (u *Updater) getLatestRelease() (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", u.RepoOwner, u.RepoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "git-workspace-updater")

	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}

	// Find first non-draft, non-prerelease
	for _, r := range releases {
		if !r.Draft && !r.Prerelease {
			return &r, nil
		}
	}

	// If all are drafts/prereleases, return the first one
	return &releases[0], nil
}

// getAssetName returns the expected asset name for the current platform
func (u *Updater) getAssetName() string {
	return fmt.Sprintf("git-workspace-%s-%s", runtime.GOOS, runtime.GOARCH)
}

// getExecutablePath returns the path to the current executable
func (u *Updater) getExecutablePath() (string, error) {
	if u.Executable != "" {
		return u.Executable, nil
	}

	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Resolve symlinks
	return filepath.EvalSymlinks(execPath)
}

// downloadToTemp downloads a file to a temporary location
func (u *Updater) downloadToTemp(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "git-workspace-updater")

	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tempFile, err := os.CreateTemp("", "git-workspace-update-*")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", err
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// replaceExecutable replaces the current executable with the new one
func (u *Updater) replaceExecutable(execPath, tempFile string) error {
	// Make the new binary executable
	if err := os.Chmod(tempFile, 0755); err != nil {
		return err
	}

	// Create backup path
	backupPath := execPath + ".bak"

	// Remove old backup if exists
	os.Remove(backupPath)

	// Rename current to backup
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current executable: %w", err)
	}

	// Move new to current location
	if err := os.Rename(tempFile, execPath); err != nil {
		// Try to restore backup
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install new executable: %w", err)
	}

	// Remove backup on success
	os.Remove(backupPath)

	return nil
}

// normalizeVersion removes 'v' prefix and returns clean version string
func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	return version
}

// isNewerVersion compares two semantic versions and returns true if new > current
func isNewerVersion(newVersion, currentVersion string) bool {
	// Handle dev version - always updatable
	if currentVersion == "dev" {
		return true
	}

	newParts := parseVersion(newVersion)
	currentParts := parseVersion(currentVersion)

	if len(newParts) < 3 || len(currentParts) < 3 {
		return false
	}

	for i := 0; i < 3; i++ {
		if newParts[i] > currentParts[i] {
			return true
		}
		if newParts[i] < currentParts[i] {
			return false
		}
	}

	return false
}

// parseVersion parses a version string into integer components
func parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	result := make([]int, len(parts))

	for i, part := range parts {
		// Remove any non-numeric suffix (e.g., "1.0.0-beta")
		numPart := strings.Split(part, "-")[0]
		var num int
		fmt.Sscanf(numPart, "%d", &num)
		result[i] = num
	}

	return result
}
