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
	if len(m.Subclones) != 0 {
		t.Errorf("expected 0 subclones, got %d", len(m.Subclones))
	}
}

func TestLoadReadError(t *testing.T) {
	dir := t.TempDir()
	// Create .gitsubs as a directory - ReadFile will fail with non-NotExist error
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
	os.WriteFile(manifestPath, []byte("subclones: [invalid yaml\n  - broken"), 0644)

	_, err := Load(dir)
	if err == nil {
		t.Error("Load should fail when YAML is invalid")
	}
}

func TestLoadNilSubclones(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, FileName)
	// Write YAML without subclones field (will be nil)
	os.WriteFile(manifestPath, []byte("# empty manifest\n"), 0644)

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if m.Subclones == nil {
		t.Error("Subclones should be initialized to empty slice, not nil")
	}
	if len(m.Subclones) != 0 {
		t.Errorf("expected 0 subclones, got %d", len(m.Subclones))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	m := &Manifest{
		Subclones: []Subclone{
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

	if len(loaded.Subclones) != 2 {
		t.Errorf("expected 2 subclones, got %d", len(loaded.Subclones))
	}

	if loaded.Subclones[0].Path != "packages/sub-a" {
		t.Errorf("expected path packages/sub-a, got %s", loaded.Subclones[0].Path)
	}

	// Branch field removed in v0.1.0
}

func TestAddAndRemove(t *testing.T) {
	m := &Manifest{Subclones: []Subclone{}}

	m.Add("test/path", "https://github.com/test/repo.git")

	if len(m.Subclones) != 1 {
		t.Errorf("expected 1 subclone, got %d", len(m.Subclones))
	}

	if !m.Exists("test/path") {
		t.Error("expected subclone to exist")
	}

	// Verify the subclone was added correctly
	sc := m.Find("test/path")
	if sc == nil {
		t.Fatal("expected to find subclone")
	}
	if sc.Repo != "https://github.com/test/repo.git" {
		t.Errorf("expected repo https://github.com/test/repo.git, got %s", sc.Repo)
	}

	if !m.Remove("test/path") {
		t.Error("expected Remove to return true")
	}

	if m.Exists("test/path") {
		t.Error("expected subclone to not exist")
	}

	if m.Remove("nonexistent") {
		t.Error("expected Remove to return false for nonexistent path")
	}
}

func TestFind(t *testing.T) {
	m := &Manifest{
		Subclones: []Subclone{
			{Path: "a", Repo: "repo-a"},
			{Path: "b", Repo: "repo-b"},
		},
	}

	sc := m.Find("a")
	if sc == nil {
		t.Fatal("expected to find subclone")
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
	// Create .gitsubs as a directory to prevent WriteFile
	manifestPath := filepath.Join(dir, FileName)
	os.MkdirAll(manifestPath, 0755)

	m := &Manifest{Subclones: []Subclone{{Path: "test", Repo: "repo"}}}
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

	m := &Manifest{Subclones: []Subclone{{Path: "test", Repo: "repo"}}}
	err := Save(dir, m)
	if err == nil {
		t.Error("Save should fail when marshal fails")
	}
}
