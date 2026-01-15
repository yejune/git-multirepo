// Package manifest handles .git.multirepos file operations
package manifest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const FileName = ".git.multirepos"

// marshalFunc is the function used to marshal YAML (allows testing)
var marshalFunc = yaml.Marshal

// WorkspaceEntry represents a single workspace entry
type WorkspaceEntry struct {
	Path   string   `yaml:"path"`
	Repo   string   `yaml:"repo"`
	Branch string   `yaml:"branch,omitempty"`
	Keep   []string `yaml:"keep,omitempty"`
	Commit string   `yaml:"commit,omitempty"` // Deprecated: kept for backward compatibility, no longer used
}

// Manifest represents the .git.multirepos file structure
type Manifest struct {
	Language   string           `yaml:"language,omitempty"`
	Keep       []string         `yaml:"keep,omitempty"`   // Mother repo: files to keep
	Ignore     []string         `yaml:"ignore,omitempty"` // Mother repo: files to ignore (gitignore-style)
	Workspaces []WorkspaceEntry `yaml:"workspaces,omitempty"`
}

// Load reads the manifest from the given directory
func Load(dir string) (*Manifest, error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{
				Workspaces: []WorkspaceEntry{},
			}, nil
		}
		return nil, err
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	// Initialize empty slice if nil
	if m.Workspaces == nil {
		m.Workspaces = []WorkspaceEntry{}
	}

	return &m, nil
}

// Save writes the manifest to the given directory
func Save(dir string, m *Manifest) error {
	path := filepath.Join(dir, FileName)
	data, err := marshalFunc(m)
	if err != nil {
		return err
	}

	// Add blank line between workspaces for better readability
	lines := string(data)
	// Insert blank line before each "- path:" except the first
	buf := bytes.NewBuffer(nil)
	inWorkspaces := false
	firstEntry := true

	for _, line := range strings.Split(lines, "\n") {
		// Detect "workspaces:"
		if strings.HasPrefix(line, "workspaces:") {
			inWorkspaces = true
			firstEntry = true
		}

		if inWorkspaces && strings.HasPrefix(line, "  - path:") {
			if !firstEntry {
				buf.WriteString("\n")
			}
			firstEntry = false
		}

		buf.WriteString(line)
		buf.WriteString("\n")
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// Add adds a new workspace to the manifest
func (m *Manifest) Add(path, repo string) {
	m.Workspaces = append(m.Workspaces, WorkspaceEntry{
		Path: path,
		Repo: repo,
	})
}

// Remove removes a workspace from the manifest by path
func (m *Manifest) Remove(path string) bool {
	for i, ws := range m.Workspaces {
		if ws.Path == path {
			m.Workspaces = append(m.Workspaces[:i], m.Workspaces[i+1:]...)
			return true
		}
	}
	return false
}

// Find finds a workspace by path
func (m *Manifest) Find(path string) *WorkspaceEntry {
	for i := range m.Workspaces {
		if m.Workspaces[i].Path == path {
			return &m.Workspaces[i]
		}
	}
	return nil
}

// Exists checks if a workspace exists at the given path
func (m *Manifest) Exists(path string) bool {
	return m.Find(path) != nil
}

// GetLanguage returns the configured language, defaults to "en"
func (m *Manifest) GetLanguage() string {
	if m.Language == "" {
		return "en"
	}
	return m.Language
}

// FindParent searches for .git.multirepo file in parent directories
// Returns the directory containing the manifest, or empty string if not found
func FindParent(startDir string) (string, error) {
	current := startDir
	for {
		manifestPath := filepath.Join(current, FileName)
		if _, err := os.Stat(manifestPath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root without finding manifest
			return "", nil
		}
		current = parent
	}
}
