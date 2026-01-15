package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-multirepo/internal/backup"
	"github.com/yejune/git-multirepo/internal/git"
	"github.com/yejune/git-multirepo/internal/manifest"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset repository state (unhide all hidden files)",
	Long: `Reset repository state by unhiding all files.

This will:
  - Unskip all keep files (root and repositories)
  - Remove all ignore patterns from .gitignore
  - Clear keep/ignore from .git.multirepos
  - Create backups before changes

All hidden files will become visible again.`,
	RunE: runReset,
}

func init() {
	// Command registered in root.go init() in workflow order
}

func runReset(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	backupDir := filepath.Join(repoRoot, ".multirepos", "backup")

	fmt.Println("Resetting repository state (unhiding all)...")

	// ============ 1. Keep 파일 처리 ============
	// Mother repo
	if len(m.Keep) > 0 {
		fmt.Println("\nMother repo:")

		// 백업
		for _, file := range m.Keep {
			if err := backup.CreateFileBackup(filepath.Join(repoRoot, file), backupDir, repoRoot); err != nil {
				return fmt.Errorf("failed to backup %s: %w", file, err)
			}
		}

		// Unskip
		if err := git.UnapplySkipWorktree(repoRoot, m.Keep); err != nil {
			return fmt.Errorf("failed to unapply skip-worktree: %w", err)
		}

		fmt.Printf("  ✓ Unskipped %d keep files\n", len(m.Keep))

		// Keep 리스트 제거
		m.Keep = []string{}
	}

	// Workspaces
	for i := range m.Workspaces {
		ws := &m.Workspaces[i]
		if len(ws.Keep) > 0 {
			fullPath := filepath.Join(repoRoot, ws.Path)
			fmt.Printf("\n%s:\n", ws.Path)

			// 백업
			for _, file := range ws.Keep {
				if err := backup.CreateFileBackup(filepath.Join(fullPath, file), backupDir, repoRoot); err != nil {
					return fmt.Errorf("failed to backup %s: %w", file, err)
				}
			}

			// Unskip
			if err := git.UnapplySkipWorktree(fullPath, ws.Keep); err != nil {
				return fmt.Errorf("failed to unapply skip-worktree in %s: %w", ws.Path, err)
			}

			fmt.Printf("  ✓ Unskipped %d keep files\n", len(ws.Keep))

			// Keep 리스트 제거
			ws.Keep = []string{}
		}
	}

	// ============ 2. Ignore 패턴 처리 ============
	if len(m.Ignore) > 0 {
		fmt.Println("\nRemoving ignore patterns...")

		// .gitignore에서 패턴 제거
		git.RemoveIgnorePatternsFromGitignore(repoRoot)

		fmt.Printf("  ✓ Removed %d ignore patterns\n", len(m.Ignore))

		// Ignore 리스트 제거
		m.Ignore = []string{}
	}

	// Manifest 저장
	manifest.Save(repoRoot, m)

	fmt.Println("\n✓ All hidden files are now visible")
	fmt.Println("ℹ Backups saved to .multirepos/backup/")
	fmt.Println("ℹ Patches preserved in .multirepos/patches/")

	return nil
}
