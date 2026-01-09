package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(m.Workspaces) != 0 {
		t.Errorf("expected 0 workspaces, got %d", len(m.Workspaces))
	}
}

func TestLoadReadError(t *testing.T) {
	dir := t.TempDir()
	// Create .workspaces as a directory - ReadFile will fail with non-NotExist error
	manifestPath := filepath.Join(dir, FileName)
	os.MkdirAll(manifestPath, 0755)

	_, err := Load(dir)
	if err == nil {
		t.Error("Load should fail when manifest path is a directory")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, FileName)
	// Write invalid YAML content
	os.WriteFile(manifestPath, []byte("workspaces: [invalid yaml\n  - broken"), 0644)

	_, err := Load(dir)
	if err == nil {
		t.Error("Load should fail when YAML is invalid")
	}
}

func TestLoadNilWorkspaces(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, FileName)
	// Write YAML without workspaces field (will be nil)
	os.WriteFile(manifestPath, []byte("# empty manifest\n"), 0644)

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if m.Workspaces == nil {
		t.Error("Workspaces should be initialized to empty slice, not nil")
	}
	if len(m.Workspaces) != 0 {
		t.Errorf("expected 0 workspaces, got %d", len(m.Workspaces))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	m := &Manifest{
		Workspaces: []WorkspaceEntry{
			{Path: "packages/sub-a", Repo: "https://github.com/test/sub-a.git"},
			{Path: "libs/sub-b", Repo: "https://github.com/test/sub-b.git"},
		},
	}

	if err := Save(dir, m); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check file exists
	path := filepath.Join(dir, FileName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("manifest file not created")
	}

	// Load and verify
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Workspaces) != 2 {
		t.Errorf("expected 2 workspaces, got %d", len(loaded.Workspaces))
	}

	if loaded.Workspaces[0].Path != "packages/sub-a" {
		t.Errorf("expected path packages/sub-a, got %s", loaded.Workspaces[0].Path)
	}

	// Branch field removed in v0.1.0
}

func TestAddAndRemove(t *testing.T) {
	m := &Manifest{Workspaces: []WorkspaceEntry{}}

	m.Add("test/path", "https://github.com/test/repo.git")

	if len(m.Workspaces) != 1 {
		t.Errorf("expected 1 workspace, got %d", len(m.Workspaces))
	}

	if !m.Exists("test/path") {
		t.Error("expected workspace to exist")
	}

	// Verify the workspace was added correctly
	sc := m.Find("test/path")
	if sc == nil {
		t.Fatal("expected to find workspace")
	}
	if sc.Repo != "https://github.com/test/repo.git" {
		t.Errorf("expected repo https://github.com/test/repo.git, got %s", sc.Repo)
	}

	if !m.Remove("test/path") {
		t.Error("expected Remove to return true")
	}

	if m.Exists("test/path") {
		t.Error("expected workspace to not exist")
	}

	if m.Remove("nonexistent") {
		t.Error("expected Remove to return false for nonexistent path")
	}
}

func TestFind(t *testing.T) {
	m := &Manifest{
		Workspaces: []WorkspaceEntry{
			{Path: "a", Repo: "repo-a"},
			{Path: "b", Repo: "repo-b"},
		},
	}

	sc := m.Find("a")
	if sc == nil {
		t.Fatal("expected to find workspace")
	}
	if sc.Repo != "repo-a" {
		t.Errorf("expected repo-a, got %s", sc.Repo)
	}

	if m.Find("nonexistent") != nil {
		t.Error("expected nil for nonexistent path")
	}
}

func TestSaveWriteError(t *testing.T) {
	dir := t.TempDir()
	// Create .workspaces as a directory to prevent WriteFile
	manifestPath := filepath.Join(dir, FileName)
	os.MkdirAll(manifestPath, 0755)

	m := &Manifest{Workspaces: []WorkspaceEntry{{Path: "test", Repo: "repo"}}}
	err := Save(dir, m)
	if err == nil {
		t.Error("Save should fail when manifest path is a directory")
	}
}

func TestSaveMarshalError(t *testing.T) {
	dir := t.TempDir()

	// Replace marshalFunc with one that always fails
	originalMarshal := marshalFunc
	marshalFunc = func(v interface{}) ([]byte, error) {
		return nil, os.ErrInvalid
	}
	defer func() { marshalFunc = originalMarshal }()

	m := &Manifest{Workspaces: []WorkspaceEntry{{Path: "test", Repo: "repo"}}}
	err := Save(dir, m)
	if err == nil {
		t.Error("Save should fail when marshal fails")
	}
}
