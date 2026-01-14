package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ============================================================================
// Package Manager Exclusion Tests
// ============================================================================

// TestMatchesExcludePattern_SwiftPM tests Swift Package Manager checkout exclusion
func TestMatchesExcludePattern_SwiftPM(t *testing.T) {
	tests := []struct {
		relPath  string
		expected bool
	}{
		// Should be excluded
		{".build/checkouts/SomePackage", true},
		{"app/.build/checkouts/SomePackage", true},
		{"nested/deep/.build/checkouts/Package", true},

		// Should NOT be excluded (different patterns)
		{".build/products", false},
		{".build/Build/Products", false},
		{"checkouts/SomePackage", false}, // Just "checkouts" without ".build/"
		{"my-checkouts/something", false},
		{"soju", false},
		{"app", false},
	}

	for _, tt := range tests {
		t.Run(tt.relPath, func(t *testing.T) {
			result := matchesExcludePattern(tt.relPath)
			if result != tt.expected {
				t.Errorf("matchesExcludePattern(%q) = %v, want %v", tt.relPath, result, tt.expected)
			}
		})
	}
}

// TestMatchesExcludePattern_XcodeSPM tests Xcode SPM checkout exclusion
func TestMatchesExcludePattern_XcodeSPM(t *testing.T) {
	tests := []struct {
		relPath  string
		expected bool
	}{
		// Should be excluded
		{"build/SourcePackages/checkouts/Package", true},
		{"app/build/SourcePackages/checkouts/SemanticVersion", true},
		{"DerivedData/Build/SourcePackages/checkouts/Pkg", true},

		// Should NOT be excluded
		{"SourcePackages/checkouts/Package", true}, // Still matches pattern
		{"build/SourcePackages/artifacts", false},
		{"build/Products", false},
	}

	for _, tt := range tests {
		t.Run(tt.relPath, func(t *testing.T) {
			result := matchesExcludePattern(tt.relPath)
			if result != tt.expected {
				t.Errorf("matchesExcludePattern(%q) = %v, want %v", tt.relPath, result, tt.expected)
			}
		})
	}
}

// TestMatchesExcludePattern_Carthage tests Carthage checkout exclusion
func TestMatchesExcludePattern_Carthage(t *testing.T) {
	tests := []struct {
		relPath  string
		expected bool
	}{
		// Should be excluded (case-sensitive: Carthage/Checkouts)
		{"Carthage/Checkouts/Alamofire", true},
		{"ios/Carthage/Checkouts/SnapKit", true},

		// Should NOT be excluded (wrong case)
		{"carthage/checkouts/Package", false}, // lowercase
		{"Carthage/Build/iOS", false},
	}

	for _, tt := range tests {
		t.Run(tt.relPath, func(t *testing.T) {
			result := matchesExcludePattern(tt.relPath)
			if result != tt.expected {
				t.Errorf("matchesExcludePattern(%q) = %v, want %v", tt.relPath, result, tt.expected)
			}
		})
	}
}

// TestIsExcludedByMarker_NodeModules tests npm/yarn node_modules exclusion
func TestIsExcludedByMarker_NodeModules(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create package.json (marker file)
	packageJSON := filepath.Join(tmpDir, "package.json")
	os.WriteFile(packageJSON, []byte(`{"name": "test"}`), 0644)

	// Create node_modules structure with .git
	nodeModulesRepo := filepath.Join(tmpDir, "node_modules", "some-package")
	os.MkdirAll(nodeModulesRepo, 0755)

	// Test: node_modules/some-package should be excluded because package.json exists
	result := isExcludedByMarker(nodeModulesRepo, tmpDir)
	if !result {
		t.Error("Expected node_modules/some-package to be excluded (package.json exists)")
	}

	// Test: without marker file
	tmpDir2 := t.TempDir()
	nodeModulesRepo2 := filepath.Join(tmpDir2, "node_modules", "some-package")
	os.MkdirAll(nodeModulesRepo2, 0755)

	result2 := isExcludedByMarker(nodeModulesRepo2, tmpDir2)
	if result2 {
		t.Error("Expected node_modules/some-package NOT to be excluded (no package.json)")
	}
}

// TestIsExcludedByMarker_ComposerVendor tests PHP Composer vendor exclusion
func TestIsExcludedByMarker_ComposerVendor(t *testing.T) {
	tmpDir := t.TempDir()

	// Create composer.json (marker file)
	composerJSON := filepath.Join(tmpDir, "composer.json")
	os.WriteFile(composerJSON, []byte(`{"name": "test/project"}`), 0644)

	// Create vendor structure
	vendorRepo := filepath.Join(tmpDir, "vendor", "some-vendor", "package")
	os.MkdirAll(vendorRepo, 0755)

	// Test: vendor/some-vendor/package should be excluded
	result := isExcludedByMarker(vendorRepo, tmpDir)
	if !result {
		t.Error("Expected vendor/some-vendor/package to be excluded (composer.json exists)")
	}
}

// TestIsExcludedByMarker_NestedProject tests nested project with marker file
func TestIsExcludedByMarker_NestedProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested project structure: tmpDir/subproject/package.json + node_modules
	subproject := filepath.Join(tmpDir, "subproject")
	os.MkdirAll(subproject, 0755)

	// Create package.json in subproject
	packageJSON := filepath.Join(subproject, "package.json")
	os.WriteFile(packageJSON, []byte(`{"name": "subproject"}`), 0644)

	// Create node_modules in subproject
	nodeModulesRepo := filepath.Join(subproject, "node_modules", "dependency")
	os.MkdirAll(nodeModulesRepo, 0755)

	// Test: subproject/node_modules/dependency should be excluded
	result := isExcludedByMarker(nodeModulesRepo, tmpDir)
	if !result {
		t.Error("Expected subproject/node_modules/dependency to be excluded")
	}
}

// TestShouldExcludeWorkspace_Combined tests combined exclusion logic
func TestShouldExcludeWorkspace_Combined(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{}`), 0644)

	tests := []struct {
		relPath  string
		expected bool
	}{
		// Pattern-based exclusions
		{".build/checkouts/SwiftPkg", true},
		{"app/build/SourcePackages/checkouts/Pkg", true},
		{"Carthage/Checkouts/Framework", true},

		// These would need marker file check (node_modules with package.json)
		{"node_modules/lodash", true}, // package.json exists

		// Should NOT be excluded
		{"app", false},
		{"soju", false},
		{"wine-fork", false},
		{"packages/shared", false},
	}

	for _, tt := range tests {
		t.Run(tt.relPath, func(t *testing.T) {
			absPath := filepath.Join(tmpDir, tt.relPath)
			os.MkdirAll(absPath, 0755)

			result := shouldExcludeWorkspace(absPath, tt.relPath, tmpDir)
			if result != tt.expected {
				t.Errorf("shouldExcludeWorkspace(%q) = %v, want %v", tt.relPath, result, tt.expected)
			}
		})
	}
}

// TestDiscoverWorkspaces_ExcludesPackageManagerDeps tests workspace discovery excludes package manager dependencies
func TestDiscoverWorkspaces_ExcludesPackageManagerDeps(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize parent repo
	exec.Command("git", "-C", tmpDir, "init").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", tmpDir, "commit", "--allow-empty", "-m", "init").Run()

	// Create legitimate workspace
	legitWorkspace := filepath.Join(tmpDir, "packages", "my-lib")
	os.MkdirAll(legitWorkspace, 0755)
	exec.Command("git", "-C", legitWorkspace, "init").Run()
	exec.Command("git", "-C", legitWorkspace, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", legitWorkspace, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", legitWorkspace, "commit", "--allow-empty", "-m", "init").Run()

	// Create Swift PM checkout (should be excluded)
	spmCheckout := filepath.Join(tmpDir, ".build", "checkouts", "SomePackage")
	os.MkdirAll(spmCheckout, 0755)
	exec.Command("git", "-C", spmCheckout, "init").Run()
	exec.Command("git", "-C", spmCheckout, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", spmCheckout, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", spmCheckout, "commit", "--allow-empty", "-m", "init").Run()

	// Create Xcode SPM checkout (should be excluded)
	xcodeSPM := filepath.Join(tmpDir, "build", "SourcePackages", "checkouts", "OtherPackage")
	os.MkdirAll(xcodeSPM, 0755)
	exec.Command("git", "-C", xcodeSPM, "init").Run()
	exec.Command("git", "-C", xcodeSPM, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", xcodeSPM, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", xcodeSPM, "commit", "--allow-empty", "-m", "init").Run()

	// Discover workspaces
	discoveries, err := discoverWorkspaces(tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("discoverWorkspaces failed: %v", err)
	}

	// Collect discovered paths
	var foundPaths []string
	for d := range discoveries {
		foundPaths = append(foundPaths, d.relPath)
	}

	// Verify: only legitimate workspace should be found
	expectedPath := filepath.Join("packages", "my-lib")
	found := false
	for _, p := range foundPaths {
		if p == expectedPath {
			found = true
		}
		// Check that excluded paths are not present
		if p == filepath.Join(".build", "checkouts", "SomePackage") {
			t.Error("Swift PM checkout should have been excluded")
		}
		if p == filepath.Join("build", "SourcePackages", "checkouts", "OtherPackage") {
			t.Error("Xcode SPM checkout should have been excluded")
		}
	}

	if !found {
		t.Errorf("Expected to find %s, found paths: %v", expectedPath, foundPaths)
	}
}

// TestDiscoverWorkspaces_ExcludesNodeModulesWithMarker tests node_modules exclusion with package.json
func TestDiscoverWorkspaces_ExcludesNodeModulesWithMarker(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize parent repo
	exec.Command("git", "-C", tmpDir, "init").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", tmpDir, "commit", "--allow-empty", "-m", "init").Run()

	// Create package.json
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644)

	// Create legitimate workspace
	legitWorkspace := filepath.Join(tmpDir, "src", "lib")
	os.MkdirAll(legitWorkspace, 0755)
	exec.Command("git", "-C", legitWorkspace, "init").Run()
	exec.Command("git", "-C", legitWorkspace, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", legitWorkspace, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", legitWorkspace, "commit", "--allow-empty", "-m", "init").Run()

	// Create node_modules with git repo (should be excluded because package.json exists)
	nodeModulesRepo := filepath.Join(tmpDir, "node_modules", "some-git-package")
	os.MkdirAll(nodeModulesRepo, 0755)
	exec.Command("git", "-C", nodeModulesRepo, "init").Run()
	exec.Command("git", "-C", nodeModulesRepo, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", nodeModulesRepo, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", nodeModulesRepo, "commit", "--allow-empty", "-m", "init").Run()

	// Discover workspaces
	discoveries, err := discoverWorkspaces(tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("discoverWorkspaces failed: %v", err)
	}

	// Collect discovered paths
	var foundPaths []string
	for d := range discoveries {
		foundPaths = append(foundPaths, d.relPath)
	}

	// Verify node_modules was excluded
	for _, p := range foundPaths {
		if p == filepath.Join("node_modules", "some-git-package") {
			t.Error("node_modules repo should have been excluded (package.json exists)")
		}
	}
}
