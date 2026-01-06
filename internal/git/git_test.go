package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "-C", dir, "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()

	return dir
}

// setupTestRepoWithCommit creates a repo with an initial commit
func setupTestRepoWithCommit(t *testing.T) string {
	t.Helper()
	dir := setupTestRepo(t)

	// Create a file and commit
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "Initial commit").Run()

	return dir
}

func TestIsRepo(t *testing.T) {
	t.Run("valid git repo", func(t *testing.T) {
		dir := setupTestRepo(t)
		if !IsRepo(dir) {
			t.Errorf("IsRepo(%q) = false, want true", dir)
		}
	})

	t.Run("not a git repo", func(t *testing.T) {
		dir := t.TempDir()
		if IsRepo(dir) {
			t.Errorf("IsRepo(%q) = true, want false", dir)
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		if IsRepo("/non/existent/path") {
			t.Error("IsRepo for non-existent path = true, want false")
		}
	})
}

func TestAddToGitignore(t *testing.T) {
	t.Run("add to empty gitignore", func(t *testing.T) {
		dir := t.TempDir()

		err := AddToGitignore(dir, "subdir")
		if err != nil {
			t.Fatalf("AddToGitignore failed: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
		if !strings.Contains(string(content), "subdir/.git/") {
			t.Errorf("gitignore should contain 'subdir/.git/', got: %s", content)
		}
	})

	t.Run("add to existing gitignore", func(t *testing.T) {
		dir := t.TempDir()
		gitignore := filepath.Join(dir, ".gitignore")
		os.WriteFile(gitignore, []byte("node_modules/\n"), 0644)

		err := AddToGitignore(dir, "packages/lib")
		if err != nil {
			t.Fatalf("AddToGitignore failed: %v", err)
		}

		content, _ := os.ReadFile(gitignore)
		if !strings.Contains(string(content), "node_modules/") {
			t.Error("should preserve existing content")
		}
		if !strings.Contains(string(content), "packages/lib/.git/") {
			t.Error("should add new entry")
		}
	})

	t.Run("no duplicate entries", func(t *testing.T) {
		dir := t.TempDir()

		AddToGitignore(dir, "lib")
		AddToGitignore(dir, "lib") // Add again

		content, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
		count := strings.Count(string(content), "lib/.git/")
		if count != 1 {
			t.Errorf("should have 1 entry, got %d", count)
		}
	})
}

func TestRemoveFromGitignore(t *testing.T) {
	t.Run("remove existing entry", func(t *testing.T) {
		dir := t.TempDir()
		gitignore := filepath.Join(dir, ".gitignore")
		os.WriteFile(gitignore, []byte("lib/.git/\nother/\n"), 0644)

		err := RemoveFromGitignore(dir, "lib")
		if err != nil {
			t.Fatalf("RemoveFromGitignore failed: %v", err)
		}

		content, _ := os.ReadFile(gitignore)
		if strings.Contains(string(content), "lib/.git/") {
			t.Error("should remove entry")
		}
		if !strings.Contains(string(content), "other/") {
			t.Error("should preserve other entries")
		}
	})

	t.Run("remove from non-existent gitignore", func(t *testing.T) {
		dir := t.TempDir()
		err := RemoveFromGitignore(dir, "lib")
		if err != nil {
			t.Errorf("should not error on non-existent file: %v", err)
		}
	})
}

func TestGetRepoRoot(t *testing.T) {
	// Save current directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	t.Run("from repo root", func(t *testing.T) {
		dir := setupTestRepo(t)
		// Resolve symlinks for macOS /var -> /private/var
		expectedDir, _ := filepath.EvalSymlinks(dir)
		os.Chdir(dir)

		root, err := GetRepoRoot()
		if err != nil {
			t.Fatalf("GetRepoRoot failed: %v", err)
		}
		if root != expectedDir {
			t.Errorf("GetRepoRoot() = %q, want %q", root, expectedDir)
		}
	})

	t.Run("from subdirectory", func(t *testing.T) {
		dir := setupTestRepo(t)
		// Resolve symlinks for macOS /var -> /private/var
		expectedDir, _ := filepath.EvalSymlinks(dir)
		subdir := filepath.Join(dir, "src", "pkg")
		os.MkdirAll(subdir, 0755)
		os.Chdir(subdir)

		root, err := GetRepoRoot()
		if err != nil {
			t.Fatalf("GetRepoRoot failed: %v", err)
		}
		if root != expectedDir {
			t.Errorf("GetRepoRoot() = %q, want %q", root, expectedDir)
		}
	})

	t.Run("not in a repo", func(t *testing.T) {
		dir := t.TempDir()
		os.Chdir(dir)

		_, err := GetRepoRoot()
		if err == nil {
			t.Error("should error when not in a repo")
		}
	})
}

func TestHasChanges(t *testing.T) {
	t.Run("clean repo", func(t *testing.T) {
		dir := setupTestRepoWithCommit(t)

		hasChanges, err := HasChanges(dir)
		if err != nil {
			t.Fatalf("HasChanges failed: %v", err)
		}
		if hasChanges {
			t.Error("clean repo should have no changes")
		}
	})

	t.Run("repo with uncommitted changes", func(t *testing.T) {
		dir := setupTestRepoWithCommit(t)

		// Create a new file
		os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)

		hasChanges, err := HasChanges(dir)
		if err != nil {
			t.Fatalf("HasChanges failed: %v", err)
		}
		if !hasChanges {
			t.Error("repo with new file should have changes")
		}
	})

	t.Run("repo with modified file", func(t *testing.T) {
		dir := setupTestRepoWithCommit(t)

		// Modify existing file
		os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Modified"), 0644)

		hasChanges, err := HasChanges(dir)
		if err != nil {
			t.Fatalf("HasChanges failed: %v", err)
		}
		if !hasChanges {
			t.Error("repo with modified file should have changes")
		}
	})
}

func TestGetCurrentBranch(t *testing.T) {
	t.Run("default branch", func(t *testing.T) {
		dir := setupTestRepoWithCommit(t)

		branch, err := GetCurrentBranch(dir)
		if err != nil {
			t.Fatalf("GetCurrentBranch failed: %v", err)
		}
		// Could be "main" or "master" depending on git config
		if branch != "main" && branch != "master" {
			t.Errorf("unexpected branch: %s", branch)
		}
	})

	t.Run("custom branch", func(t *testing.T) {
		dir := setupTestRepoWithCommit(t)
		exec.Command("git", "-C", dir, "checkout", "-b", "feature").Run()

		branch, err := GetCurrentBranch(dir)
		if err != nil {
			t.Fatalf("GetCurrentBranch failed: %v", err)
		}
		if branch != "feature" {
			t.Errorf("GetCurrentBranch() = %q, want 'feature'", branch)
		}
	})
}

func TestGetRemoteURL(t *testing.T) {
	t.Run("repo with remote", func(t *testing.T) {
		dir := setupTestRepo(t)
		exec.Command("git", "-C", dir, "remote", "add", "origin", "https://github.com/test/repo.git").Run()

		url, err := GetRemoteURL(dir)
		if err != nil {
			t.Fatalf("GetRemoteURL failed: %v", err)
		}
		if url != "https://github.com/test/repo.git" {
			t.Errorf("GetRemoteURL() = %q, want 'https://github.com/test/repo.git'", url)
		}
	})

	t.Run("repo without remote", func(t *testing.T) {
		dir := setupTestRepo(t)

		_, err := GetRemoteURL(dir)
		if err == nil {
			t.Error("should error when no remote configured")
		}
	})
}

func TestClone(t *testing.T) {
	t.Run("clone local repo", func(t *testing.T) {
		// Create source repo
		srcDir := setupTestRepoWithCommit(t)

		// Clone to destination
		dstDir := filepath.Join(t.TempDir(), "cloned")

		err := Clone(srcDir, dstDir, "")
		if err != nil {
			t.Fatalf("Clone failed: %v", err)
		}

		if !IsRepo(dstDir) {
			t.Error("cloned directory should be a git repo")
		}

		// Check file exists
		if _, err := os.Stat(filepath.Join(dstDir, "README.md")); os.IsNotExist(err) {
			t.Error("cloned repo should contain README.md")
		}
	})

	t.Run("clone with branch", func(t *testing.T) {
		// Create source repo with branch
		srcDir := setupTestRepoWithCommit(t)
		exec.Command("git", "-C", srcDir, "checkout", "-b", "develop").Run()
		os.WriteFile(filepath.Join(srcDir, "develop.txt"), []byte("develop"), 0644)
		exec.Command("git", "-C", srcDir, "add", ".").Run()
		exec.Command("git", "-C", srcDir, "commit", "-m", "Develop commit").Run()

		// Clone to destination
		dstDir := filepath.Join(t.TempDir(), "cloned")

		err := Clone(srcDir, dstDir, "develop")
		if err != nil {
			t.Fatalf("Clone failed: %v", err)
		}

		branch, _ := GetCurrentBranch(dstDir)
		if branch != "develop" {
			t.Errorf("cloned branch = %q, want 'develop'", branch)
		}
	})
}

func TestPull(t *testing.T) {
	t.Run("pull from remote", func(t *testing.T) {
		// Create source repo
		srcDir := setupTestRepoWithCommit(t)

		// Clone to destination
		dstDir := filepath.Join(t.TempDir(), "cloned")
		Clone(srcDir, dstDir, "")

		// Add new commit to source
		os.WriteFile(filepath.Join(srcDir, "new.txt"), []byte("new"), 0644)
		exec.Command("git", "-C", srcDir, "add", ".").Run()
		exec.Command("git", "-C", srcDir, "commit", "-m", "New commit").Run()

		// Pull in destination
		err := Pull(dstDir)
		if err != nil {
			t.Fatalf("Pull failed: %v", err)
		}

		// Check new file exists
		if _, err := os.Stat(filepath.Join(dstDir, "new.txt")); os.IsNotExist(err) {
			t.Error("pulled repo should contain new.txt")
		}
	})
}

func TestInitRepo(t *testing.T) {
	t.Run("init new repo", func(t *testing.T) {
		dir := t.TempDir()
		repoDir := filepath.Join(dir, "newrepo")
		os.MkdirAll(repoDir, 0755)

		// Create a bare repo as remote
		remoteDir := filepath.Join(dir, "remote.git")
		exec.Command("git", "init", "--bare", remoteDir).Run()

		err := InitRepo(repoDir, remoteDir, "main")
		if err != nil {
			t.Fatalf("InitRepo failed: %v", err)
		}

		if !IsRepo(repoDir) {
			t.Error("should create git repo")
		}

		url, _ := GetRemoteURL(repoDir)
		if url != remoteDir {
			t.Errorf("remote URL = %q, want %q", url, remoteDir)
		}
	})

	t.Run("init with empty branch", func(t *testing.T) {
		dir := t.TempDir()
		repoDir := filepath.Join(dir, "newrepo2")
		os.MkdirAll(repoDir, 0755)

		remoteDir := filepath.Join(dir, "remote2.git")
		exec.Command("git", "init", "--bare", remoteDir).Run()

		err := InitRepo(repoDir, remoteDir, "")
		if err != nil {
			t.Fatalf("InitRepo with empty branch failed: %v", err)
		}
	})
}

func TestPush(t *testing.T) {
	t.Run("push without remote", func(t *testing.T) {
		dir := setupTestRepoWithCommit(t)

		// Push should fail because no remote configured
		err := Push(dir)
		if err == nil {
			t.Error("push without remote should fail")
		}
	})

	t.Run("push to remote", func(t *testing.T) {
		// Create bare repo as remote
		remoteDir := t.TempDir()
		exec.Command("git", "init", "--bare", remoteDir).Run()

		// Create local repo
		localDir := setupTestRepoWithCommit(t)

		// Add remote and set upstream
		exec.Command("git", "-C", localDir, "remote", "add", "origin", remoteDir).Run()
		// Push with set-upstream
		exec.Command("git", "-C", localDir, "push", "-u", "origin", "main").Run()

		// Now Push should work
		err := Push(localDir)
		if err != nil {
			t.Errorf("push to remote failed: %v", err)
		}
	})
}
