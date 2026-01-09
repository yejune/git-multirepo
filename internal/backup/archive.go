package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ArchiveOldBackups archives previous month backups to tar.gz and removes originals
// Archives are saved as: archived/YYYY-MM-{modified|patched}.tar.gz
// Only previous months are archived, current month is preserved
func ArchiveOldBackups(backupDir string) error {
	now := time.Now()
	currentYear := now.Format("2006")
	currentMonth := now.Format("01")

	fmt.Println("\n[Archive] Checking for old backups to archive...")

	// Process modified backups
	if err := archiveBackupType(backupDir, "modified", currentYear, currentMonth); err != nil {
		return fmt.Errorf("failed to archive modified backups: %w", err)
	}

	// Process patched backups
	if err := archiveBackupType(backupDir, "patched", currentYear, currentMonth); err != nil {
		return fmt.Errorf("failed to archive patched backups: %w", err)
	}

	fmt.Println("[Archive] Completed")
	return nil
}

// archiveBackupType archives a specific backup type (modified or patched)
func archiveBackupType(backupDir, backupType, currentYear, currentMonth string) error {
	typeDir := filepath.Join(backupDir, backupType)

	// Check if directory exists
	if _, err := os.Stat(typeDir); os.IsNotExist(err) {
		return nil // No backups to archive
	}

	// Get all year directories
	years, err := os.ReadDir(typeDir)
	if err != nil {
		return fmt.Errorf("failed to read %s directory: %w", backupType, err)
	}

	archivedCount := 0

	for _, yearEntry := range years {
		if !yearEntry.IsDir() {
			continue
		}

		year := yearEntry.Name()
		yearPath := filepath.Join(typeDir, year)

		// Get all month directories
		months, err := os.ReadDir(yearPath)
		if err != nil {
			fmt.Printf("  [Archive] Warning: failed to read year %s: %v\n", year, err)
			continue
		}

		for _, monthEntry := range months {
			if !monthEntry.IsDir() {
				continue
			}

			month := monthEntry.Name()
			monthPath := filepath.Join(yearPath, month)

			// Skip current month
			if year == currentYear && month == currentMonth {
				fmt.Printf("  [Archive] Skipping current month: %s/%s\n", year, month)
				continue
			}

			// Archive this month
			archiveName := fmt.Sprintf("%s-%s-%s.tar.gz", year, month, backupType)
			archivePath := filepath.Join(backupDir, "archived", archiveName)

			// Check if archive already exists
			if _, err := os.Stat(archivePath); err == nil {
				fmt.Printf("  [Archive] Already exists: %s\n", archiveName)
				continue
			}

			fmt.Printf("  [Archive] Archiving %s/%s/%s -> %s\n", backupType, year, month, archiveName)

			// Create archived directory if not exists
			archivedDir := filepath.Join(backupDir, "archived")
			if err := os.MkdirAll(archivedDir, 0755); err != nil {
				return fmt.Errorf("failed to create archived directory: %w", err)
			}

			// Create tar.gz archive
			// Use relative path from typeDir to preserve directory structure
			if err := createTarGz(typeDir, archivePath, year, month); err != nil {
				return fmt.Errorf("failed to create archive %s: %w", archiveName, err)
			}

			// Verify archive
			if err := verifyTarGz(archivePath); err != nil {
				// Remove corrupted archive
				os.Remove(archivePath)
				return fmt.Errorf("archive verification failed for %s: %w", archiveName, err)
			}

			fmt.Printf("  [Archive] Verified: %s\n", archiveName)

			// Remove original directory only after successful archive and verification
			if err := os.RemoveAll(monthPath); err != nil {
				return fmt.Errorf("failed to remove original directory %s: %w", monthPath, err)
			}

			fmt.Printf("  [Archive] Removed original: %s/%s/%s\n", backupType, year, month)
			archivedCount++

			// Clean up empty year directory
			remaining, err := os.ReadDir(yearPath)
			if err == nil && len(remaining) == 0 {
				os.Remove(yearPath)
			}
		}
	}

	if archivedCount > 0 {
		fmt.Printf("  [Archive] Archived %d month(s) for %s\n", archivedCount, backupType)
	}

	return nil
}

// createTarGz creates a tar.gz archive from a directory
func createTarGz(baseDir, archivePath, year, month string) error {
	// Change to base directory and tar relative path
	// This preserves the YYYY/MM directory structure in the archive
	cmd := exec.Command("tar", "-czf", archivePath, "-C", baseDir, filepath.Join(year, month))

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// verifyTarGz verifies the integrity of a tar.gz archive
func verifyTarGz(archivePath string) error {
	cmd := exec.Command("tar", "-tzf", archivePath)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("verification command failed: %w\nOutput: %s", err, string(output))
	}

	// Check if output contains files
	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("archive is empty")
	}

	return nil
}

// ShouldRunArchive checks if archiving should run (24 hours since last check)
func ShouldRunArchive(workspacesDir string) bool {
	checkFile := filepath.Join(workspacesDir, ".last-archive-check")

	// Check if file exists
	info, err := os.Stat(checkFile)
	if err != nil {
		// File doesn't exist, should run
		return true
	}

	// Check if 24 hours have passed
	lastCheck := info.ModTime()
	elapsed := time.Since(lastCheck)

	return elapsed >= 24*time.Hour
}

// UpdateArchiveCheck updates the last archive check timestamp
func UpdateArchiveCheck(workspacesDir string) error {
	checkFile := filepath.Join(workspacesDir, ".last-archive-check")

	// Create or update the file
	file, err := os.Create(checkFile)
	if err != nil {
		return fmt.Errorf("failed to create check file: %w", err)
	}
	defer file.Close()

	// Write current timestamp
	timestamp := time.Now().Format(time.RFC3339)
	if _, err := file.WriteString(timestamp + "\n"); err != nil {
		return fmt.Errorf("failed to write timestamp: %w", err)
	}

	return nil
}
