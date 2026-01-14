package common

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yejune/git-multirepo/internal/git"
	"github.com/yejune/git-multirepo/internal/i18n"
	"github.com/yejune/git-multirepo/internal/manifest"
)

// WorkspaceContext holds the repository root and manifest for workspace operations
type WorkspaceContext struct {
	RepoRoot     string
	Manifest     *manifest.Manifest
	CurrentDir   string // Directory where command was executed
	ScanRootDir  string // Root directory to scan (for workspace subdirectory sync)
}

// LoadWorkspaceContext initializes workspace context by loading repository root and manifest
// If executed from a workspace subdirectory, it detects the parent manifest and sets ScanRootDir accordingly
func LoadWorkspaceContext() (*WorkspaceContext, error) {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get git repository root
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	// Try to find parent manifest
	manifestRoot, err := manifest.FindParent(currentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to search for parent manifest: %w", err)
	}

	// Determine actual root and scan directory
	var actualRoot string
	var scanRoot string

	if manifestRoot != "" {
		// Parent manifest found
		actualRoot = manifestRoot

		// If we're in a subdirectory of manifestRoot, set scanRoot to currentDir
		// Otherwise use manifestRoot
		relPath, err := filepath.Rel(manifestRoot, currentDir)
		if err == nil && relPath != "." && relPath != "" {
			// We're in a subdirectory - scan only from current directory
			scanRoot = currentDir
		} else {
			// We're at manifestRoot - scan everything
			scanRoot = manifestRoot
		}
	} else {
		// No parent manifest - use git repo root
		actualRoot = repoRoot
		scanRoot = repoRoot
	}

	// Load manifest from the actual root
	m, err := manifest.Load(actualRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	i18n.SetLanguage(m.GetLanguage())

	return &WorkspaceContext{
		RepoRoot:    actualRoot,
		Manifest:    m,
		CurrentDir:  currentDir,
		ScanRootDir: scanRoot,
	}, nil
}

// SaveManifest saves the current manifest to disk
func (ctx *WorkspaceContext) SaveManifest() error {
	return manifest.Save(ctx.RepoRoot, ctx.Manifest)
}
