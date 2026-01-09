// Package cmd implements the CLI commands for git-workspace
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via -ldflags
	Version = "0.2.11"
	// Root command flags
	rootBranch string
	rootPath   string
)

// Deprecated: Use 'clone' command instead
var rootCmd = &cobra.Command{
	Use:   "git-workspace [url] [path]",
	Short: "Manage nested git repositories with independent push capability",
	Long: `git-workspace manages nested git repositories within a parent project.

Each workspace maintains its own .git directory and can push to its own remote,
while the parent project tracks the source files (but not .git).

Commands:
  clone    Clone a new workspace repository
  sync     Clone or pull all workspaces
  list     List all registered workspaces
  remove   Remove a workspace
  status   Show workspace status
  pull     Pull workspace changes
  reset    Reset workspace state
  branch   Manage workspace branches
  selfupdate Update git-workspace to latest version`,
	Version: Version,
	Args:    cobra.MaximumNArgs(2),
	RunE:    runRoot,
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Flags().StringVarP(&rootBranch, "branch", "b", "", "Branch to clone")
	rootCmd.Flags().StringVarP(&rootPath, "path", "p", "", "Destination path")
}

func runRoot(cmd *cobra.Command, args []string) error {
	// No args = show help
	if len(args) == 0 {
		return cmd.Help()
	}

	// Show deprecation warning
	fmt.Println("⚠️  'git workspace <url>' is deprecated")
	fmt.Println("Use 'git workspace clone <url>' instead")
	fmt.Println()

	// Delegate to cloneCmd
	// Transfer flags from root to clone
	cloneBranch = rootBranch
	clonePath = rootPath

	return cloneCmd.RunE(cmd, args)
}

// osExit is a variable that can be overridden in tests
var osExit = os.Exit

// Execute runs the root command and exits with code 1 on error
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		osExit(1)
	}
}
