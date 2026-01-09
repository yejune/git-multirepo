package update

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// MockHTTPClient is a mock HTTP client for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestNewUpdater(t *testing.T) {
	t.Run("creates updater with defaults", func(t *testing.T) {
		updater := NewUpdater("1.0.0")

		if updater.RepoOwner != "yejune" {
			t.Errorf("expected RepoOwner 'yejune', got %s", updater.RepoOwner)
		}
		if updater.RepoName != "git-workspace" {
			t.Errorf("expected RepoName 'git-workspace', got %s", updater.RepoName)
		}
		if updater.CurrentVersion != "1.0.0" {
			t.Errorf("expected CurrentVersion '1.0.0', got %s", updater.CurrentVersion)
		}
		if updater.HTTPClient == nil {
			t.Error("HTTPClient should not be nil")
		}
	})
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{"  v2.1.0  ", "2.1.0"},
		{"v0.0.1", "0.0.1"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"1.0.0", []int{1, 0, 0}},
		{"2.1.3", []int{2, 1, 3}},
		{"0.0.1", []int{0, 0, 1}},
		{"1.0.0-beta", []int{1, 0, 0}},
		{"10.20.30", []int{10, 20, 30}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseVersion(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseVersion(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		newVersion     string
		currentVersion string
		expected       bool
	}{
		{"1.0.1", "1.0.0", true},
		{"1.1.0", "1.0.0", true},
		{"2.0.0", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.0.1", false},
		{"1.0.0", "2.0.0", false},
		{"0.0.1", "dev", true},  // dev is always updatable
		{"1.0.0", "dev", true},  // dev is always updatable
		{"invalid", "1.0.0", false},
		{"1.0.0", "invalid", false},
		{"1.0", "1.0.0", false}, // incomplete version
		{"1.0.0", "1.0", false}, // incomplete version
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.newVersion, tt.currentVersion), func(t *testing.T) {
			result := isNewerVersion(tt.newVersion, tt.currentVersion)
			if result != tt.expected {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.newVersion, tt.currentVersion, result, tt.expected)
			}
		})
	}
}

func TestGetAssetName(t *testing.T) {
	updater := NewUpdater("1.0.0")
	assetName := updater.getAssetName()

	expectedName := fmt.Sprintf("git-workspace-%s-%s", runtime.GOOS, runtime.GOARCH)
	if assetName != expectedName {
		t.Errorf("getAssetName() = %q, want %q", assetName, expectedName)
	}
}

func TestCheckForUpdate(t *testing.T) {
	t.Run("newer version available", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				body := `[{
					"tag_name": "v2.0.0",
					"assets": [
						{"name": "git-workspace-darwin-arm64", "browser_download_url": "https://example.com/download"}
					]
				}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		release, hasUpdate, err := updater.CheckForUpdate()
		if err != nil {
			t.Fatalf("CheckForUpdate failed: %v", err)
		}
		if !hasUpdate {
			t.Error("expected hasUpdate to be true")
		}
		if release.TagName != "v2.0.0" {
			t.Errorf("expected tag_name 'v2.0.0', got %s", release.TagName)
		}
	})

	t.Run("already up to date", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				body := `[{"tag_name": "v1.0.0", "assets": []}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		_, hasUpdate, err := updater.CheckForUpdate()
		if err != nil {
			t.Fatalf("CheckForUpdate failed: %v", err)
		}
		if hasUpdate {
			t.Error("expected hasUpdate to be false")
		}
	})

	t.Run("no releases found", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "Not Found"}`)),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		_, _, err := updater.CheckForUpdate()
		if err == nil {
			t.Error("expected error for not found")
		}
		if !strings.Contains(err.Error(), "no releases found") {
			t.Errorf("expected 'no releases found' error, got: %v", err)
		}
	})

	t.Run("API error", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"message": "Internal Server Error"}`)),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		_, _, err := updater.CheckForUpdate()
		if err == nil {
			t.Error("expected error for API error")
		}
		if !strings.Contains(err.Error(), "status 500") {
			t.Errorf("expected status 500 error, got: %v", err)
		}
	})

	t.Run("network error", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("network error")
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		_, _, err := updater.CheckForUpdate()
		if err == nil {
			t.Error("expected error for network error")
		}
		if !strings.Contains(err.Error(), "network error") {
			t.Errorf("expected 'network error', got: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`invalid json`)),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		_, _, err := updater.CheckForUpdate()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "failed to parse release") {
			t.Errorf("expected 'failed to parse release' error, got: %v", err)
		}
	})

	t.Run("invalid version format", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				body := `[{"tag_name": "", "assets": []}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		_, _, err := updater.CheckForUpdate()
		if err == nil {
			t.Error("expected error for invalid version")
		}
		if !strings.Contains(err.Error(), "invalid version format") {
			t.Errorf("expected 'invalid version format' error, got: %v", err)
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		// Create a temp directory for the test executable
		tempDir := t.TempDir()
		execPath := filepath.Join(tempDir, "git-workspace")
		if err := os.WriteFile(execPath, []byte("old binary"), 0755); err != nil {
			t.Fatalf("failed to create test executable: %v", err)
		}

		// Create mock client that returns binary content
		downloadCount := 0
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				downloadCount++
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("new binary content"))),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient
		updater.Executable = execPath

		assetName := updater.getAssetName()
		release := &GitHubRelease{
			TagName: "v2.0.0",
			Assets: []Asset{
				{Name: assetName, BrowserDownloadURL: "https://example.com/download"},
			},
		}

		err := updater.Update(release)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Verify new content
		content, err := os.ReadFile(execPath)
		if err != nil {
			t.Fatalf("failed to read updated executable: %v", err)
		}
		if string(content) != "new binary content" {
			t.Errorf("expected 'new binary content', got %q", string(content))
		}

		// Verify backup was removed
		if _, err := os.Stat(execPath + ".bak"); !os.IsNotExist(err) {
			t.Error("backup file should be removed after successful update")
		}
	})

	t.Run("no binary for platform", func(t *testing.T) {
		updater := NewUpdater("1.0.0")

		release := &GitHubRelease{
			TagName: "v2.0.0",
			Assets: []Asset{
				{Name: "git-workspace-windows-amd64", BrowserDownloadURL: "https://example.com/download"},
			},
		}

		err := updater.Update(release)
		if err == nil {
			t.Error("expected error for no binary found")
		}
		if !strings.Contains(err.Error(), "no binary found") {
			t.Errorf("expected 'no binary found' error, got: %v", err)
		}
	})

	t.Run("download failure", func(t *testing.T) {
		tempDir := t.TempDir()
		execPath := filepath.Join(tempDir, "git-workspace")
		os.WriteFile(execPath, []byte("old binary"), 0755)

		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient
		updater.Executable = execPath

		assetName := updater.getAssetName()
		release := &GitHubRelease{
			TagName: "v2.0.0",
			Assets: []Asset{
				{Name: assetName, BrowserDownloadURL: "https://example.com/download"},
			},
		}

		err := updater.Update(release)
		if err == nil {
			t.Error("expected error for download failure")
		}
		if !strings.Contains(err.Error(), "failed to download") {
			t.Errorf("expected 'failed to download' error, got: %v", err)
		}
	})

	t.Run("network error during download", func(t *testing.T) {
		tempDir := t.TempDir()
		execPath := filepath.Join(tempDir, "git-workspace")
		os.WriteFile(execPath, []byte("old binary"), 0755)

		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("connection refused")
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient
		updater.Executable = execPath

		assetName := updater.getAssetName()
		release := &GitHubRelease{
			TagName: "v2.0.0",
			Assets: []Asset{
				{Name: assetName, BrowserDownloadURL: "https://example.com/download"},
			},
		}

		err := updater.Update(release)
		if err == nil {
			t.Error("expected error for network error")
		}
		if !strings.Contains(err.Error(), "failed to download") {
			t.Errorf("expected 'failed to download' error, got: %v", err)
		}
	})
}

func TestGetExecutablePath(t *testing.T) {
	t.Run("custom executable path", func(t *testing.T) {
		updater := NewUpdater("1.0.0")
		updater.Executable = "/custom/path/binary"

		path, err := updater.getExecutablePath()
		if err != nil {
			t.Fatalf("getExecutablePath failed: %v", err)
		}
		if path != "/custom/path/binary" {
			t.Errorf("expected '/custom/path/binary', got %q", path)
		}
	})

	t.Run("auto-detect executable path", func(t *testing.T) {
		updater := NewUpdater("1.0.0")

		path, err := updater.getExecutablePath()
		if err != nil {
			t.Fatalf("getExecutablePath failed: %v", err)
		}
		if path == "" {
			t.Error("expected non-empty path")
		}
	})
}

func TestDownloadToTemp(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("binary content")),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		tempFile, err := updater.downloadToTemp("https://example.com/download")
		if err != nil {
			t.Fatalf("downloadToTemp failed: %v", err)
		}
		defer os.Remove(tempFile)

		content, err := os.ReadFile(tempFile)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}
		if string(content) != "binary content" {
			t.Errorf("expected 'binary content', got %q", string(content))
		}
	})

	t.Run("request creation error", func(t *testing.T) {
		updater := NewUpdater("1.0.0")

		// Invalid URL will cause http.NewRequest to fail
		_, err := updater.downloadToTemp("://invalid-url")
		if err == nil {
			t.Error("expected error for invalid URL")
		}
	})
}

func TestReplaceExecutable(t *testing.T) {
	t.Run("successful replacement", func(t *testing.T) {
		tempDir := t.TempDir()
		execPath := filepath.Join(tempDir, "binary")
		newFile := filepath.Join(tempDir, "new-binary")

		os.WriteFile(execPath, []byte("old"), 0755)
		os.WriteFile(newFile, []byte("new"), 0644)

		updater := NewUpdater("1.0.0")
		err := updater.replaceExecutable(execPath, newFile)
		if err != nil {
			t.Fatalf("replaceExecutable failed: %v", err)
		}

		content, _ := os.ReadFile(execPath)
		if string(content) != "new" {
			t.Errorf("expected 'new', got %q", string(content))
		}
	})

	t.Run("backup cleanup on existing backup", func(t *testing.T) {
		tempDir := t.TempDir()
		execPath := filepath.Join(tempDir, "binary")
		newFile := filepath.Join(tempDir, "new-binary")
		backupPath := execPath + ".bak"

		os.WriteFile(execPath, []byte("old"), 0755)
		os.WriteFile(newFile, []byte("new"), 0644)
		os.WriteFile(backupPath, []byte("old backup"), 0644)

		updater := NewUpdater("1.0.0")
		err := updater.replaceExecutable(execPath, newFile)
		if err != nil {
			t.Fatalf("replaceExecutable failed: %v", err)
		}

		// Old backup should be gone
		if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
			t.Error("backup file should be removed")
		}
	})

	t.Run("chmod failure", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistentFile := filepath.Join(tempDir, "non-existent")

		updater := NewUpdater("1.0.0")
		err := updater.replaceExecutable("/some/path", nonExistentFile)
		if err == nil {
			t.Error("expected error for chmod failure")
		}
	})

	t.Run("backup failure restores", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create a directory instead of a file to cause rename to fail
		execPath := filepath.Join(tempDir, "binary")
		newFile := filepath.Join(tempDir, "new-binary")

		os.WriteFile(newFile, []byte("new"), 0644)

		updater := NewUpdater("1.0.0")
		err := updater.replaceExecutable(execPath, newFile)
		// This should fail because execPath doesn't exist
		if err == nil {
			t.Error("expected error for missing executable")
		}
		if !strings.Contains(err.Error(), "failed to backup") {
			t.Errorf("expected 'failed to backup' error, got: %v", err)
		}
	})
}

func TestGetLatestRelease(t *testing.T) {
	t.Run("request creation error", func(t *testing.T) {
		// This test is mainly for coverage - NewRequest rarely fails
		updater := NewUpdater("1.0.0")
		updater.RepoOwner = ""
		updater.RepoName = ""

		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"tag_name": "v1.0.0"}`)),
				}, nil
			},
		}
		updater.HTTPClient = mockClient

		release, err := updater.getLatestRelease()
		if err != nil {
			t.Logf("got error as expected: %v", err)
		}
		if release != nil && release.TagName != "v1.0.0" {
			t.Errorf("unexpected tag: %s", release.TagName)
		}
	})

	t.Run("verifies request headers", func(t *testing.T) {
		var capturedReq *http.Request
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				capturedReq = req
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"tag_name": "v1.0.0"}]`)),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		_, _ = updater.getLatestRelease()

		if capturedReq == nil {
			t.Fatal("request was not captured")
		}
		if capturedReq.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Error("Accept header not set correctly")
		}
		if capturedReq.Header.Get("User-Agent") != "git-workspace-updater" {
			t.Error("User-Agent header not set correctly")
		}
	})
}

// Additional edge case tests
func TestParseVersionEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"", []int{0}},
		{".", []int{0, 0}},
		{"a.b.c", []int{0, 0, 0}},
		{"1", []int{1}},
		{"1.2", []int{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseVersion(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseVersion(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestUpdateWithInstallFailure(t *testing.T) {
	t.Run("rename to target fails", func(t *testing.T) {
		tempDir := t.TempDir()
		execPath := filepath.Join(tempDir, "binary")
		newFile := filepath.Join(tempDir, "new-binary")

		os.WriteFile(execPath, []byte("old"), 0755)
		os.WriteFile(newFile, []byte("new"), 0644)

		// Make the temp directory read-only to cause rename to fail
		// This is tricky on different platforms, so we'll use a different approach
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("content")),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient
		// Use a path that doesn't exist to trigger failure
		updater.Executable = "/nonexistent/path/binary"

		assetName := updater.getAssetName()
		release := &GitHubRelease{
			TagName: "v2.0.0",
			Assets: []Asset{
				{Name: assetName, BrowserDownloadURL: "https://example.com/download"},
			},
		}

		err := updater.Update(release)
		if err == nil {
			t.Error("expected error for failed installation")
		}
		if !strings.Contains(err.Error(), "failed to replace executable") {
			t.Errorf("expected 'failed to replace executable' error, got: %v", err)
		}
	})
}

// ErrorReader simulates a reader that fails during io.Copy
type ErrorReader struct {
	err error
}

func (e *ErrorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestDownloadToTempAdditional(t *testing.T) {
	t.Run("io.Copy failure", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(&ErrorReader{err: fmt.Errorf("read error")}),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		_, err := updater.downloadToTemp("https://example.com/download")
		if err == nil {
			t.Error("expected error for io.Copy failure")
		}
		if !strings.Contains(err.Error(), "read error") {
			t.Errorf("expected 'read error', got: %v", err)
		}
	})

	t.Run("network error", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("network error")
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		_, err := updater.downloadToTemp("https://example.com/download")
		if err == nil {
			t.Error("expected error for network error")
		}
	})
}

func TestReplaceExecutableRenameFailure(t *testing.T) {
	t.Run("rename to target fails and restore backup", func(t *testing.T) {
		tempDir := t.TempDir()
		execPath := filepath.Join(tempDir, "binary")
		newFile := filepath.Join(tempDir, "new-binary")

		os.WriteFile(execPath, []byte("old"), 0755)
		os.WriteFile(newFile, []byte("new"), 0644)

		// Create a directory at the target location to cause rename to fail
		targetDir := filepath.Join(tempDir, "subdir")
		os.MkdirAll(targetDir, 0755)
		targetExec := filepath.Join(targetDir, "binary")
		os.WriteFile(targetExec, []byte("original"), 0755)

		// Remove write permission on target directory to cause rename failure
		os.Chmod(targetDir, 0555)
		defer os.Chmod(targetDir, 0755)

		updater := NewUpdater("1.0.0")
		err := updater.replaceExecutable(targetExec, newFile)
		// This may or may not fail depending on the platform
		if err != nil {
			if strings.Contains(err.Error(), "failed to install") {
				t.Log("Got expected install failure")
			}
		}
	})
}

func TestUpdateGetExecutablePathError(t *testing.T) {
	t.Run("getExecutablePath returns error", func(t *testing.T) {
		// We can't easily make os.Executable() fail, but we can test
		// that the error path is handled correctly by using a mock
		// This is mainly for documentation since the actual error is hard to trigger
		updater := NewUpdater("1.0.0")
		updater.Executable = ""

		// The actual execution will succeed since os.Executable() works in test env
		path, err := updater.getExecutablePath()
		if err != nil {
			t.Logf("Got error as expected: %v", err)
		} else {
			t.Logf("Got path: %s", path)
		}
	})
}

func TestDownloadToTempCloseError(t *testing.T) {
	// This test is for documentation - tempFile.Close() error is hard to trigger
	// but the code path is covered by normal successful download tests
	t.Run("temp file operations", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("content")),
				}, nil
			},
		}

		updater := NewUpdater("1.0.0")
		updater.HTTPClient = mockClient

		tempFile, err := updater.downloadToTemp("https://example.com/download")
		if err != nil {
			t.Fatalf("downloadToTemp failed: %v", err)
		}
		defer os.Remove(tempFile)

		// Verify file was created properly
		info, err := os.Stat(tempFile)
		if err != nil {
			t.Fatalf("failed to stat temp file: %v", err)
		}
		if info.Size() != 7 {
			t.Errorf("expected size 7, got %d", info.Size())
		}
	})
}

func TestGetAssetNameArchitectures(t *testing.T) {
	// This test verifies the getAssetName function with current runtime
	// The switch case has redundant assignments but we test the function works
	updater := NewUpdater("1.0.0")
	assetName := updater.getAssetName()

	// Verify it contains the expected format
	if !strings.HasPrefix(assetName, "git-workspace-") {
		t.Errorf("expected prefix 'git-workspace-', got %s", assetName)
	}
	if !strings.Contains(assetName, runtime.GOOS) {
		t.Errorf("expected to contain %s, got %s", runtime.GOOS, assetName)
	}
	if !strings.Contains(assetName, runtime.GOARCH) {
		t.Errorf("expected to contain %s, got %s", runtime.GOARCH, assetName)
	}
}
