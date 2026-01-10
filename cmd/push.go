// Package cmd implements the CLI commands for git-workspace
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/config"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/github"
)

var pushCmd = &cobra.Command{
	Use:    "push [path]",
	Short:  "Push workspace to organization",
	Long: `Push local commits to organization repository.

Creates a private repository if it doesn't exist.

Examples:
  git workspace push              # Push current directory
  git workspace push apps/admin   # Push specific workspace

Prerequisites:
  - ~/.git.workspaces must exist with organization configured
  - GitHub authentication (gh CLI or git credential helper)`,
	Hidden: true, // Hidden command
	RunE:   runPush,
}

func init() {
	// CRITICAL: Only register command if config exists
	if shouldEnablePushCommand() {
		rootCmd.AddCommand(pushCmd)
	}
}

// shouldEnablePushCommand checks if push command should be enabled
func shouldEnablePushCommand() bool {
	if !config.ConfigExists() {
		return false
	}

	org, err := config.GetOrganization()
	return err == nil && org != ""
}

// runPush implements the push workflow
func runPush(cmd *cobra.Command, args []string) error {
	// 1. Determine workspace path (from args or current dir)
	workspacePath, err := determineWorkspacePath(args)
	if err != nil {
		return err
	}

	// Verify it's a git repository
	if !git.IsRepo(workspacePath) {
		return fmt.Errorf("not a git repository: %s", workspacePath)
	}

	// 2. Get organization URL from config
	orgURL, err := config.GetOrganization()
	if err != nil {
		return fmt.Errorf("organization not configured in ~/.git.workspaces: %w", err)
	}

	// 3. Determine repository name (normalize using config)
	repoName := filepath.Base(workspacePath)
	repoName, err = config.NormalizeRepoName(repoName)
	if err != nil {
		return fmt.Errorf("failed to normalize repo name: %w", err)
	}

	// 4. Prompt for repository name confirmation
	repoName, err = promptRepositoryName(repoName, orgURL)
	if err != nil {
		return err
	}

	if strings.TrimSpace(repoName) == "" {
		return fmt.Errorf("repository name cannot be empty")
	}

	// 5. Get authentication token
	token, err := github.GetAuthToken()
	if err != nil {
		return err
	}

	// 6. Create GitHub client
	client, err := github.NewClient(token, orgURL)
	if err != nil {
		return err
	}

	// 7. Check if repository exists
	exists, err := client.RepositoryExists(repoName)
	if err != nil {
		return fmt.Errorf("failed to check repository: %w", err)
	}

	// 8. Create repository if needed (interactive prompt)
	if !exists {
		if err := createRepositoryInteractive(client, repoName); err != nil {
			return err
		}
	}

	// 9. Setup git remote
	repoURL := client.GetRepoURL(repoName)
	if err := setupRemote(workspacePath, repoURL); err != nil {
		return err
	}

	// 10. Push to remote
	orgName := filepath.Base(orgURL)
	fmt.Printf("\nPushing to %s/%s...\n", orgName, repoName)

	if err := git.Push(workspacePath); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	fmt.Printf("✓ Successfully pushed to %s\n", repoURL)
	return nil
}

// determineWorkspacePath gets workspace path from args or current dir
func determineWorkspacePath(args []string) (string, error) {
	if len(args) > 0 {
		path := args[0]
		if !filepath.IsAbs(path) {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to get current directory: %w", err)
			}
			path = filepath.Join(cwd, path)
		}
		return path, nil
	}

	// Use current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return cwd, nil
}

// promptRepositoryName asks user to confirm/edit repository name
func promptRepositoryName(defaultName, orgURL string) (string, error) {
	orgName := filepath.Base(orgURL)
	fmt.Printf("\nRepository: %s/%s\n", orgName, defaultName)

	var repoName string
	prompt := &survey.Input{
		Message: "Repository name:",
		Default: defaultName,
	}

	if err := survey.AskOne(prompt, &repoName); err != nil {
		return "", err
	}

	return strings.TrimSpace(repoName), nil
}

// createRepositoryInteractive prompts and creates repository
func createRepositoryInteractive(client *github.Client, repoName string) error {
	fmt.Println("\nRepository not found.")

	var createRepo bool
	prompt := &survey.Confirm{
		Message: "Create private repository?",
		Default: false,
	}

	if err := survey.AskOne(prompt, &createRepo); err != nil {
		return err
	}

	if !createRepo {
		return fmt.Errorf("repository creation cancelled")
	}

	fmt.Println("\nCreating repository...")
	if err := client.CreateRepository(repoName); err != nil {
		return err
	}

	fmt.Printf("✓ Created private repository: %s\n", repoName)
	return nil
}

// setupRemote adds or verifies git remote
func setupRemote(workspacePath, repoURL string) error {
	existingURL, err := git.GetRemoteURL(workspacePath)

	if err != nil {
		// No remote - add it
		cmd := exec.Command("git", "-C", workspacePath,
			"remote", "add", "origin", repoURL)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
		return nil
	}

	// Remote exists - check URL
	if existingURL != repoURL {
		return fmt.Errorf(
			"remote 'origin' already exists with different URL:\n"+
				"  Current:  %s\n"+
				"  Expected: %s\n\n"+
				"Update manually: git remote set-url origin %s",
			existingURL, repoURL, repoURL)
	}

	return nil // URL matches
}
