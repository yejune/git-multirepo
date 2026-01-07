package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yejune/git-sub/internal/update"
)

var selfupdateCmd = &cobra.Command{
	Use:   "selfupdate",
	Short: "Update git-subclone to the latest version",
	Long: `Check for and install the latest version of git-subclone.

This command checks GitHub releases for a newer version and automatically
downloads and installs it, replacing the current executable.

Examples:
  git-subclone selfupdate`,
	RunE: runSelfupdate,
}

func init() {
	rootCmd.AddCommand(selfupdateCmd)
}

// updaterFactory allows dependency injection for testing
var updaterFactory = func(version string) *update.Updater {
	return update.NewUpdater(version)
}

func runSelfupdate(cmd *cobra.Command, args []string) error {
	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	updater := updaterFactory(Version)

	release, hasUpdate, err := updater.CheckForUpdate()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !hasUpdate {
		fmt.Println("Already up to date.")
		return nil
	}

	fmt.Printf("New version available: %s\n", release.TagName)
	fmt.Println("Downloading...")

	if err := updater.Update(release); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Printf("Successfully updated to %s\n", release.TagName)
	return nil
}
