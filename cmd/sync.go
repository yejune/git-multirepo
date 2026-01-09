package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/backup"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/hooks"
	"github.com/yejune/git-workspace/internal/i18n"
	"github.com/yejune/git-workspace/internal/manifest"
	"github.com/yejune/git-workspace/internal/patch"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Clone missing workspaces and apply configurations",
	Long: `Sync all workspaces from .workspaces manifest:
  - Clone missing workspaces automatically
  - Install git hooks if not present
  - Apply ignore patterns to .gitignore
  - Apply skip-worktree to specified files
  - Verify .gitignore entries for workspaces

Examples:
  git workspace sync`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load manifest early to get language setting
	m, err := manifest.Load(repoRoot)
	if err == nil {
		i18n.SetLanguage(m.GetLanguage())
	}

	fmt.Println(i18n.T("syncing"))

	// 1. Auto-install hooks
	if !hooks.IsInstalled(repoRoot) {
		fmt.Println(i18n.T("installing_hooks"))
		if err := hooks.Install(repoRoot); err != nil {
			fmt.Printf("  %s\n", i18n.T("hooks_failed", err))
		} else {
			fmt.Printf("  %s\n", i18n.T("hooks_installed"))
		}
	}

	// 2. Load manifest
	if err != nil || len(m.Workspaces) == 0 {
		// No manifest or empty - scan directories for existing workspaces
		fmt.Println(i18n.T("no_gitsubs_found"))
		discovered, scanErr := scanForWorkspaces(repoRoot)
		if scanErr != nil {
			return fmt.Errorf(i18n.T("failed_scan"), scanErr)
		}

		if len(discovered) == 0 {
			fmt.Println(i18n.T("no_subs_found"))
			fmt.Println(i18n.T("to_add_sub"))
			fmt.Println(i18n.T("cmd_git_sub_clone"))
			return nil
		}

		// Create manifest from discovered workspaces
		m = &manifest.Manifest{
			Workspaces: discovered,
		}

		if err := manifest.Save(repoRoot, m); err != nil {
			return fmt.Errorf("failed to save manifest: %w", err)
		}

		fmt.Printf(i18n.T("created_gitsubs", len(discovered)))
		for _, ws := range discovered {
			fmt.Printf("  - %s (%s)\n", ws.Path, ws.Repo)
		}
	}

	// 3. Apply ignore patterns to mother repo
	if len(m.Ignore) > 0 {
		fmt.Println(i18n.T("applying_ignore"))
		if err := git.AddIgnorePatternsToGitignore(repoRoot, m.Ignore); err != nil {
			fmt.Printf("  %s\n", i18n.T("hooks_failed", err))
		} else {
			fmt.Printf("  %s\n", i18n.T("applied_patterns", len(m.Ignore)))
		}
	}

	// 4. Process Mother repo keep files
	issues := 0
	motherKeepFiles := m.Keep
	if len(motherKeepFiles) > 0 {
		fmt.Printf("\n%s\n", i18n.T("processing_mother_keep"))
		processKeepFiles(repoRoot, repoRoot, motherKeepFiles, &issues)
	}

	if len(m.Workspaces) == 0 {
		fmt.Println(i18n.T("no_subclones"))
		return nil
	}

	// 5. Process each workspace
	fmt.Println(i18n.T("processing_subclones"))

	for _, ws := range m.Workspaces {
		fullPath := filepath.Join(repoRoot, ws.Path)
		fmt.Printf("\n  %s\n", ws.Path)

		// Check if workspace exists
		if !git.IsRepo(fullPath) {
			// Check if directory has files (parent is tracking source)
			entries, err := os.ReadDir(fullPath)
			if err == nil && len(entries) > 0 {
				// Directory exists with files - init git in place
				fmt.Printf("    %s\n", i18n.T("initializing_git"))

				if err := git.InitRepo(fullPath, ws.Repo, ws.Branch, ws.Commit); err != nil {
					fmt.Printf("    %s\n", i18n.T("failed_initialize", err))
					issues++
					continue
				}

				// Add to .gitignore
				if err := git.AddToGitignore(repoRoot, ws.Path); err != nil {
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
			if err := git.AddToGitignore(repoRoot, ws.Path); err != nil {
				fmt.Printf("    %s\n", i18n.T("failed_update_gitignore", err))
			}

			fmt.Printf("    %s\n", i18n.T("cloned_successfully"))
			continue
		}

		// Auto-update commit hash in .workspaces
		commit, err := git.GetCurrentCommit(fullPath)
		if err == nil && commit != ws.Commit {
			// Check if pushed
			hasUnpushed, checkErr := git.HasUnpushedCommits(fullPath)
			if checkErr == nil {
				if hasUnpushed {
					fmt.Printf("    %s\n", i18n.T("has_unpushed", commit[:7]))
					fmt.Printf("      %s\n", i18n.T("push_first", ws.Path))
				} else {
					// Update .workspaces with pushed commit
					oldCommit := "none"
					if ws.Commit != "" {
						oldCommit = ws.Commit[:7]
					}
					m.UpdateCommit(ws.Path, commit)
					fmt.Printf("    %s\n", i18n.T("updated_commit", oldCommit, commit[:7]))
				}
			}
		}

		// Verify and fix .gitignore entry
		if !hasGitignoreEntry(repoRoot, ws.Path) {
			fmt.Printf("    %s\n", i18n.T("adding_to_gitignore"))
			if err := git.AddToGitignore(repoRoot, ws.Path); err != nil {
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
			processKeepFiles(repoRoot, fullPath, keepFiles, &issues)
		}

		// Install/update post-commit hook in workspace
		if !hooks.IsSubHookInstalled(fullPath) {
			fmt.Printf("    %s\n", i18n.T("installing_hook"))
			if err := hooks.InstallSubHook(fullPath); err != nil {
				fmt.Printf("    %s\n", i18n.T("hook_failed", err))
			} else {
				fmt.Printf("    %s\n", i18n.T("hook_installed"))
			}
		}
	}

	// Save manifest if any commits were updated
	if err := manifest.Save(repoRoot, m); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
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

// scanForWorkspaces recursively scans directories for git repositories
func scanForWorkspaces(repoRoot string) ([]manifest.WorkspaceEntry, error) {
	var workspaces []manifest.WorkspaceEntry

	// Walk the directory tree
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip parent's .git directory
		if path == filepath.Join(repoRoot, ".git") {
			return filepath.SkipDir
		}

		// Check if this is a .git directory
		if !info.IsDir() || info.Name() != ".git" {
			return nil
		}

		// Get the repository path (parent of .git)
		workspacePath := filepath.Dir(path)

		// Skip if it's the parent repo itself
		if workspacePath == repoRoot {
			return filepath.SkipDir
		}

		// Get relative path from parent
		relPath, err := filepath.Rel(repoRoot, workspacePath)
		if err != nil {
			return nil
		}

		// Extract git info
		repo, err := git.GetRemoteURL(workspacePath)
		if err != nil {
			fmt.Println(i18n.T("failed_get_remote", relPath, err))
			return filepath.SkipDir
		}

		commit, err := git.GetCurrentCommit(workspacePath)
		if err != nil {
			fmt.Println(i18n.T("failed_get_commit", relPath, err))
			return filepath.SkipDir
		}

		// Detect modified files for auto-keep
		var keepFiles []string
		// Get skip-worktree files (these are the keep files)
		skipFiles, err := git.ListSkipWorktree(workspacePath)
		if err == nil && len(skipFiles) > 0 {
			keepFiles = skipFiles
		} else {
			// Fallback: detect modified files for first-time setup
			modifiedFiles, err := git.GetModifiedFiles(workspacePath)
			if err == nil && len(modifiedFiles) > 0 {
				// Clean up file list
				for _, file := range modifiedFiles {
					if strings.TrimSpace(file) != "" {
						keepFiles = append(keepFiles, file)
					}
				}
			}
		}

		fmt.Printf("  %s\n", i18n.T("found_sub", relPath))

		workspaces = append(workspaces, manifest.WorkspaceEntry{
			Path:   relPath,
			Repo:   repo,
			Commit: commit,
			Keep:   keepFiles,
		})

		// Skip descending into this workspace's subdirectories
		return filepath.SkipDir
	})

	return workspaces, err
}

// processKeepFiles handles backup, patch creation, and skip-worktree for keep files
func processKeepFiles(repoRoot, workspacePath string, keepFiles []string, issues *int) {
	backupDir := filepath.Join(repoRoot, ".workspaces", "backup")
	patchBaseDir := filepath.Join(repoRoot, ".workspaces", "patches")

	// Determine workspace path for patches and backups
	relPath, err := filepath.Rel(repoRoot, workspacePath)
	if err != nil {
		relPath = filepath.Base(workspacePath)
	}
	if relPath == "." {
		relPath = ""
	}

	// Clean slate strategy: Remove directories before saving to prevent file leakage

	// 1. Clean patches directory (complete workspace patch dir)
	patchDir := filepath.Join(patchBaseDir, relPath)
	os.RemoveAll(patchDir)
	os.MkdirAll(patchDir, 0755)

	// 2. Clean today's backup directories
	today := time.Now().Format("2006/01/02")

	// Clean today's modified backup
	modifiedDir := filepath.Join(backupDir, "modified", today, relPath)
	os.RemoveAll(modifiedDir)
	os.MkdirAll(modifiedDir, 0755)

	// Clean today's patched backup
	patchedDir := filepath.Join(backupDir, "patched", today, relPath)
	os.RemoveAll(patchedDir)
	os.MkdirAll(patchedDir, 0755)

	// 3. Get ALL modified files (not just Keep files)
	modifiedFiles, err := git.GetModifiedFiles(workspacePath)
	if err != nil {
		fmt.Printf("        Failed to get modified files: %v\n", err)
		*issues++
		return
	}

	// Remove empty strings from the list
	var cleanModifiedFiles []string
	for _, file := range modifiedFiles {
		if strings.TrimSpace(file) != "" {
			cleanModifiedFiles = append(cleanModifiedFiles, file)
		}
	}
	modifiedFiles = cleanModifiedFiles

	// 4. Auto-populate Keep list if empty and there are modified files
	if len(keepFiles) == 0 && len(modifiedFiles) > 0 {
		// Load manifest to update it
		m, loadErr := manifest.Load(repoRoot)
		if loadErr != nil {
			fmt.Printf("        Failed to load manifest: %v\n", loadErr)
			*issues++
			return
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
			fmt.Printf("        Failed to save manifest: %v\n", saveErr)
			*issues++
			return
		}

		// Update keepFiles for this run
		keepFiles = modifiedFiles

		fmt.Printf("\n✓ Found %d modified files and added to keep list:\n", len(modifiedFiles))
		for _, f := range modifiedFiles {
			fmt.Printf("  - %s\n", f)
		}
		fmt.Println("\nEdit .git.workspaces to keep only the files you need")
	}

	// 5. Process ALL modified files (backup + patch for all)
	for _, file := range modifiedFiles {
		filePath := filepath.Join(workspacePath, file)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue // Skip if file doesn't exist
		}

		// 5a. Backup original file to backup/modified/
		if err := backup.CreateFileBackup(filePath, backupDir, repoRoot); err != nil {
			fmt.Printf("        Failed to backup %s: %v\n", file, err)
			*issues++
			continue
		}

		// 5b. Create patch (git diff HEAD file)
		patchPath := filepath.Join(patchBaseDir, relPath, file+".patch")
		if err := patch.Create(workspacePath, file, patchPath); err != nil {
			fmt.Printf("        Failed to create patch for %s: %v\n", file, err)
			*issues++
			continue
		}

		// 5c. Backup patch to backup/patched/
		if err := backup.CreatePatchBackup(patchPath, backupDir); err != nil {
			fmt.Printf("        Failed to backup patch for %s: %v\n", file, err)
			*issues++
			continue
		}
	}

	// 6. Apply skip-worktree ONLY to Keep files
	if len(keepFiles) > 0 {
		fmt.Printf("        Applying skip-worktree to %d keep files...\n", len(keepFiles))
		for _, file := range keepFiles {
			filePath := filepath.Join(workspacePath, file)

			// Check if file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				fmt.Printf("        %s (skip: file not found)\n", file)
				continue
			}

			// Apply skip-worktree
			if err := git.ApplySkipWorktree(workspacePath, []string{file}); err != nil {
				fmt.Printf("        Failed to apply skip-worktree to %s: %v\n", file, err)
				*issues++
				continue
			}

			fmt.Printf("        ✓ %s (skip-worktree applied)\n", file)
		}
	}

	// Summary message
	if len(modifiedFiles) > 0 {
		fmt.Printf("        ✓ Processed %d modified files (%d with skip-worktree)\n", len(modifiedFiles), len(keepFiles))
	}
}
