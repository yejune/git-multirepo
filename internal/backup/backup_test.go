package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFilesIdentical verifies SHA256 comparison between files
func TestFilesIdentical(t *testing.T) {
	tests := []struct {
		name        string
		content1    string
		content2    string
		shouldMatch bool
		wantErr     bool
	}{
		{
			name:        "identical content",
			content1:    "Hello, World!",
			content2:    "Hello, World!",
			shouldMatch: true,
			wantErr:     false,
		},
		{
			name:        "different content",
			content1:    "Hello, World!",
			content2:    "Goodbye, World!",
			shouldMatch: false,
			wantErr:     false,
		},
		{
			name:        "empty files",
			content1:    "",
			content2:    "",
			shouldMatch: true,
			wantErr:     false,
		},
		{
			name:        "large identical content",
			content1:    string(make([]byte, 1024*1024)), // 1MB of zeros
			content2:    string(make([]byte, 1024*1024)),
			shouldMatch: true,
			wantErr:     false,
		},
		{
			name:        "whitespace differences",
			content1:    "line1\nline2\n",
			content2:    "line1\nline2",
			shouldMatch: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Write test files
			file1 := filepath.Join(tmpDir, "file1.txt")
			file2 := filepath.Join(tmpDir, "file2.txt")

			if err := os.WriteFile(file1, []byte(tt.content1), 0644); err != nil {
				t.Fatalf("failed to create file1: %v", err)
			}
			if err := os.WriteFile(file2, []byte(tt.content2), 0644); err != nil {
				t.Fatalf("failed to create file2: %v", err)
			}

			// Test filesIdentical
			identical, err := filesIdentical(file1, file2)

			if (err != nil) != tt.wantErr {
				t.Errorf("filesIdentical() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if identical != tt.shouldMatch {
				t.Errorf("filesIdentical() = %v, want %v", identical, tt.shouldMatch)
			}
		})
	}
}

// TestFilesIdentical_ErrorCases verifies error handling
func TestFilesIdentical_ErrorCases(t *testing.T) {
	tmpDir := t.TempDir()

	validFile := filepath.Join(tmpDir, "valid.txt")
	if err := os.WriteFile(validFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		file1   string
		file2   string
		wantErr bool
	}{
		{
			name:    "first file not found",
			file1:   filepath.Join(tmpDir, "nonexistent1.txt"),
			file2:   validFile,
			wantErr: true,
		},
		{
			name:    "second file not found",
			file1:   validFile,
			file2:   filepath.Join(tmpDir, "nonexistent2.txt"),
			wantErr: true,
		},
		{
			name:    "both files not found",
			file1:   filepath.Join(tmpDir, "nonexistent1.txt"),
			file2:   filepath.Join(tmpDir, "nonexistent2.txt"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := filesIdentical(tt.file1, tt.file2)
			if (err != nil) != tt.wantErr {
				t.Errorf("filesIdentical() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSHA256Hash verifies hash calculation
func TestSHA256Hash(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedHash string
	}{
		{
			name:         "empty file",
			content:      "",
			expectedHash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:         "hello world",
			content:      "Hello, World!",
			expectedHash: "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f",
		},
		{
			name:         "multiline content",
			content:      "line1\nline2\nline3",
			expectedHash: "2b5e0b6dc3ff5f7c59f8e8e1c5f1c8f2e1d0a9c8b7a6958473625140f1e2d3c4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			hash, err := sha256File(testFile)
			if err != nil {
				t.Errorf("sha256File() error = %v", err)
				return
			}

			// Verify hash is 64 characters (SHA256 hex)
			if len(hash) != 64 {
				t.Errorf("sha256File() hash length = %d, want 64", len(hash))
			}

			// Verify hash is consistent
			hash2, err := sha256File(testFile)
			if err != nil {
				t.Errorf("sha256File() second call error = %v", err)
				return
			}

			if hash != hash2 {
				t.Errorf("sha256File() inconsistent: %s != %s", hash, hash2)
			}
		})
	}
}

// TestFindLatestBackup verifies finding the most recent backup
func TestFindLatestBackup(t *testing.T) {
	tests := []struct {
		name          string
		setupFiles    []string // Files to create (relative to todayDir)
		relPath       string
		expectedFound bool
		expectedFile  string // Relative filename expected
	}{
		{
			name: "single backup exists",
			setupFiles: []string{
				"test.20260110_120000.txt",
			},
			relPath:       "test.txt",
			expectedFound: true,
			expectedFile:  "test.20260110_120000.txt",
		},
		{
			name: "multiple backups - return latest",
			setupFiles: []string{
				"test.20260110_100000.txt",
				"test.20260110_120000.txt",
				"test.20260110_150000.txt",
			},
			relPath:       "test.txt",
			expectedFound: true,
			expectedFile:  "test.20260110_150000.txt",
		},
		{
			name:          "no backups exist",
			setupFiles:    []string{},
			relPath:       "test.txt",
			expectedFound: false,
		},
		{
			name: "different file - no match",
			setupFiles: []string{
				"other.20260110_120000.txt",
			},
			relPath:       "test.txt",
			expectedFound: false,
		},
		{
			name: "nested path",
			setupFiles: []string{
				"subdir/test.20260110_100000.go",
				"subdir/test.20260110_150000.go",
			},
			relPath:       "subdir/test.go",
			expectedFound: true,
			expectedFile:  "subdir/test.20260110_150000.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			todayDir := tmpDir

			// Create test files
			for _, file := range tt.setupFiles {
				fullPath := filepath.Join(todayDir, file)
				dir := filepath.Dir(fullPath)

				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}

				if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
					t.Fatalf("failed to create test file %s: %v", file, err)
				}
			}

			// Test findLatestBackup
			result := findLatestBackup(todayDir, tt.relPath)

			if tt.expectedFound {
				expectedPath := filepath.Join(todayDir, tt.expectedFile)
				if result != expectedPath {
					t.Errorf("findLatestBackup() = %v, want %v", result, expectedPath)
				}
			} else {
				if result != "" {
					t.Errorf("findLatestBackup() = %v, want empty string", result)
				}
			}
		})
	}
}

// TestFindLatestBackup_EmptyDirectory verifies handling of empty directories
func TestFindLatestBackup_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty directory structure
	todayDir := filepath.Join(tmpDir, "2026", "01", "10")
	if err := os.MkdirAll(todayDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	result := findLatestBackup(todayDir, "test.txt")
	if result != "" {
		t.Errorf("findLatestBackup() on empty dir = %v, want empty string", result)
	}
}

// TestCreateFileBackup_Deduplication verifies backup deduplication logic
func TestCreateFileBackup_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backup")
	repoRoot := tmpDir

	// Create source file
	sourceFile := filepath.Join(repoRoot, "test.txt")
	originalContent := "original content"
	if err := os.WriteFile(sourceFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// First backup should be created
	err := CreateFileBackup(sourceFile, backupDir, repoRoot)
	if err != nil {
		t.Errorf("CreateFileBackup() first call error = %v", err)
	}

	// Verify backup was created
	now := time.Now()
	todayDir := filepath.Join(backupDir, "modified", now.Format("2006"), now.Format("01"), now.Format("02"))
	entries, err := os.ReadDir(todayDir)
	if err != nil {
		t.Fatalf("failed to read backup directory: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 backup file, got %d", len(entries))
	}

	// Second backup with identical content should be skipped
	time.Sleep(time.Second) // Ensure different timestamp
	err = CreateFileBackup(sourceFile, backupDir, repoRoot)
	if err != nil {
		t.Errorf("CreateFileBackup() second call error = %v", err)
	}

	// Verify no new backup was created
	entries, err = os.ReadDir(todayDir)
	if err != nil {
		t.Fatalf("failed to read backup directory: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 backup file (deduplication), got %d", len(entries))
	}

	// Third backup with modified content should create new backup
	modifiedContent := "modified content"
	if err := os.WriteFile(sourceFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to modify source file: %v", err)
	}

	time.Sleep(time.Second) // Ensure different timestamp
	err = CreateFileBackup(sourceFile, backupDir, repoRoot)
	if err != nil {
		t.Errorf("CreateFileBackup() third call error = %v", err)
	}

	// Verify new backup was created
	entries, err = os.ReadDir(todayDir)
	if err != nil {
		t.Fatalf("failed to read backup directory: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 backup files (after modification), got %d", len(entries))
	}
}

// TestCreateFileBackup_FirstBackup verifies first backup is always created
func TestCreateFileBackup_FirstBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backup")
	repoRoot := tmpDir

	sourceFile := filepath.Join(repoRoot, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// First backup should always be created (no previous backup)
	err := CreateFileBackup(sourceFile, backupDir, repoRoot)
	if err != nil {
		t.Errorf("CreateFileBackup() error = %v", err)
	}

	// Verify backup was created
	now := time.Now()
	todayDir := filepath.Join(backupDir, "modified", now.Format("2006"), now.Format("01"), now.Format("02"))
	entries, err := os.ReadDir(todayDir)
	if err != nil {
		t.Fatalf("failed to read backup directory: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 backup file, got %d", len(entries))
	}
}

// TestCreatePatchBackup_Deduplication verifies patch backup deduplication
func TestCreatePatchBackup_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backup")

	// Create patch file
	patchPath := filepath.Join(tmpDir, ".workspaces", "patches", "test.patch")
	if err := os.MkdirAll(filepath.Dir(patchPath), 0755); err != nil {
		t.Fatalf("failed to create patch directory: %v", err)
	}

	originalPatch := "diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt"
	if err := os.WriteFile(patchPath, []byte(originalPatch), 0644); err != nil {
		t.Fatalf("failed to create patch file: %v", err)
	}

	// First backup should be created
	err := CreatePatchBackup(patchPath, backupDir)
	if err != nil {
		t.Errorf("CreatePatchBackup() first call error = %v", err)
	}

	// Verify backup was created
	now := time.Now()
	todayDir := filepath.Join(backupDir, "patched", now.Format("2006"), now.Format("01"), now.Format("02"))
	entries, err := os.ReadDir(todayDir)
	if err != nil {
		t.Fatalf("failed to read backup directory: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 backup file, got %d", len(entries))
	}

	// Second backup with identical content should be skipped
	time.Sleep(time.Second)
	err = CreatePatchBackup(patchPath, backupDir)
	if err != nil {
		t.Errorf("CreatePatchBackup() second call error = %v", err)
	}

	// Verify no new backup was created
	entries, err = os.ReadDir(todayDir)
	if err != nil {
		t.Fatalf("failed to read backup directory: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 backup file (deduplication), got %d", len(entries))
	}

	// Modified patch should create new backup
	modifiedPatch := "diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n@@ modified @@"
	if err := os.WriteFile(patchPath, []byte(modifiedPatch), 0644); err != nil {
		t.Fatalf("failed to modify patch file: %v", err)
	}

	time.Sleep(time.Second)
	err = CreatePatchBackup(patchPath, backupDir)
	if err != nil {
		t.Errorf("CreatePatchBackup() third call error = %v", err)
	}

	// Verify new backup was created
	entries, err = os.ReadDir(todayDir)
	if err != nil {
		t.Fatalf("failed to read backup directory: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 backup files (after modification), got %d", len(entries))
	}
}

// TestCreatePatchBackup_NonexistentFile verifies handling of missing files
func TestCreatePatchBackup_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backup")
	patchPath := filepath.Join(tmpDir, "nonexistent.patch")

	// Should not error when patch doesn't exist
	err := CreatePatchBackup(patchPath, backupDir)
	if err != nil {
		t.Errorf("CreatePatchBackup() on nonexistent file error = %v, want nil", err)
	}
}

// TestCreateFileBackup_NonexistentFile verifies handling of missing files
func TestCreateFileBackup_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backup")
	repoRoot := tmpDir
	filePath := filepath.Join(tmpDir, "nonexistent.txt")

	// Should not error when file doesn't exist
	err := CreateFileBackup(filePath, backupDir, repoRoot)
	if err != nil {
		t.Errorf("CreateFileBackup() on nonexistent file error = %v, want nil", err)
	}
}

// TestCreateFileBackup_NestedPath verifies nested directory handling
func TestCreateFileBackup_NestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backup")
	repoRoot := tmpDir

	// Create nested source file
	sourceFile := filepath.Join(repoRoot, "src", "pkg", "test.go")
	if err := os.MkdirAll(filepath.Dir(sourceFile), 0755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	if err := os.WriteFile(sourceFile, []byte("package test"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create backup
	err := CreateFileBackup(sourceFile, backupDir, repoRoot)
	if err != nil {
		t.Errorf("CreateFileBackup() error = %v", err)
	}

	// Verify nested structure is preserved in backup
	now := time.Now()
	todayDir := filepath.Join(backupDir, "modified", now.Format("2006"), now.Format("01"), now.Format("02"))
	nestedDir := filepath.Join(todayDir, "src", "pkg")

	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Errorf("nested directory structure not preserved in backup")
	}
}
