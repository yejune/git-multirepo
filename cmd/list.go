package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/manifest"
)

var listRecursive bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all registered workspaces",
	Long: `Display all workspaces registered in .workspaces.

Shows path, repository URL, branch, and current status.

Examples:
  git workspace list
  git workspace ls
  git workspace ls -r`,
	RunE: runList,
}

func init() {
	listCmd.Flags().BoolVarP(&listRecursive, "recursive", "r", false, "Recursively list workspaces within workspaces")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	return listDir(repoRoot, listRecursive, 0)
}

func listDir(dir string, recursive bool, depth int) error {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	m, err := manifest.Load(dir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if len(m.Workspaces) == 0 {
		if depth == 0 {
			fmt.Println("No workspaces registered.")
		}
		return nil
	}

	for _, ws := range m.Workspaces {
		fullPath := filepath.Join(dir, ws.Path)

		// Check status
		var status string
		if !git.IsRepo(fullPath) {
			status = "not cloned"
		} else {
			hasChanges, err := git.HasChanges(fullPath)
			if err != nil {
				status = "error"
			} else if hasChanges {
				status = "modified"
			} else {
				status = "clean"
			}
		}

		// Format output
		statusIcon := map[string]string{
			"clean":      "✓",
			"modified":   "●",
			"not cloned": "○",
			"error":      "✗",
		}[status]

		fmt.Printf("%s%s %s\n", indent, statusIcon, ws.Path)
		fmt.Printf("%s  └─ %s\n", indent, ws.Repo)

		// Recursive list
		if recursive {
			subManifest := filepath.Join(fullPath, manifest.FileName)
			if _, err := os.Stat(subManifest); err == nil {
				if err := listDir(fullPath, recursive, depth+1); err != nil {
					fmt.Printf("%s  ⚠ Warning: %v\n", indent, err)
				}
			}
		}
	}

	return nil
}
