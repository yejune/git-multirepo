// Package manifest handles .workspaces file operations
package manifest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const FileName = ".workspaces"

// marshalFunc is the function used to marshal YAML (allows testing)
var marshalFunc = yaml.Marshal

// WorkspaceEntry represents a single workspace entry (subclone or mother repo)
type WorkspaceEntry struct {
	Path   string   `yaml:"path"`
	Repo   string   `yaml:"repo"`
	Branch string   `yaml:"branch,omitempty"`
	Commit string   `yaml:"commit,omitempty"`
	Keep   []string `yaml:"keep,omitempty"`
	Skip   []string `yaml:"skip,omitempty"` // Deprecated: use Keep instead
}

// Subclone is an alias for backward compatibility
type Subclone = WorkspaceEntry

// Manifest represents the .workspaces file structure
type Manifest struct {
	Language   string           `yaml:"language,omitempty"`
	Skip       []string         `yaml:"skip,omitempty"`   // Deprecated: use Keep instead
	Keep       []string         `yaml:"keep,omitempty"`   // Mother repo: files to keep
	Ignore     []string         `yaml:"ignore,omitempty"` // Mother repo: files to ignore (gitignore-style)
	Workspaces []WorkspaceEntry `yaml:"workspaces,omitempty"`
	Subclones  []WorkspaceEntry `yaml:"subclones,omitempty"` // Deprecated: use Workspaces instead
}

// Load reads the manifest from the given directory
func Load(dir string) (*Manifest, error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{
				Workspaces: []WorkspaceEntry{},
				Subclones:  []WorkspaceEntry{},
			}, nil
		}
		return nil, err
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	// Initialize empty slices if nil for backward compatibility
	if m.Workspaces == nil {
		m.Workspaces = []WorkspaceEntry{}
	}
	if m.Subclones == nil {
		m.Subclones = []WorkspaceEntry{}
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

	// Add blank line between workspaces/subclones for better readability
	lines := string(data)
	// Insert blank line before each "- path:" except the first
	buf := bytes.NewBuffer(nil)
	inWorkspaces := false
	firstEntry := true

	for _, line := range strings.Split(lines, "\n") {
		// Detect both "workspaces:" and "subclones:" for backward compatibility
		if strings.HasPrefix(line, "workspaces:") || strings.HasPrefix(line, "subclones:") {
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

// Add adds a new subclone to the manifest
func (m *Manifest) Add(path, repo string) {
	m.Subclones = append(m.Subclones, Subclone{
		Path: path,
		Repo: repo,
	})
}

// Remove removes a subclone from the manifest by path
func (m *Manifest) Remove(path string) bool {
	for i, sc := range m.Subclones {
		if sc.Path == path {
			m.Subclones = append(m.Subclones[:i], m.Subclones[i+1:]...)
			return true
		}
	}
	return false
}

// UpdateCommit updates the commit hash for a subclone
func (m *Manifest) UpdateCommit(path, commit string) bool {
	for i, sc := range m.Subclones {
		if sc.Path == path {
			m.Subclones[i].Commit = commit
			return true
		}
	}
	return false
}

// Find finds a subclone by path
func (m *Manifest) Find(path string) *Subclone {
	for i := range m.Subclones {
		if m.Subclones[i].Path == path {
			return &m.Subclones[i]
		}
	}
	return nil
}

// Exists checks if a subclone exists at the given path
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

// GetWorkspaces returns workspaces, falling back to subclones for backward compatibility
func (m *Manifest) GetWorkspaces() []WorkspaceEntry {
	if len(m.Workspaces) > 0 {
		return m.Workspaces
	}
	return m.Subclones
}

// GetKeepFiles returns the Keep list for a WorkspaceEntry, falling back to Skip for backward compatibility
func (w *WorkspaceEntry) GetKeepFiles() []string {
	if len(w.Keep) > 0 {
		return w.Keep
	}
	return w.Skip
}
