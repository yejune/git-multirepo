// Package manifest handles .gitsubs file operations
package manifest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const FileName = ".gitsubs"

// marshalFunc is the function used to marshal YAML (allows testing)
var marshalFunc = yaml.Marshal

// Subclone represents a single subclone entry
type Subclone struct {
	Path string   `yaml:"path"`
	Repo string   `yaml:"repo"`
	Skip []string `yaml:"skip,omitempty"`
}

// Manifest represents the .gitsubs file structure
type Manifest struct {
	Skip      []string   `yaml:"skip,omitempty"`
	Ignore    []string   `yaml:"ignore,omitempty"`
	Subclones []Subclone `yaml:"subclones"`
}

// Load reads the manifest from the given directory
func Load(dir string) (*Manifest, error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{Subclones: []Subclone{}}, nil
		}
		return nil, err
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	if m.Subclones == nil {
		m.Subclones = []Subclone{}
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

	// Add blank line between subclones for better readability
	lines := string(data)
	// Insert blank line before each "- path:" except the first
	buf := bytes.NewBuffer(nil)
	inSubclones := false
	firstSubclone := true

	for _, line := range strings.Split(lines, "\n") {
		if strings.HasPrefix(line, "subclones:") {
			inSubclones = true
		}

		if inSubclones && strings.HasPrefix(line, "  - path:") {
			if !firstSubclone {
				buf.WriteString("\n")
			}
			firstSubclone = false
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
