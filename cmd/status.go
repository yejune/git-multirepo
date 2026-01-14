package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yejune/git-multirepo/internal/common"
	"github.com/yejune/git-multirepo/internal/git"
	"github.com/yejune/git-multirepo/internal/i18n"
)

var (
	statusFetch bool
)

var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Show detailed status of repositories",
	Long: `Display comprehensive status information for each repository:

Examples:
  git multirepo status              # Show status for all repositories
  git multirepo status --fetch      # Fetch from remote before showing status
  git multirepo status apps/admin   # Show status for specific repository

For each repository, shows:
  1. Local Status (modified, untracked, staged files)
  2. Remote Status (commits behind/ahead based on last fetch)
  3. How to resolve (step-by-step commands)`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusFetch, "fetch", false, "Fetch from remote before showing status")
	rootCmd.AddCommand(statusCmd)
}

// IntegrityIssue represents an integrity validation issue
type IntegrityIssue struct {
	Level   string // "critical", "warning", "info"
	Message string
	Path    string
	Fix     string
}

// validateMultirepoIntegrity performs comprehensive integrity checks
func validateMultirepoIntegrity(ctx *common.WorkspaceContext) []IntegrityIssue {
	var issues []IntegrityIssue

	// 1. Check for nested manifests (CRITICAL)
	nestedManifests := findNestedManifests(ctx)
	for _, path := range nestedManifests {
		issues = append(issues, IntegrityIssue{
			Level:   "critical",
			Message: i18n.T("nested_manifest_critical"),
			Path:    path,
			Fix:     fmt.Sprintf("rm %s", filepath.Join(path, ".git.multirepos")),
		})
	}

	// 2. Check for parent manifest (WARNING)
	parentPath := findParentManifest(ctx)
	if parentPath != "" {
		issues = append(issues, IntegrityIssue{
			Level:   "warning",
			Message: i18n.T("parent_manifest_warning"),
			Path:    parentPath,
			Fix:     "",
		})
	}

	// 3. Check for unregistered workspaces (WARNING)
	unregistered := findUnregisteredWorkspaces(ctx)
	if len(unregistered) > 0 {
		issues = append(issues, IntegrityIssue{
			Level:   "warning",
			Message: fmt.Sprintf(i18n.T("unregistered_workspace_warning"), len(unregistered)),
			Path:    strings.Join(unregistered, "\n"),
			Fix:     "git multirepo sync",
		})
	}

	// 4. Check for remote URL mismatches (WARNING)
	mismatches := findRemoteURLMismatches(ctx)
	issues = append(issues, mismatches...)

	// 5. Check for local path repos (WARNING)
	localPathRepos := findLocalPathRepos(ctx)
	issues = append(issues, localPathRepos...)

	// 6. Check for package manager dependencies registered as workspaces (WARNING)
	pkgManagerDeps := findPackageManagerDependencies(ctx)
	issues = append(issues, pkgManagerDeps...)

	return issues
}

// findLocalPathRepos checks for workspaces with local filesystem paths as repo URLs
func findLocalPathRepos(ctx *common.WorkspaceContext) []IntegrityIssue {
	var issues []IntegrityIssue

	for _, ws := range ctx.Manifest.Workspaces {
		if strings.HasPrefix(ws.Repo, "/") {
			issues = append(issues, IntegrityIssue{
				Level:   "warning",
				Message: "Local path repo URL detected",
				Path:    ws.Path,
				Fix:     fmt.Sprintf("Repo URL is '%s' - this won't work on other machines.\n    Update .git.multirepos with a valid remote URL or remove with:\n    git multirepo remove %s", ws.Repo, ws.Path),
			})
		}
	}

	return issues
}

// findPackageManagerDependencies checks for package manager dependencies registered as workspaces
func findPackageManagerDependencies(ctx *common.WorkspaceContext) []IntegrityIssue {
	var issues []IntegrityIssue

	for _, ws := range ctx.Manifest.Workspaces {
		absPath := filepath.Join(ctx.RepoRoot, ws.Path)
		if shouldExcludeWorkspace(absPath, ws.Path, ctx.RepoRoot) {
			issues = append(issues, IntegrityIssue{
				Level:   "warning",
				Message: "Package manager dependency registered as workspace",
				Path:    ws.Path,
				Fix:     fmt.Sprintf("This appears to be a package manager dependency (e.g., Swift PM, npm).\n    Remove with: git multirepo remove %s", ws.Path),
			})
		}
	}

	return issues
}

// findNestedManifests searches for .git.multirepos files within workspace directories
func findNestedManifests(ctx *common.WorkspaceContext) []string {
	var nested []string

	for _, ws := range ctx.Manifest.Workspaces {
		wsPath := filepath.Join(ctx.RepoRoot, ws.Path)
		manifestPath := filepath.Join(wsPath, ".git.multirepos")

		if _, err := os.Stat(manifestPath); err == nil {
			nested = append(nested, ws.Path)
		}
	}

	return nested
}

// findParentManifest checks if there's a parent .git.multirepos above the current repo root
func findParentManifest(ctx *common.WorkspaceContext) string {
	parent := filepath.Dir(ctx.RepoRoot)

	// Don't search beyond filesystem root
	if parent == ctx.RepoRoot || parent == "/" {
		return ""
	}

	manifestPath := filepath.Join(parent, ".git.multirepos")
	if _, err := os.Stat(manifestPath); err == nil {
		return parent
	}

	return ""
}

// findUnregisteredWorkspaces looks for .git directories not in the manifest
func findUnregisteredWorkspaces(ctx *common.WorkspaceContext) []string {
	var unregistered []string

	// Build a map of registered workspace paths for quick lookup
	registered := make(map[string]bool)
	for _, ws := range ctx.Manifest.Workspaces {
		registered[ws.Path] = true
	}

	// Walk the repository to find .git directories
	err := filepath.Walk(ctx.RepoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		// Skip the root .git
		if path == filepath.Join(ctx.RepoRoot, ".git") {
			return filepath.SkipDir
		}

		// Found a .git directory
		if info.IsDir() && info.Name() == ".git" {
			// Get relative path from repo root
			wsPath := filepath.Dir(path)
			relPath, err := filepath.Rel(ctx.RepoRoot, wsPath)
			if err != nil {
				return nil
			}

			// Skip if it's the root itself
			if relPath == "." {
				return filepath.SkipDir
			}

			// Skip package manager dependencies (they shouldn't be registered anyway)
			if shouldExcludeWorkspace(wsPath, relPath, ctx.RepoRoot) {
				return filepath.SkipDir
			}

			// Check if it's registered
			if !registered[relPath] {
				unregistered = append(unregistered, relPath)
			}

			// Don't descend into the workspace
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		// Silently continue if walk fails
		return unregistered
	}

	return unregistered
}

// findRemoteURLMismatches checks if workspace remote URLs match manifest
func findRemoteURLMismatches(ctx *common.WorkspaceContext) []IntegrityIssue {
	var issues []IntegrityIssue

	for _, ws := range ctx.Manifest.Workspaces {
		wsPath := filepath.Join(ctx.RepoRoot, ws.Path)

		// Skip if not cloned
		if !git.IsRepo(wsPath) {
			continue
		}

		// Get actual remote URL
		actualURL, err := git.GetRemoteURL(wsPath)
		if err != nil {
			continue // Skip if no remote configured
		}

		// Compare with manifest
		if actualURL != ws.Repo {
			issues = append(issues, IntegrityIssue{
				Level:   "warning",
				Message: i18n.T("remote_url_mismatch"),
				Path:    ws.Path,
				Fix:     fmt.Sprintf("Expected: %s\nActual: %s", ws.Repo, actualURL),
			})
		}
	}

	return issues
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Define color printers
	// Use Fprintf to always print to the correct stdout
	var (
		printCyan   = func(format string, a ...interface{}) { color.New(color.FgCyan, color.Bold).Fprintf(os.Stdout, format, a...) }
		printBlue   = func(format string, a ...interface{}) { color.New(color.FgBlue, color.Bold).Fprintf(os.Stdout, format, a...) }
		printGreen  = func(format string, a ...interface{}) { color.New(color.FgGreen).Fprintf(os.Stdout, format, a...) }
		printYellow = func(format string, a ...interface{}) { color.New(color.FgYellow).Fprintf(os.Stdout, format, a...) }
		printRed    = func(format string, a ...interface{}) { color.New(color.FgRed, color.Bold).Fprintf(os.Stdout, format, a...) }
		printGray   = func(format string, a ...interface{}) { color.New(color.Faint).Fprintf(os.Stdout, format, a...) }
	)

	ctx, err := common.LoadWorkspaceContext()
	if err != nil {
		return err
	}

	// Section 0: Multirepo Integrity Check
	printGray("%s\n", strings.Repeat("━", 80))
	printBlue("%s\n", i18n.T("integrity_check"))
	printGray("%s\n\n", strings.Repeat("━", 80))

	issues := validateMultirepoIntegrity(ctx)

	if len(issues) == 0 {
		// All clean
		printGreen("%s\n", i18n.T("no_nested_manifests"))
		printGreen("%s\n", i18n.T("no_parent_manifest"))
		printGreen("%s\n", i18n.T("all_workspaces_registered"))
	} else {
		// Display issues grouped by level
		for _, issue := range issues {
			switch issue.Level {
			case "critical":
				printRed("%s\n", issue.Message)
				if issue.Path != "" {
					printRed(i18n.T("nested_manifest_path") + "\n", issue.Path)
				}
				fmt.Println()
				printGray("%s\n", i18n.T("nested_manifest_explanation"))
				printGray("%s\n", i18n.T("nested_manifest_fix"))
				printGray(i18n.T("nested_manifest_cmd") + "\n", issue.Fix)
				fmt.Println()

			case "warning":
				if strings.Contains(issue.Message, "Parent manifest") || strings.Contains(issue.Message, "부모 manifest") {
					printYellow("%s\n", issue.Message)
					printYellow(i18n.T("parent_manifest_path") + "\n", issue.Path)
					printGray("%s\n", i18n.T("parent_manifest_explanation"))
					fmt.Println()
				} else if strings.Contains(issue.Message, "unregistered") || strings.Contains(issue.Message, "미등록") {
					printYellow("%s\n", issue.Message)
					for _, wsPath := range strings.Split(issue.Path, "\n") {
						if wsPath != "" {
							printGray(i18n.T("unregistered_workspace_item") + "\n", wsPath)
						}
					}
					fmt.Println()
					printGray("%s\n", i18n.T("unregistered_workspace_fix"))
					printGray("    %s\n", issue.Fix)
					fmt.Println()
				} else if strings.Contains(issue.Message, "Remote URL") || strings.Contains(issue.Message, "Remote URL") {
					printYellow("%s\n", issue.Message)
					printGray(i18n.T("remote_url_workspace") + "\n", issue.Path)
					lines := strings.Split(issue.Fix, "\n")
					for _, line := range lines {
						if strings.HasPrefix(line, "Expected:") {
							printGray(i18n.T("remote_url_expected") + "\n", strings.TrimPrefix(line, "Expected: "))
						} else if strings.HasPrefix(line, "Actual:") {
							printGray(i18n.T("remote_url_actual") + "\n", strings.TrimPrefix(line, "Actual: "))
						}
					}
					fmt.Println()
				} else if strings.Contains(issue.Message, "Local path repo") {
					printYellow("⚠ %s: %s\n", issue.Path, issue.Message)
					lines := strings.Split(issue.Fix, "\n")
					for _, line := range lines {
						printGray("    %s\n", line)
					}
					fmt.Println()
				} else if strings.Contains(issue.Message, "Package manager dependency") {
					printYellow("⚠ %s: %s\n", issue.Path, issue.Message)
					lines := strings.Split(issue.Fix, "\n")
					for _, line := range lines {
						printGray("    %s\n", line)
					}
					fmt.Println()
				}
			}
		}
	}

	printGray("%s\n\n", strings.Repeat("━", 80))

	if len(ctx.Manifest.Workspaces) == 0 {
		fmt.Println(i18n.T("no_subs_registered"))
		return nil
	}

	// Filter workspaces if path argument provided
	workspacesToProcess, err := ctx.FilterWorkspaces(args)
	if err != nil {
		return fmt.Errorf("%s", i18n.T("sub_not_found", args[0]))
	}

	for idx, ws := range workspacesToProcess {
		if idx > 0 {
			// Add separator between workspaces
			printGray("%s\n", strings.Repeat("─", 80))
			fmt.Println()
		}

		fullPath := filepath.Join(ctx.RepoRoot, ws.Path)

		// Workspace header
		printCyan("%s", ws.Path)

		if !git.IsRepo(fullPath) {
			printRed(" %s\n", i18n.T("not_cloned"))
			fmt.Println()
			printBlue("  %s\n", i18n.T("how_to_resolve"))
			printGray("    git multirepo sync\n")
			fmt.Println()
			continue
		}

		// Get current branch
		branch, err := git.GetCurrentBranch(fullPath)
		if err != nil {
			branch = "unknown"
		}
		printGray(" (%s)\n", branch)
		fmt.Println()

		// Section 1: Local Status
		printBlue("  %s\n", i18n.T("local_status"))

		// Get workspace status using unified pattern
		status, err := git.GetWorkspaceStatus(fullPath, ws.Keep)
		hasLocalChanges := false
		if err != nil {
			printRed("    Failed to get status: %v\n", err)
		} else {
			if len(status.ModifiedFiles) > 0 {
				hasLocalChanges = true
				printYellow("    %s\n", i18n.T("files_modified", len(status.ModifiedFiles)))
				for _, file := range status.ModifiedFiles {
					printGray("      - %s\n", file)
				}
			}

			if len(status.UntrackedFiles) > 0 {
				hasLocalChanges = true
				printYellow("    %s\n", i18n.T("files_untracked", len(status.UntrackedFiles)))
				for _, file := range status.UntrackedFiles {
					printGray("      - %s\n", file)
				}
			}

			if len(status.StagedFiles) > 0 {
				hasLocalChanges = true
				printYellow("    %s\n", i18n.T("files_staged", len(status.StagedFiles)))
				for _, file := range status.StagedFiles {
					printGray("      - %s\n", file)
				}
			}

			if !hasLocalChanges {
				printGreen("    %s\n", i18n.T("clean_working_tree"))
			}
		}
		fmt.Println()

		// Section 2: Remote Status
		printBlue("  %s\n", i18n.T("remote_status"))

		// Fetch from remote only if --fetch flag is set
		if statusFetch {
			if err := git.Fetch(fullPath); err != nil {
				if err == git.ErrFetchTimeout {
					printYellow("    ⚠ Fetch timed out, using cached data\n")
				}
			}
		}

		behindCount, _ := git.GetBehindCount(fullPath, branch)
		aheadCount, _ := git.GetAheadCount(fullPath, branch)

		if behindCount > 0 {
			printYellow("    %s\n", i18n.T("commits_behind", behindCount, branch))
		}

		if aheadCount > 0 {
			printYellow("    %s\n", i18n.T("commits_ahead", aheadCount))
		}

		if behindCount == 0 && aheadCount == 0 {
			printGreen("    %s\n", i18n.T("up_to_date"))
		}

		fmt.Println()

		// Section 3: How to resolve
		needsResolution := hasLocalChanges || behindCount > 0 || aheadCount > 0

		if needsResolution {
			printBlue("  %s\n", i18n.T("how_to_resolve"))
			fmt.Println()

			if hasLocalChanges {
				printYellow("    %s\n", i18n.T("resolve_commit"))
				printGray("       cd %s\n", ws.Path)
				if len(status.StagedFiles) > 0 || len(status.ModifiedFiles) > 0 {
					printGray("       git add .\n")
					printGray("       git commit -m \"your message\"\n")
				}
				if len(status.UntrackedFiles) > 0 {
					printGray("       %s\n", i18n.T("resolve_or_gitignore"))
				}
				fmt.Println()
			}

			if behindCount > 0 {
				printYellow("    %s\n", i18n.T("resolve_pull"))
				printGray("       git multirepo pull %s\n", ws.Path)
				fmt.Println()
			}

			if aheadCount > 0 {
				printYellow("    %s\n", i18n.T("resolve_push"))
				printGray("       cd %s\n", ws.Path)
				printGray("       git push\n")
				fmt.Println()
			}
		} else {
			printGreen("  %s\n", i18n.T("no_action_needed"))
			fmt.Println()
		}
	}

	return nil
}
