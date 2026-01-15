package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CreatePatchBackup backs up a patch file with timestamp to the backup directory
// Backup structure: backup/patched/{workspace|multirepo}/{workspace-path}/{branch}/yyyy/mm/dd/file.yyyymmdd_hhmmss.patch
func CreatePatchBackup(patchPath, backupDir, workspace, branch string) error {
	// Check if patch file exists
	if _, err := os.Stat(patchPath); os.IsNotExist(err) {
		return nil // No patch to backup
	}

	// Generate timestamp
	now := time.Now()
	timestamp := now.Format("20060102_150405")

	// Extract relative path from .multirepos/patches/ (handle both absolute and relative paths)
	relPath := patchPath
	// Try to find .multirepos/patches/ in the path
	if idx := strings.Index(patchPath, ".multirepos/patches/"); idx != -1 {
		relPath = patchPath[idx+len(".multirepos/patches/"):]
	} else if idx := strings.Index(patchPath, ".multirepos-patches/"); idx != -1 {
		// Fallback for old naming
		relPath = patchPath[idx+len(".multirepos-patches/"):]
	}

	// For workspace patches, remove workspace prefix from relPath
	if workspace != "" && strings.HasPrefix(relPath, workspace+"/") {
		relPath = strings.TrimPrefix(relPath, workspace+"/")
	}

	// Determine backup type directory
	var typeDir string
	if workspace == "" {
		// Root repository: backup/patched/workspace/{branch}/{date}/file.ext
		typeDir = filepath.Join(backupDir, "patched", "workspace", branch,
			now.Format("2006"), now.Format("01"), now.Format("02"))
	} else {
		// Singlerepo: backup/patched/multirepo/{workspace}/{branch}/{date}/file.ext
		typeDir = filepath.Join(backupDir, "patched", "multirepo", workspace, branch,
			now.Format("2006"), now.Format("01"), now.Format("02"))
	}

	// NEW: Check if today's backup with identical content exists
	latestBackup := findLatestBackup(typeDir, relPath)

	// NEW: Compare with latest backup
	if latestBackup != "" {
		identical, err := filesIdentical(patchPath, latestBackup)
		if err == nil && identical {
			// Skip: content is identical to latest backup
			return nil
		}
	}

	// Build backup path
	backupPath := filepath.Join(typeDir, relPath)

	// Add timestamp to filename
	dir := filepath.Dir(backupPath)
	base := filepath.Base(backupPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	backupPath = filepath.Join(dir, fmt.Sprintf("%s.%s%s", name, timestamp, ext))

	// Create directory if not exists
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy patch file to backup location
	return copyFile(patchPath, backupPath)
}

// CreateFileBackup backs up the entire file with timestamp
// Backup structure: backup/modified/{workspace|multirepo}/{workspace-path}/{branch}/yyyy/mm/dd/file.yyyymmdd_hhmmss.ext
func CreateFileBackup(filePath, backupDir, repoRoot, workspace, branch string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // No file to backup
	}

	// Generate timestamp
	now := time.Now()
	timestamp := now.Format("20060102_150405")

	// Extract relative path from appropriate base directory
	var baseDir string
	if workspace == "" {
		// Root repository: use repoRoot as base
		baseDir = repoRoot
	} else {
		// Workspace repository: use workspace directory as base
		baseDir = filepath.Join(repoRoot, workspace)
	}

	relPath := filePath
	if filepath.IsAbs(filePath) {
		var err error
		relPath, err = filepath.Rel(baseDir, filePath)
		if err != nil {
			// Fallback: use the original path if we can't make it relative
			relPath = filePath
		}
	}

	// Determine backup type directory
	var typeDir string
	if workspace == "" {
		// Root repository: backup/modified/workspace/{branch}/{date}/file.ext
		typeDir = filepath.Join(backupDir, "modified", "workspace", branch,
			now.Format("2006"), now.Format("01"), now.Format("02"))
	} else {
		// Singlerepo: backup/modified/multirepo/{workspace}/{branch}/{date}/file.ext
		typeDir = filepath.Join(backupDir, "modified", "multirepo", workspace, branch,
			now.Format("2006"), now.Format("01"), now.Format("02"))
	}

	// NEW: Check if today's backup with identical content exists
	latestBackup := findLatestBackup(typeDir, relPath)

	// NEW: Compare with latest backup
	if latestBackup != "" {
		identical, err := filesIdentical(filePath, latestBackup)
		if err == nil && identical {
			// Skip: content is identical to latest backup
			return nil
		}
	}

	// Build backup path
	backupPath := filepath.Join(typeDir, relPath)

	// Add timestamp to filename
	dir := filepath.Dir(backupPath)
	base := filepath.Base(backupPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	backupPath = filepath.Join(dir, fmt.Sprintf("%s.%s%s", name, timestamp, ext))

	// Create directory if not exists
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy file to backup location
	return copyFile(filePath, backupPath)
}

// Cleanup removes backups older than specified days
func Cleanup(backupDir string, days int) error {
	if days <= 0 {
		return fmt.Errorf("days must be positive")
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)

	return filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is older than cutoff
		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove old backup %s: %w", path, err)
			}
		}

		return nil
	})
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// sha256File calculates SHA256 hash of a file
func sha256File(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// filesIdentical checks if two files have identical content using SHA256
func filesIdentical(file1, file2 string) (bool, error) {
	hash1, err := sha256File(file1)
	if err != nil {
		return false, err
	}

	hash2, err := sha256File(file2)
	if err != nil {
		return false, err
	}

	return hash1 == hash2, nil
}

// findLatestBackup finds the most recent backup of a file in today's directory
// Returns empty string if no backup found
func findLatestBackup(todayDir, relPath string) string {
	// Build pattern: todayDir/relPath_without_ext.*.ext
	targetDir := filepath.Join(todayDir, filepath.Dir(relPath))
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return "" // Directory doesn't exist
	}

	base := filepath.Base(relPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	// List all files in directory
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return ""
	}

	// Find files matching pattern: nameWithoutExt.TIMESTAMP.ext
	var matches []string
	prefix := nameWithoutExt + "."

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check if it matches: name.YYYYMMDD_HHMMSS.ext
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ext) {
			matches = append(matches, filepath.Join(targetDir, name))
		}
	}

	if len(matches) == 0 {
		return ""
	}

	// Sort by filename (timestamp is in filename) - latest last
	sort.Strings(matches)
	return matches[len(matches)-1]
}
