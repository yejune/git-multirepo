package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/git-multirepo/internal/update"
)

var selfupdateCmd = &cobra.Command{
	Use:     "selfupdate",
	Aliases: []string{"self-update"},
	Short:   "Update git-multirepo to the latest version",
	Long: `Check for and install the latest version of git-multirepo.

This command checks GitHub releases for a newer version and automatically
downloads and installs it, replacing the current executable.

Examples:
  git multirepo selfupdate`,
	RunE: runSelfupdate,
}

func init() {
	// Command registered in root.go init() in workflow order
}

// updaterFactory allows dependency injection for testing
var updaterFactory = func(version string) *update.Updater {
	return update.NewUpdater(version)
}

func runSelfupdate(cmd *cobra.Command, args []string) error {
	// Check if installed via Homebrew
	execPath, err := os.Executable()
	if err == nil {
		execPath, _ = filepath.EvalSymlinks(execPath)
		if strings.Contains(execPath, "/homebrew/") || strings.Contains(execPath, "/Cellar/") || strings.Contains(execPath, "/Homebrew/") {
			fmt.Println("Detected Homebrew installation")
			fmt.Println("Running: brew upgrade yejune/tap/git-multirepo")

			brewCmd := exec.Command("brew", "upgrade", "yejune/tap/git-multirepo")
			brewCmd.Stdout = os.Stdout
			brewCmd.Stderr = os.Stderr
			return brewCmd.Run()
		}
	}

	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	updater := updaterFactory(Version)

	release, hasUpdate, err := updater.CheckForUpdate()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !hasUpdate {
		fmt.Printf("\nLatest version:  %s\n", Version)
		fmt.Println("Already up to date.")
		return nil
	}

	fmt.Printf("\nLatest version:  %s\n", release.TagName)
	fmt.Println()
	fmt.Println("Downloading and installing...")

	if err := updater.Update(release); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Printf("✓ Successfully updated to %s\n", release.TagName)
	return nil
}
