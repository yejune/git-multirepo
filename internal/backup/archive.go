package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ArchiveOldBackups archives previous month backups to tar.gz and removes originals
// Archives structure:
//   - archived/{modified|patched}/workspace/YYYY-MM-{branch}.tar.gz
//   - archived/{modified|patched}/multirepo/{workspace}/YYYY-MM-{branch}.tar.gz
// Only previous months are archived, current month is preserved
func ArchiveOldBackups(backupDir string) error {
	now := time.Now()
	currentYear := now.Format("2006")
	currentMonth := now.Format("01")

	fmt.Println("\n[Archive] Starting automatic backup archiving...")
	fmt.Printf("  Current month: %s/%s (will be skipped)\n", currentYear, currentMonth)

	totalArchived := 0
	totalSkipped := 0

	// Process modified backups
	archived, skipped, err := archiveBackupType(backupDir, "modified", currentYear, currentMonth)
	if err != nil {
		return fmt.Errorf("failed to archive modified backups: %w", err)
	}
	totalArchived += archived
	totalSkipped += skipped

	// Process patched backups
	archived, skipped, err = archiveBackupType(backupDir, "patched", currentYear, currentMonth)
	if err != nil {
		return fmt.Errorf("failed to archive patched backups: %w", err)
	}
	totalArchived += archived
	totalSkipped += skipped

	fmt.Printf("[Archive] Completed: %d total archive(s) created, %d month(s) skipped (current)\n",
		totalArchived, totalSkipped)
	return nil
}

// archiveBackupType archives a specific backup type (modified or patched)
// New structure: backup/{type}/{workspace|multirepo}/{path}/{branch}/{year}/{month}
// Returns: (archived count, skipped count, error)
func archiveBackupType(backupDir, backupType, currentYear, currentMonth string) (int, int, error) {
	typeDir := filepath.Join(backupDir, backupType)

	fmt.Printf("\n  [Archive] Processing '%s' backups...\n", backupType)

	// Check if directory exists
	if _, err := os.Stat(typeDir); os.IsNotExist(err) {
		fmt.Printf("  [Archive] No '%s' backups directory found\n", backupType)
		return 0, 0, nil // No backups to archive
	}

	totalArchived := 0
	totalSkipped := 0

	// Process both workspace and multirepo directories
	for _, topLevel := range []string{"workspace", "multirepo"} {
		topLevelDir := filepath.Join(typeDir, topLevel)

		// Check if directory exists
		if _, err := os.Stat(topLevelDir); os.IsNotExist(err) {
			continue // Skip if doesn't exist
		}

		var archived, skipped int
		var err error
		if topLevel == "workspace" {
			// For workspace: directly access branch directories
			archived, skipped, err = archiveWorkspaceBackups(topLevelDir, backupDir, backupType, currentYear, currentMonth)
		} else {
			// For multirepo: traverse singlerepo directories first
			archived, skipped, err = archiveMultirepoBackups(topLevelDir, backupDir, backupType, currentYear, currentMonth)
		}

		if err != nil {
			return totalArchived, totalSkipped, fmt.Errorf("failed to archive %s backups: %w", topLevel, err)
		}

		totalArchived += archived
		totalSkipped += skipped
	}

	return totalArchived, totalSkipped, nil
}

// formatSize formats byte size to human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// archiveWorkspaceBackups handles workspace (root repository) backups
// Structure: backup/{type}/workspace/{branch}/{year}/{month}/
// Returns: (archived count, skipped count, error)
func archiveWorkspaceBackups(workspaceDir, backupDir, backupType, currentYear, currentMonth string) (int, int, error) {
	// Get all branch directories
	branches, err := os.ReadDir(workspaceDir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	archivedCount := 0
	skippedCount := 0

	for _, branchEntry := range branches {
		if !branchEntry.IsDir() {
			continue
		}

		branch := branchEntry.Name()
		branchPath := filepath.Join(workspaceDir, branch)

		// Get all year directories
		years, err := os.ReadDir(branchPath)
		if err != nil {
			fmt.Printf("  [Archive] Warning: failed to read branch %s: %v\n", branch, err)
			continue
		}

		for _, yearEntry := range years {
			if !yearEntry.IsDir() {
				continue
			}

			year := yearEntry.Name()
			yearPath := filepath.Join(branchPath, year)

			// Get all month directories
			months, err := os.ReadDir(yearPath)
			if err != nil {
				fmt.Printf("  [Archive] Warning: failed to read year %s/%s: %v\n", branch, year, err)
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
					skippedCount++
					continue
				}

				// Create archive name: {year}-{month}-{branch}.tar.gz
				// Replace / in branch name with _
				safeBranch := filepath.ToSlash(branch)
				safeBranch = filepath.Base(safeBranch) // Use only the last part if it's a path
				if branch != safeBranch {
					safeBranch = branch
				}
				safeBranch = sanitizePath(safeBranch)

				archiveName := fmt.Sprintf("%s-%s-%s.tar.gz", year, month, safeBranch)
				archivePath := filepath.Join(backupDir, "archived", backupType, "workspace", archiveName)

				// Check if archive already exists
				if _, err := os.Stat(archivePath); err == nil {
					fmt.Printf("  [Archive] Already exists: %s\n", archiveName)
					continue
				}

				// Create archived directory structure if not exists
				archiveDir := filepath.Dir(archivePath)
				if err := os.MkdirAll(archiveDir, 0755); err != nil {
					return 0, 0, fmt.Errorf("failed to create archive directory: %w", err)
				}

				// Create tar.gz archive from monthPath
				if err := createTarGzFromDir(monthPath, archivePath); err != nil {
					return 0, 0, fmt.Errorf("failed to create archive %s: %w", archiveName, err)
				}

				// Verify archive
				if err := verifyTarGz(archivePath); err != nil {
					os.Remove(archivePath)
					return 0, 0, fmt.Errorf("archive verification failed for %s: %w", archiveName, err)
				}

				// Get archive file size
				fileInfo, err := os.Stat(archivePath)
				var size int64
				if err == nil {
					size = fileInfo.Size()
				}

				// Get relative path for display
				relPath, err := filepath.Rel(backupDir, archivePath)
				if err != nil {
					relPath = archivePath
				}

				fmt.Printf("  [Archive] ✓ Created: %s (size: %s)\n", relPath, formatSize(size))

				// Remove original directory
				if err := os.RemoveAll(monthPath); err != nil {
					return 0, 0, fmt.Errorf("failed to remove original directory %s: %w", monthPath, err)
				}

				archivedCount++

				// Clean up empty year directory
				remaining, err := os.ReadDir(yearPath)
				if err == nil && len(remaining) == 0 {
					os.Remove(yearPath)
					// Clean up empty branch directory
					remaining, err = os.ReadDir(branchPath)
					if err == nil && len(remaining) == 0 {
						os.Remove(branchPath)
					}
				}
			}
		}
	}

	if archivedCount > 0 {
		fmt.Printf("  [Archive] ✓ %d workspace archive(s) created for %s\n", archivedCount, backupType)
	} else if skippedCount > 0 {
		fmt.Printf("  [Archive] No old workspace backups to archive for %s (only current month)\n", backupType)
	}

	return archivedCount, skippedCount, nil
}

// archiveMultirepoBackups handles multirepo (singlerepo) backups
// Structure: backup/{type}/multirepo/{workspace}/{branch}/{year}/{month}/
// Returns: (archived count, skipped count, error)
func archiveMultirepoBackups(multirepoDir, backupDir, backupType, currentYear, currentMonth string) (int, int, error) {
	// Get all workspace directories
	workspaces, err := os.ReadDir(multirepoDir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read multirepo directory: %w", err)
	}

	archivedCount := 0
	skippedCount := 0

	for _, workspaceEntry := range workspaces {
		if !workspaceEntry.IsDir() {
			continue
		}

		workspace := workspaceEntry.Name()
		workspacePath := filepath.Join(multirepoDir, workspace)

		// Get all branch directories
		branches, err := os.ReadDir(workspacePath)
		if err != nil {
			fmt.Printf("  [Archive] Warning: failed to read workspace %s: %v\n", workspace, err)
			continue
		}

		for _, branchEntry := range branches {
			if !branchEntry.IsDir() {
				continue
			}

			branch := branchEntry.Name()
			branchPath := filepath.Join(workspacePath, branch)

			// Get all year directories
			years, err := os.ReadDir(branchPath)
			if err != nil {
				fmt.Printf("  [Archive] Warning: failed to read branch %s/%s: %v\n", workspace, branch, err)
				continue
			}

			for _, yearEntry := range years {
				if !yearEntry.IsDir() {
					continue
				}

				year := yearEntry.Name()
				yearPath := filepath.Join(branchPath, year)

				// Get all month directories
				months, err := os.ReadDir(yearPath)
				if err != nil {
					fmt.Printf("  [Archive] Warning: failed to read year %s/%s/%s: %v\n", workspace, branch, year, err)
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
						skippedCount++
						continue
					}

					// Create archive name: {year}-{month}-{branch}.tar.gz
					// Replace / in workspace and branch names with _
					safeWorkspace := sanitizePath(workspace)
					safeBranch := sanitizePath(branch)

					archiveName := fmt.Sprintf("%s-%s-%s.tar.gz", year, month, safeBranch)
					archivePath := filepath.Join(backupDir, "archived", backupType, "multirepo", safeWorkspace, archiveName)

					// Check if archive already exists
					if _, err := os.Stat(archivePath); err == nil {
						fmt.Printf("  [Archive] Already exists: %s\n", archiveName)
						continue
					}

					// Create archived directory structure if not exists
					archiveDir := filepath.Dir(archivePath)
					if err := os.MkdirAll(archiveDir, 0755); err != nil {
						return 0, 0, fmt.Errorf("failed to create archive directory: %w", err)
					}

					// Create tar.gz archive from monthPath
					if err := createTarGzFromDir(monthPath, archivePath); err != nil {
						return 0, 0, fmt.Errorf("failed to create archive %s: %w", archiveName, err)
					}

					// Verify archive
					if err := verifyTarGz(archivePath); err != nil {
						os.Remove(archivePath)
						return 0, 0, fmt.Errorf("archive verification failed for %s: %w", archiveName, err)
					}

					// Get archive file size
					fileInfo, err := os.Stat(archivePath)
					var size int64
					if err == nil {
						size = fileInfo.Size()
					}

					// Get relative path for display
					relPath, err := filepath.Rel(backupDir, archivePath)
					if err != nil {
						relPath = archivePath
					}

					fmt.Printf("  [Archive] ✓ Created: %s (size: %s)\n", relPath, formatSize(size))

					// Remove original directory
					if err := os.RemoveAll(monthPath); err != nil {
						return 0, 0, fmt.Errorf("failed to remove original directory %s: %w", monthPath, err)
					}

					archivedCount++

					// Clean up empty year directory
					remaining, err := os.ReadDir(yearPath)
					if err == nil && len(remaining) == 0 {
						os.Remove(yearPath)
						// Clean up empty branch directory
						remaining, err = os.ReadDir(branchPath)
						if err == nil && len(remaining) == 0 {
							os.Remove(branchPath)
							// Clean up empty workspace directory
							remaining, err = os.ReadDir(workspacePath)
							if err == nil && len(remaining) == 0 {
								os.Remove(workspacePath)
							}
						}
					}
				}
			}
		}
	}

	if archivedCount > 0 {
		fmt.Printf("  [Archive] ✓ %d multirepo archive(s) created for %s\n", archivedCount, backupType)
	} else if skippedCount > 0 {
		fmt.Printf("  [Archive] No old multirepo backups to archive for %s (only current month)\n", backupType)
	}

	return archivedCount, skippedCount, nil
}

// sanitizePath replaces / with _ in path segments for safe filenames
func sanitizePath(path string) string {
	// Replace / and \ with _
	result := filepath.ToSlash(path)
	result = filepath.Base(result)
	if path != result {
		// It was a path, use the full sanitized version
		result = path
	}
	// Replace any remaining slashes
	result = filepath.ToSlash(result)
	if len(result) > 0 && (result[0] == '/' || result[0] == '\\') {
		result = result[1:]
	}
	// Simple replacement for safety
	for _, char := range []string{"/", "\\"} {
		result = replaceAll(result, char, "_")
	}
	return result
}

// replaceAll is a simple string replacement helper
func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}

// createTarGzFromDir creates a tar.gz archive from a directory
// Archives the entire contents of srcDir into archivePath
func createTarGzFromDir(srcDir, archivePath string) error {
	// Create archive file
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer archiveFile.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(archiveFile)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk directory and add files to tar
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from srcDir
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", path, err)
		}

		// Use relative path as name
		header.Name = filepath.ToSlash(relPath)

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// Write file contents if not directory
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to write file %s to tar: %w", path, err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}

	return nil
}

// createTarGz creates a tar.gz archive from a directory using native Go (legacy)
// This function is kept for backward compatibility but is no longer used
func createTarGz(baseDir, archivePath, year, month string) error {
	srcDir := filepath.Join(baseDir, year, month)

	// Create archive file
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer archiveFile.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(archiveFile)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk directory and add files to tar
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from baseDir (preserves YYYY/MM structure)
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", path, err)
		}

		// Use relative path as name
		header.Name = filepath.ToSlash(relPath)

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// Write file contents if not directory
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to write file %s to tar: %w", path, err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}

	return nil
}

// verifyTarGz verifies the integrity of a tar.gz archive using native Go
func verifyTarGz(archivePath string) error {
	// Open archive file
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer archiveFile.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Read all entries to verify archive structure
	fileCount := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		fileCount++

		// Try to read file contents to verify integrity
		if !header.FileInfo().IsDir() {
			// Just consume the data without storing it
			if _, err := io.Copy(io.Discard, tarReader); err != nil {
				return fmt.Errorf("failed to verify file %s: %w", header.Name, err)
			}
		}
	}

	if fileCount == 0 {
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
