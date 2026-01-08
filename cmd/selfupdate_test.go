package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/yejune/git-workspace/internal/update"
)

// MockHTTPClient is a mock HTTP client for testing
type MockHTTPClientCmd struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClientCmd) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestRunSelfupdate(t *testing.T) {
	// Save original updaterFactory
	originalFactory := updaterFactory
	defer func() { updaterFactory = originalFactory }()

	t.Run("already up to date", func(t *testing.T) {
		updaterFactory = func(version string) *update.Updater {
			mockClient := &MockHTTPClientCmd{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					body := `{"tag_name": "v1.0.0", "assets": []}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
					}, nil
				},
			}
			u := update.NewUpdater(version)
			u.HTTPClient = mockClient
			return u
		}

		// Save original Version
		originalVersion := Version
		Version = "1.0.0"
		defer func() { Version = originalVersion }()

		output := captureOutput(func() {
			err := runSelfupdate(selfupdateCmd, []string{})
			if err != nil {
				t.Errorf("runSelfupdate failed: %v", err)
			}
		})

		if !strings.Contains(output, "Already up to date") {
			t.Errorf("expected 'Already up to date', got: %s", output)
		}
	})

	t.Run("check for update fails", func(t *testing.T) {
		updaterFactory = func(version string) *update.Updater {
			mockClient := &MockHTTPClientCmd{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return nil, fmt.Errorf("network error")
				},
			}
			u := update.NewUpdater(version)
			u.HTTPClient = mockClient
			return u
		}

		err := runSelfupdate(selfupdateCmd, []string{})
		if err == nil {
			t.Error("expected error for network failure")
		}
		if !strings.Contains(err.Error(), "failed to check for updates") {
			t.Errorf("expected 'failed to check for updates' error, got: %v", err)
		}
	})

	t.Run("update available and download fails", func(t *testing.T) {
		tempDir := t.TempDir()
		execPath := tempDir + "/git-subclone"
		os.WriteFile(execPath, []byte("old binary"), 0755)

		updaterFactory = func(version string) *update.Updater {
			requestCount := 0
			mockClient := &MockHTTPClientCmd{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					requestCount++
					if requestCount == 1 {
						// First request: check for update
						assetName := fmt.Sprintf("git-subclone-%s-%s", runtime.GOOS, runtime.GOARCH)
						body := fmt.Sprintf(`{
							"tag_name": "v2.0.0",
							"assets": [{"name": "%s", "browser_download_url": "https://example.com/download"}]
						}`, assetName)
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(body)),
						}, nil
					}
					// Second request: download - fail
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(strings.NewReader("not found")),
					}, nil
				},
			}
			u := update.NewUpdater(version)
			u.HTTPClient = mockClient
			u.Executable = execPath
			return u
		}

		// Save original Version
		originalVersion := Version
		Version = "1.0.0"
		defer func() { Version = originalVersion }()

		output := captureOutput(func() {
			err := runSelfupdate(selfupdateCmd, []string{})
			if err == nil {
				t.Error("expected error for download failure")
			}
			if !strings.Contains(err.Error(), "failed to update") {
				t.Errorf("expected 'failed to update' error, got: %v", err)
			}
		})

		if !strings.Contains(output, "New version available") {
			t.Errorf("expected 'New version available', got: %s", output)
		}
	})

	t.Run("successful update", func(t *testing.T) {
		tempDir := t.TempDir()
		execPath := tempDir + "/git-subclone"
		os.WriteFile(execPath, []byte("old binary"), 0755)

		updaterFactory = func(version string) *update.Updater {
			requestCount := 0
			mockClient := &MockHTTPClientCmd{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					requestCount++
					if requestCount == 1 {
						// First request: check for update
						assetName := fmt.Sprintf("git-subclone-%s-%s", runtime.GOOS, runtime.GOARCH)
						body := fmt.Sprintf(`{
							"tag_name": "v2.0.0",
							"assets": [{"name": "%s", "browser_download_url": "https://example.com/download"}]
						}`, assetName)
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(body)),
						}, nil
					}
					// Second request: download - success
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte("new binary content"))),
					}, nil
				},
			}
			u := update.NewUpdater(version)
			u.HTTPClient = mockClient
			u.Executable = execPath
			return u
		}

		// Save original Version
		originalVersion := Version
		Version = "1.0.0"
		defer func() { Version = originalVersion }()

		output := captureOutput(func() {
			err := runSelfupdate(selfupdateCmd, []string{})
			if err != nil {
				t.Errorf("runSelfupdate failed: %v", err)
			}
		})

		if !strings.Contains(output, "Successfully updated") {
			t.Errorf("expected 'Successfully updated', got: %s", output)
		}
		if !strings.Contains(output, "v2.0.0") {
			t.Errorf("expected 'v2.0.0' in output, got: %s", output)
		}

		// Verify executable was updated
		content, _ := os.ReadFile(execPath)
		if string(content) != "new binary content" {
			t.Errorf("expected 'new binary content', got: %s", string(content))
		}
	})

	t.Run("dev version can update", func(t *testing.T) {
		tempDir := t.TempDir()
		execPath := tempDir + "/git-subclone"
		os.WriteFile(execPath, []byte("old binary"), 0755)

		updaterFactory = func(version string) *update.Updater {
			requestCount := 0
			mockClient := &MockHTTPClientCmd{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					requestCount++
					if requestCount == 1 {
						assetName := fmt.Sprintf("git-subclone-%s-%s", runtime.GOOS, runtime.GOARCH)
						body := fmt.Sprintf(`{
							"tag_name": "v1.0.0",
							"assets": [{"name": "%s", "browser_download_url": "https://example.com/download"}]
						}`, assetName)
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(body)),
						}, nil
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte("new binary"))),
					}, nil
				},
			}
			u := update.NewUpdater(version)
			u.HTTPClient = mockClient
			u.Executable = execPath
			return u
		}

		// Set version to dev
		originalVersion := Version
		Version = "dev"
		defer func() { Version = originalVersion }()

		output := captureOutput(func() {
			err := runSelfupdate(selfupdateCmd, []string{})
			if err != nil {
				t.Errorf("runSelfupdate failed: %v", err)
			}
		})

		if !strings.Contains(output, "Successfully updated") {
			t.Errorf("expected 'Successfully updated', got: %s", output)
		}
	})
}

func TestSelfupdateCommand(t *testing.T) {
	t.Run("selfupdate command is registered", func(t *testing.T) {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "selfupdate" {
				found = true
				break
			}
		}
		if !found {
			t.Error("selfupdate command should be registered with rootCmd")
		}
	})

	t.Run("selfupdate command short description", func(t *testing.T) {
		if selfupdateCmd.Short == "" {
			t.Error("selfupdate command should have a short description")
		}
	})

	t.Run("selfupdate command long description", func(t *testing.T) {
		if selfupdateCmd.Long == "" {
			t.Error("selfupdate command should have a long description")
		}
	})
}
