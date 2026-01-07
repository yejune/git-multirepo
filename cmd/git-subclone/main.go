package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/git-subclone/internal/git"
	"github.com/yejune/git-subclone/internal/hooks"
	"github.com/yejune/git-subclone/internal/manifest"
)

var (
	version   = "0.1.0"
	recursive bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "git-subclone",
		Short:   "Manage nested git repositories",
		Long:    `git-subclone manages independent nested git repositories within a parent project.`,
		Version: version,
	}

	// add command
	addCmd := &cobra.Command{
		Use:   "add <repo> <path>",
		Short: "Add a new subclone",
		Args:  cobra.ExactArgs(2),
		RunE:  runAdd,
	}
	rootCmd.AddCommand(addCmd)

	// sync command
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync all subclones (clone or pull)",
		RunE:  runSync,
	}
	syncCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively sync nested subclones")
	rootCmd.AddCommand(syncCmd)

	// list command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all subclones",
		RunE:  runList,
	}
	rootCmd.AddCommand(listCmd)

	// push command
	pushCmd := &cobra.Command{
		Use:   "push [path]",
		Short: "Push subclone(s) to remote",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runPush,
	}
	rootCmd.AddCommand(pushCmd)

	// remove command
	removeCmd := &cobra.Command{
		Use:   "remove <path>",
		Short: "Remove a subclone",
		Args:  cobra.ExactArgs(1),
		RunE:  runRemove,
	}
	rootCmd.AddCommand(removeCmd)

	// init command (detect nested git and install hooks)
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize git-subclone (detect nested repos and install hooks)",
		RunE:  runInit,
	}
	rootCmd.AddCommand(initCmd)

	// verify command (check integrity)
	verifyCmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify subclone integrity and fix issues",
		RunE:  runVerify,
	}
	verifyCmd.Flags().BoolP("fix", "f", false, "Automatically fix issues")
	rootCmd.AddCommand(verifyCmd)

	// clean command (clean autoIgnore)
	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean autoIgnore patterns from .gitignore",
		RunE:  runClean,
	}
	rootCmd.AddCommand(cleanCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// applyLocalFilesFromManifest applies skip-worktree to all localFiles in manifest
func applyLocalFilesFromManifest(repoRoot string) error {
	m, err := manifest.Load(repoRoot)
	if err != nil {
		return err
	}

	// Apply mother repo localFiles
	if len(m.LocalFiles) > 0 {
		if err := git.ApplySkipWorktree(repoRoot, m.LocalFiles); err != nil {
			return err
		}
	}

	// Apply each subclone's localFiles
	for _, sc := range m.Subclones {
		if len(sc.LocalFiles) == 0 {
			continue
		}

		fullPath := filepath.Join(repoRoot, sc.Path)
		if !git.IsRepo(fullPath) {
			continue
		}

		if err := git.ApplySkipWorktree(fullPath, sc.LocalFiles); err != nil {
			return err
		}
	}

	return nil
}

// applyAutoIgnoreFromManifest applies autoIgnore patterns to .gitignore
func applyAutoIgnoreFromManifest(repoRoot string) error {
	m, err := manifest.Load(repoRoot)
	if err != nil {
		return err
	}

	if len(m.AutoIgnore) > 0 {
		return git.AddPatternsToGitignore(repoRoot, m.AutoIgnore)
	}

	return nil
}

func runAdd(cmd *cobra.Command, args []string) error {
	repo := args[0]
	path := args[1]

	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return err
	}

	// Apply localFiles and autoIgnore first
	applyLocalFilesFromManifest(repoRoot)
	applyAutoIgnoreFromManifest(repoRoot)

	fullPath := filepath.Join(repoRoot, path)

	// Check if path already exists
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("path %s already exists", path)
	}

	// Load manifest
	m, err := manifest.Load(repoRoot)
	if err != nil {
		return err
	}

	// Check if already registered
	if m.Exists(path) {
		return fmt.Errorf("subclone %s already registered", path)
	}

	// Clone repository
	fmt.Printf("Cloning %s into %s...\n", repo, path)
	if err := git.Clone(repo, fullPath, ""); err != nil {
		return fmt.Errorf("failed to clone: %w", err)
	}

	// Add to manifest
	m.Add(path, repo)
	if err := manifest.Save(repoRoot, m); err != nil {
		return err
	}

	// Add to .gitignore
	if err := git.AddToGitignore(repoRoot, path); err != nil {
		return err
	}

	fmt.Printf("Added subclone: %s\n", path)
	return nil
}

func runSync(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return err
	}

	if err := syncDir(repoRoot, recursive); err != nil {
		return err
	}

	// Apply localFiles and autoIgnore after sync
	if err := applyLocalFilesFromManifest(repoRoot); err != nil {
		return err
	}
	return applyAutoIgnoreFromManifest(repoRoot)
}

func syncDir(dir string, recursive bool) error {
	m, err := manifest.Load(dir)
	if err != nil {
		return err
	}

	if len(m.Subclones) == 0 {
		fmt.Println("No subclones configured")
		return nil
	}

	for _, sc := range m.Subclones {
		fullPath := filepath.Join(dir, sc.Path)

		if git.IsRepo(fullPath) {
			// Already has .git, just pull
			fmt.Printf("Pulling %s...\n", sc.Path)
			if err := git.Pull(fullPath); err != nil {
				fmt.Printf("Warning: failed to pull %s: %v\n", sc.Path, err)
			}
		} else if dirExists(fullPath) {
			// Directory exists but no .git (cloned from mother)
			// Initialize git and connect to remote
			fmt.Printf("Initializing %s...\n", sc.Path)
			if err := git.InitRepo(fullPath, sc.Repo, sc.Branch); err != nil {
				fmt.Printf("Warning: failed to init %s: %v\n", sc.Path, err)
				continue
			}
			fmt.Printf("Connected %s to %s\n", sc.Path, sc.Repo)
		} else {
			// Directory doesn't exist, clone fresh
			fmt.Printf("Cloning %s...\n", sc.Path)
			if err := git.Clone(sc.Repo, fullPath, sc.Branch); err != nil {
				fmt.Printf("Warning: failed to clone %s: %v\n", sc.Path, err)
				continue
			}
		}

		// Recursive sync
		if recursive {
			subManifest := filepath.Join(fullPath, manifest.FileName)
			if _, err := os.Stat(subManifest); err == nil {
				if err := syncDir(fullPath, recursive); err != nil {
					fmt.Printf("Warning: failed to sync nested subclones in %s: %v\n", sc.Path, err)
				}
			}
		}
	}

	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func runList(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return err
	}

	// Apply localFiles and autoIgnore first
	applyLocalFilesFromManifest(repoRoot)
	applyAutoIgnoreFromManifest(repoRoot)

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return err
	}

	if len(m.Subclones) == 0 {
		fmt.Println("No subclones configured")
		return nil
	}

	fmt.Println("Subclones:")
	for _, sc := range m.Subclones {
		fullPath := filepath.Join(repoRoot, sc.Path)
		status := "not cloned"
		if git.IsRepo(fullPath) {
			status = "cloned"
			if hasChanges, _ := git.HasChanges(fullPath); hasChanges {
				status = "modified"
			}
		}
		branchInfo := ""
		if sc.Branch != "" {
			branchInfo = fmt.Sprintf(" [%s]", sc.Branch)
		}
		fmt.Printf("  %s%s (%s) - %s\n", sc.Path, branchInfo, status, sc.Repo)
	}

	return nil
}

func runPush(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return err
	}

	// Apply localFiles and autoIgnore first
	applyLocalFilesFromManifest(repoRoot)
	applyAutoIgnoreFromManifest(repoRoot)

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		// Push specific subclone
		path := args[0]
		sc := m.Find(path)
		if sc == nil {
			return fmt.Errorf("subclone %s not found", path)
		}

		fullPath := filepath.Join(repoRoot, path)
		if !git.IsRepo(fullPath) {
			return fmt.Errorf("subclone %s is not cloned", path)
		}

		fmt.Printf("Pushing %s...\n", path)
		return git.Push(fullPath)
	}

	// Push all subclones with changes
	for _, sc := range m.Subclones {
		fullPath := filepath.Join(repoRoot, sc.Path)
		if !git.IsRepo(fullPath) {
			continue
		}

		hasChanges, err := git.HasChanges(fullPath)
		if err != nil {
			continue
		}

		if hasChanges {
			fmt.Printf("Skipping %s (has uncommitted changes)\n", sc.Path)
			continue
		}

		fmt.Printf("Pushing %s...\n", sc.Path)
		if err := git.Push(fullPath); err != nil {
			fmt.Printf("Warning: failed to push %s: %v\n", sc.Path, err)
		}
	}

	return nil
}

func runRemove(cmd *cobra.Command, args []string) error {
	path := args[0]

	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return err
	}

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return err
	}

	if !m.Remove(path) {
		return fmt.Errorf("subclone %s not found", path)
	}

	if err := manifest.Save(repoRoot, m); err != nil {
		return err
	}

	if err := git.RemoveFromGitignore(repoRoot, path); err != nil {
		return err
	}

	// Optionally remove the directory
	fullPath := filepath.Join(repoRoot, path)
	if _, err := os.Stat(fullPath); err == nil {
		fmt.Printf("Directory %s still exists. Remove it manually if needed.\n", path)
	}

	fmt.Printf("Removed subclone: %s\n", path)
	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return err
	}

	// Detect existing nested git repositories
	fmt.Println("Scanning for nested git repositories...")
	nested, err := findNestedGitRepos(repoRoot)
	if err != nil {
		return err
	}

	if len(nested) == 0 {
		fmt.Println("No nested git repositories found")
	} else {
		fmt.Printf("Found %d nested git repositories:\n", len(nested))

		m, err := manifest.Load(repoRoot)
		if err != nil {
			return err
		}

		// Filter out already registered
		var toRegister []nestedGitInfo
		for _, info := range nested {
			if m.Exists(info.Path) {
				fmt.Printf("  %s - already registered\n", info.Path)
			} else {
				fmt.Printf("  %s (%s)\n", info.Path, info.Repo)
				toRegister = append(toRegister, info)
			}
		}

		if len(toRegister) == 0 {
			fmt.Println("All nested repositories are already registered")
		} else {
			// Ask user for confirmation
			fmt.Printf("\nRegister %d new subclone(s)? (Y/n): ", len(toRegister))
			var response string
			fmt.Scanln(&response)

			if response == "" || strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
				added := 0
				for _, info := range toRegister {
					// Add to manifest
					m.Add(info.Path, info.Repo, info.Branch)

					// Add to .gitignore
					if err := git.AddToGitignore(repoRoot, info.Path); err != nil {
						fmt.Printf("  %s - warning: failed to update .gitignore: %v\n", info.Path, err)
					}

					fmt.Printf("  %s - registered\n", info.Path)
					added++
				}

				if added > 0 {
					if err := manifest.Save(repoRoot, m); err != nil {
						return err
					}
					fmt.Printf("Registered %d new subclones\n", added)
				}
			} else {
				fmt.Println("Cancelled")
			}
		}
	}

	// Install hooks
	if err := hooks.Install(repoRoot); err != nil {
		return err
	}

	// Apply autoIgnore patterns
	applyAutoIgnoreFromManifest(repoRoot)

	fmt.Println("Installed git-subclone hooks")
	return nil
}

type nestedGitInfo struct {
	Path   string
	Repo   string
	Branch string
}

func findNestedGitRepos(root string) ([]nestedGitInfo, error) {
	var result []nestedGitInfo

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip root .git
		if path == filepath.Join(root, ".git") {
			return filepath.SkipDir
		}

		// Skip common directories
		name := filepath.Base(path)
		if name == "node_modules" || name == "vendor" || name == ".terraform" {
			return filepath.SkipDir
		}

		// Check if this is a git repo
		if info.IsDir() && name == ".git" {
			parentDir := filepath.Dir(path)
			relPath, err := filepath.Rel(root, parentDir)
			if err != nil {
				return err
			}

			// Get remote URL
			repo, err := git.GetRemoteURL(parentDir)
			if err != nil {
				repo = "" // No remote configured
			}

			// Get current branch
			branch, err := git.GetCurrentBranch(parentDir)
			if err != nil {
				branch = ""
			}

			result = append(result, nestedGitInfo{
				Path:   relPath,
				Repo:   repo,
				Branch: branch,
			})

			return filepath.SkipDir // Don't recurse into this git repo
		}

		return nil
	})

	return result, err
}

func runVerify(cmd *cobra.Command, args []string) error {
	fix, _ := cmd.Flags().GetBool("fix")

	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return err
	}

	// Apply localFiles and autoIgnore first
	applyLocalFilesFromManifest(repoRoot)
	applyAutoIgnoreFromManifest(repoRoot)

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return err
	}

	fmt.Println("Verifying subclone integrity...")
	issues := 0

	// Check each subclone
	for _, sc := range m.Subclones {
		fullPath := filepath.Join(repoRoot, sc.Path)

		// Check 1: .gitignore entry
		gitignorePath := filepath.Join(repoRoot, ".gitignore")
		content, err := os.ReadFile(gitignorePath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		expected := sc.Path + "/.git/"
		found := false
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) == expected {
					found = true
					break
				}
			}
		}

		if !found {
			issues++
			fmt.Printf("✗ %s: missing from .gitignore\n", sc.Path)
			if fix {
				if err := git.AddToGitignore(repoRoot, sc.Path); err != nil {
					fmt.Printf("  Failed to fix: %v\n", err)
				} else {
					fmt.Printf("  Fixed: added to .gitignore\n")
				}
			}
		}

		// Check 2: Directory exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			issues++
			fmt.Printf("✗ %s: directory does not exist\n", sc.Path)
		}

		// Check 3: Has .git or is empty
		if git.IsRepo(fullPath) {
			// Has .git - check remote matches
			remote, err := git.GetRemoteURL(fullPath)
			if err == nil && remote != sc.Repo {
				issues++
				fmt.Printf("✗ %s: remote mismatch (expected %s, got %s)\n", sc.Path, sc.Repo, remote)
			}
		} else if dirExists(fullPath) {
			// Directory exists but no .git
			fmt.Printf("! %s: directory exists but not initialized (run 'git subclone sync')\n", sc.Path)
		}
	}

	if issues == 0 {
		fmt.Println("✓ All subclones verified successfully")
	} else {
		fmt.Printf("\nFound %d issue(s)\n", issues)
		if !fix {
			fmt.Println("Run with --fix to automatically fix issues")
		}
	}

	return nil
}
