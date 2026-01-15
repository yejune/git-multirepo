// Package cmd implements the CLI commands for git-multirepo
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
	Use:   "git-multirepo [url] [path]",
	Short: "Manage nested git repositories with independent push capability",
	Long: `git-multirepo manages nested git repositories within a parent project.

Each workspace maintains its own .git directory and can push to its own remote,
while the parent project tracks the source files (but not .git).

Commands (workflow order):
  clone          Clone a new repository
  sync           Clone missing workspaces and apply configurations
  install-hook   Install git hook for automatic sync
  uninstall-hook Remove git hook
  status         Show detailed status of repositories
  pull           Pull latest changes
  branch         Show branch information
  list           List all registered workspaces
  remove         Remove a repository
  reset          Reset repository state
  selfupdate     Update git-multirepo to latest version`,
	Version: Version,
	Args:    cobra.MaximumNArgs(2),
	RunE:    runRoot,
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Flags().StringVarP(&rootBranch, "branch", "b", "", "Branch to clone")
	rootCmd.Flags().StringVarP(&rootPath, "path", "p", "", "Destination path")

	// Add commands in workflow order
	// This explicit ordering ensures help output shows commands in logical sequence
	rootCmd.AddCommand(cloneCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(installHookCmd)
	rootCmd.AddCommand(uninstallHookCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(branchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(selfupdateCmd)

	// Set custom usage template to show commands in workflow order
	rootCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)
}

func runRoot(cmd *cobra.Command, args []string) error {
	// No args = show help
	if len(args) == 0 {
		return cmd.Help()
	}

	// Show deprecation warning
	fmt.Println("⚠️  'git multirepo <url>' is deprecated")
	fmt.Println("Use 'git multirepo clone <url>' instead")
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
