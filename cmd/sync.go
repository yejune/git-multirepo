package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/yejune/git-multirepo/internal/backup"
	"github.com/yejune/git-multirepo/internal/common"
	"github.com/yejune/git-multirepo/internal/git"
	"github.com/yejune/git-multirepo/internal/hooks"
	"github.com/yejune/git-multirepo/internal/i18n"
	"github.com/yejune/git-multirepo/internal/manifest"
	"github.com/yejune/git-multirepo/internal/patch"
	"golang.org/x/sync/errgroup"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Clone missing workspaces and apply configurations",
	Long: `Sync all workspaces from .git.multirepos manifest:
  - Clone missing workspaces automatically
  - Install git hooks if not present
  - Apply ignore patterns to .gitignore
  - Apply skip-worktree to specified files
  - Verify .gitignore entries for workspaces

Examples:
  git multirepo sync`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	// Use common context loading pattern
	ctx, err := common.LoadWorkspaceContext()
	if err != nil {
		return err
	}

	fmt.Println(i18n.T("syncing"))

	// 1. Auto-install hooks
	if !hooks.IsInstalled(ctx.RepoRoot) {
		fmt.Println(i18n.T("installing_hooks"))
		if err := hooks.Install(ctx.RepoRoot); err != nil {
			fmt.Printf("  %s\n", i18n.T("hooks_failed", err))
		} else {
			fmt.Printf("  %s\n", i18n.T("hooks_installed"))
		}
	}

	// 2. If no workspaces in manifest, scan for existing sub repos
	// Use ScanRootDir instead of RepoRoot to support workspace subdirectory sync
	if len(ctx.Manifest.Workspaces) == 0 {
		fmt.Println(i18n.T("no_gitsubs_found"))
		discovered, scanErr := scanForWorkspaces(ctx.ScanRootDir, ctx.RepoRoot)
		if scanErr != nil {
			return fmt.Errorf(i18n.T("failed_scan"), scanErr)
		}

		if len(discovered) > 0 {
			// Create manifest from discovered workspaces
			ctx.Manifest = &manifest.Manifest{
				Workspaces: discovered,
				Ignore:     ctx.Manifest.Ignore, // Preserve ignore patterns
				Keep:       ctx.Manifest.Keep,   // Preserve keep files
			}

			if err := ctx.SaveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			fmt.Print(i18n.T("created_gitsubs", len(discovered)))
			for _, ws := range discovered {
				fmt.Printf("  - %s (%s)\n", ws.Path, ws.Repo)
			}
		} else {
			fmt.Println(i18n.T("no_subs_found"))
			fmt.Println(i18n.T("to_add_sub"))
			fmt.Println(i18n.T("cmd_git_sub_clone"))
			// Don't return - continue to apply ignore patterns and keep files
		}
	}

	// 3. Apply ignore patterns to mother repo
	if len(ctx.Manifest.Ignore) > 0 {
		fmt.Println(i18n.T("applying_ignore"))
		if err := git.AddIgnorePatternsToGitignore(ctx.RepoRoot, ctx.Manifest.Ignore); err != nil {
			fmt.Printf("  %s\n", i18n.T("hooks_failed", err))
		} else {
			fmt.Printf("  %s\n", i18n.T("applied_patterns", len(ctx.Manifest.Ignore)))
		}
	}

	// 4. Process Mother repo keep files
	issues := 0
	motherKeepFiles := ctx.Manifest.Keep
	if len(motherKeepFiles) > 0 {
		fmt.Printf("\n%s\n", i18n.T("processing_mother_keep"))
		processKeepFiles(ctx.RepoRoot, ctx.RepoRoot, motherKeepFiles, &issues)
	}

	if len(ctx.Manifest.Workspaces) == 0 {
		fmt.Println(i18n.T("no_subclones"))
		return nil
	}

	// 5. Process each workspace
	fmt.Println(i18n.T("processing_subclones"))

	for _, ws := range ctx.Manifest.Workspaces {
		fullPath := filepath.Join(ctx.RepoRoot, ws.Path)
		fmt.Printf("\n  %s\n", ws.Path)

		// Check if workspace exists
		if !git.IsRepo(fullPath) {
			// Check if directory has files (parent is tracking source)
			entries, err := os.ReadDir(fullPath)
			if err == nil && len(entries) > 0 {
				// Directory exists with files - init git in place
				fmt.Printf("    %s\n", i18n.T("initializing_git"))

				if err := git.InitRepo(fullPath, ws.Repo, ws.Branch); err != nil {
					fmt.Printf("    %s\n", i18n.T("failed_initialize", err))
					issues++
					continue
				}

				// Add to .gitignore
				if err := git.AddToGitignore(ctx.RepoRoot, ws.Path); err != nil {
					fmt.Printf("    %s\n", i18n.T("failed_update_gitignore", err))
				}

				fmt.Printf("    %s\n", i18n.T("initialized_git"))
				continue
			}

			// Directory empty or doesn't exist - clone normally
			fmt.Printf("    %s\n", i18n.T("cloning_from", ws.Repo))

			// Create parent directory if needed
			parentDir := filepath.Dir(fullPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				fmt.Printf("    %s\n", i18n.T("failed_create_dir", err))
				issues++
				continue
			}

			// Clone the repository
			if err := git.Clone(ws.Repo, fullPath, ws.Branch); err != nil {
				fmt.Printf("    %s\n", i18n.T("clone_failed", err))
				issues++
				continue
			}

			// Add to .gitignore
			if err := git.AddToGitignore(ctx.RepoRoot, ws.Path); err != nil {
				fmt.Printf("    %s\n", i18n.T("failed_update_gitignore", err))
			}

			fmt.Printf("    %s\n", i18n.T("cloned_successfully"))
			continue
		}

		// Verify and fix .gitignore entry
		if !hasGitignoreEntry(ctx.RepoRoot, ws.Path) {
			fmt.Printf("    %s\n", i18n.T("adding_to_gitignore"))
			if err := git.AddToGitignore(ctx.RepoRoot, ws.Path); err != nil {
				fmt.Printf("    %s\n", i18n.T("hooks_failed", err))
				issues++
			} else {
				fmt.Printf("    %s\n", i18n.T("added_to_gitignore"))
			}
		}

		// Process keep files for this workspace
		keepFiles := ws.Keep
		if len(keepFiles) > 0 {
			fmt.Printf("    %s\n", i18n.T("processing_keep_files", len(keepFiles)))
			processKeepFiles(ctx.RepoRoot, fullPath, keepFiles, &issues)
		}

		// Install/update post-commit hook in workspace
		if !hooks.IsWorkspaceHookInstalled(fullPath) {
			fmt.Printf("    %s\n", i18n.T("installing_hook"))
			if err := hooks.InstallWorkspaceHook(fullPath); err != nil {
				fmt.Printf("    %s\n", i18n.T("hook_failed", err))
			} else {
				fmt.Printf("    %s\n", i18n.T("hook_installed"))
			}
		}
	}

	// Save manifest if any commits were updated
	if err := ctx.SaveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	// 6. Check if archiving should run (24 hours check)
	multireposDir := filepath.Join(ctx.RepoRoot, ".multirepos")
	if backup.ShouldRunArchive(multireposDir) {
		backupDir := filepath.Join(multireposDir, "backup")
		if err := backup.ArchiveOldBackups(backupDir); err != nil {
			fmt.Printf("\n⚠️  Archive failed: %v\n", err)
			// Don't fail the entire sync if archiving fails
		} else {
			// Update check time only on success
			if err := backup.UpdateArchiveCheck(multireposDir); err != nil {
				fmt.Printf("\n⚠️  Failed to update archive check time: %v\n", err)
			}
		}
	}

	// Summary
	fmt.Println()
	if issues > 0 {
		fmt.Println(i18n.T("completed_issues", issues))
	} else {
		fmt.Println(i18n.T("all_success"))
	}

	return nil
}

func hasGitignoreEntry(repoRoot, path string) bool {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return false
	}

	expected := path + "/.git/"
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == expected {
			return true
		}
	}
	return false
}

// workspaceDiscovery represents a discovered workspace location
type workspaceDiscovery struct {
	path    string
	relPath string
}

// discoverWorkspaces sequentially scans for .git directories and sends them to a channel
// scanRoot: directory to start scanning from
// manifestRoot: parent directory containing .git.multirepo (for calculating relative paths)
func discoverWorkspaces(scanRoot, manifestRoot string) (<-chan workspaceDiscovery, error) {
	discoveries := make(chan workspaceDiscovery, 100)

	go func() {
		defer close(discoveries)

		// Walk the directory tree starting from scanRoot
		filepath.Walk(scanRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			// Skip parent's .git directory
			if path == filepath.Join(manifestRoot, ".git") {
				return filepath.SkipDir
			}

			// Check if this is a .git directory
			if !info.IsDir() || info.Name() != ".git" {
				return nil
			}

			// Get the repository path (parent of .git)
			workspacePath := filepath.Dir(path)

			// Skip if it's the parent repo itself
			if workspacePath == manifestRoot {
				return filepath.SkipDir
			}

			// Get relative path from manifest root (not scan root)
			relPath, err := filepath.Rel(manifestRoot, workspacePath)
			if err != nil {
				return nil
			}

			// Send discovery to channel
			discoveries <- workspaceDiscovery{
				path:    workspacePath,
				relPath: relPath,
			}

			// Skip descending into this workspace's subdirectories
			return filepath.SkipDir
		})
	}()

	return discoveries, nil
}

// getOptimalWorkerCount determines the optimal number of workers for parallel processing
// Priority order:
// 1. GIT_MULTIREPO_WORKERS environment variable (if valid)
// 2. CPU cores * 2 (I/O bound operations benefit from higher concurrency)
// Constraints: min=1, max=32 (prevent context switching overhead)
func getOptimalWorkerCount() int {
	// Check environment variable
	if envWorkers := os.Getenv("GIT_MULTIREPO_WORKERS"); envWorkers != "" {
		if workers, err := strconv.Atoi(envWorkers); err == nil && workers > 0 {
			// Apply max constraint
			if workers > 32 {
				return 32
			}
			return workers
		}
		// Invalid value in env var, fall through to default
	}

	// CPU-based default: NumCPU * 2 for I/O-bound workloads
	workers := runtime.NumCPU() * 2

	// Apply constraints
	if workers < 1 {
		workers = 1
	}
	if workers > 32 {
		workers = 32
	}

	return workers
}

// processWorkspacesParallel processes discovered workspaces in parallel using a worker pool
func processWorkspacesParallel(ctx context.Context, discoveries <-chan workspaceDiscovery) ([]manifest.WorkspaceEntry, error) {
	return processWorkspacesParallelWithWorkers(ctx, discoveries, getOptimalWorkerCount())
}

// processWorkspacesParallelWithWorkers processes workspaces with configurable worker count
func processWorkspacesParallelWithWorkers(ctx context.Context, discoveries <-chan workspaceDiscovery, numWorkers int) ([]manifest.WorkspaceEntry, error) {
	var mu sync.Mutex
	var workspaces []manifest.WorkspaceEntry

	eg, ctx := errgroup.WithContext(ctx)

	// Semaphore for worker pool
	sem := make(chan struct{}, numWorkers)

	for discovery := range discoveries {
		d := discovery // Capture loop variable
		eg.Go(func() error {
			// Acquire worker
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			defer func() { <-sem }() // Release worker

			// Extract git info
			repo, err := git.GetRemoteURL(d.path)
			if err != nil {
				// Warning only - continue processing workspace with empty remote
				fmt.Printf("⚠ %s\n", i18n.T("warn_no_remote", d.relPath))
				repo = "" // Empty remote is valid for local-only repos
			}

			// Detect modified files for auto-keep
			var keepFiles []string
			// Get skip-worktree files (these are the keep files)
			skipFiles, err := git.ListSkipWorktree(d.path)
			if err == nil && len(skipFiles) > 0 {
				keepFiles = skipFiles
			} else {
				// Fallback: detect modified files for first-time setup
				var modifiedFiles []string
				git.WithSkipWorktreeTransaction(d.path, []string{}, func() error {
					var err error
					modifiedFiles, err = git.GetModifiedFiles(d.path)
					return err
				})
				if len(modifiedFiles) > 0 {
					// Clean up file list
					for _, file := range modifiedFiles {
						if strings.TrimSpace(file) != "" {
							keepFiles = append(keepFiles, file)
						}
					}
				}
			}

			fmt.Printf("  %s\n", i18n.T("found_sub", d.relPath))

			// Thread-safe append
			mu.Lock()
			workspaces = append(workspaces, manifest.WorkspaceEntry{
				Path: d.relPath,
				Repo: repo,
				Keep: keepFiles,
			})
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return workspaces, nil
}

// scanForWorkspaces recursively scans directories for git repositories using parallel processing
// scanRoot: directory to start scanning from
// manifestRoot: parent directory containing .git.multirepo (for calculating relative paths)
func scanForWorkspaces(scanRoot, manifestRoot string) ([]manifest.WorkspaceEntry, error) {
	ctx := context.Background()

	// Phase 1: Discover workspaces sequentially
	discoveries, err := discoverWorkspaces(scanRoot, manifestRoot)
	if err != nil {
		return nil, err
	}

	// Phase 2: Process workspaces in parallel
	workspaces, err := processWorkspacesParallel(ctx, discoveries)
	if err != nil {
		return nil, err
	}

	return workspaces, nil
}

// processKeepFiles handles backup, patch creation, and skip-worktree for keep files
func processKeepFiles(repoRoot, workspacePath string, keepFiles []string, issues *int) {
	backupDir := filepath.Join(repoRoot, ".multirepos", "backup")
	patchBaseDir := filepath.Join(repoRoot, ".multirepos", "patches")

	// Determine workspace path for patches and backups
	relPath, err := filepath.Rel(repoRoot, workspacePath)
	if err != nil {
		relPath = filepath.Base(workspacePath)
	}
	if relPath == "." {
		relPath = ""
	}

	// Clean slate strategy: Remove directories before saving to prevent file leakage

	// 1. Clean patches directory (complete workspace patch dir) - 최신 상태만 유지
	patchDir := filepath.Join(patchBaseDir, relPath)
	os.RemoveAll(patchDir)
	os.MkdirAll(patchDir, 0755)

	// 2. Prepare today's backup directories (타임스탬프 기반 누적, 삭제 금지)
	today := time.Now().Format("2006/01/02")

	// Ensure today's modified backup directory exists (누적)
	modifiedDir := filepath.Join(backupDir, "modified", today, relPath)
	os.MkdirAll(modifiedDir, 0755)

	// Ensure today's patched backup directory exists (누적)
	patchedDir := filepath.Join(backupDir, "patched", today, relPath)
	os.MkdirAll(patchedDir, 0755)

	// 3. Process ALL modified files within a single transaction
	var modifiedFiles []string
	err = git.WithSkipWorktreeTransaction(workspacePath, keepFiles, func() error {
		// 3a. Get modified files
		var err error
		modifiedFiles, err = git.GetModifiedFiles(workspacePath)
		if err != nil {
			return err
		}

		// Remove empty strings from the list
		var cleanModifiedFiles []string
		for _, file := range modifiedFiles {
			if strings.TrimSpace(file) != "" {
				cleanModifiedFiles = append(cleanModifiedFiles, file)
			}
		}
		modifiedFiles = cleanModifiedFiles

		// 3b. Auto-populate Keep list if empty and there are modified files
		if len(keepFiles) == 0 && len(modifiedFiles) > 0 {
			// Load manifest to update it
			m, loadErr := manifest.Load(repoRoot)
			if loadErr != nil {
				return fmt.Errorf("failed to load manifest: %w", loadErr)
			}

			// Update the keep list in manifest
			if relPath == "" || relPath == "." {
				// Mother repo
				m.Keep = modifiedFiles
			} else {
				// Workspace entry
				for i := range m.Workspaces {
					if m.Workspaces[i].Path == relPath {
						m.Workspaces[i].Keep = modifiedFiles
						break
					}
				}
			}

			// Save manifest
			if saveErr := manifest.Save(repoRoot, m); saveErr != nil {
				return fmt.Errorf("failed to save manifest: %w", saveErr)
			}

			// Update keepFiles for this run (will be re-applied by defer)
			keepFiles = modifiedFiles

			fmt.Printf("\n✓ Found %d modified files and added to keep list:\n", len(modifiedFiles))
			for _, f := range modifiedFiles {
				fmt.Printf("  - %s\n", f)
			}
			fmt.Println("\nEdit .git.multirepos to keep only the files you need")
		}

		// 3c. Process ALL modified files (backup + patch for all)
		for _, file := range modifiedFiles {
			filePath := filepath.Join(workspacePath, file)

			// Check if file exists
			if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
				continue // Skip if file doesn't exist
			}

			// Backup original file to backup/modified/
			if backupErr := backup.CreateFileBackup(filePath, backupDir, repoRoot); backupErr != nil {
				fmt.Printf("        Failed to backup %s: %v\n", file, backupErr)
				*issues++
				continue
			}

			// Create patch (git diff HEAD file)
			patchPath := filepath.Join(patchBaseDir, relPath, file+".patch")
			if patchErr := patch.Create(workspacePath, file, patchPath); patchErr != nil {
				fmt.Printf("        Failed to create patch for %s: %v\n", file, patchErr)
				*issues++
				continue
			}

			// Backup patch to backup/patched/
			if patchBackupErr := backup.CreatePatchBackup(patchPath, backupDir); patchBackupErr != nil {
				fmt.Printf("        Failed to backup patch for %s: %v\n", file, patchBackupErr)
				*issues++
				continue
			}
		}

		return nil
	})
	if err != nil {
		fmt.Printf("        Failed to process keep files: %v\n", err)
		*issues++
		return
	}

	// Note: defer in WithSkipWorktreeTransaction automatically re-applies skip-worktree to keepFiles

	// Summary message
	if len(modifiedFiles) > 0 {
		fmt.Printf("        ✓ Processed %d modified files (%d with skip-worktree)\n", len(modifiedFiles), len(keepFiles))
	}
}
