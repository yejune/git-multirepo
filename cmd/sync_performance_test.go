package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestSync_Performance_100Workspaces tests parallel processing performance with 100 workspaces
// This test measures how different worker pool sizes affect processing time
func TestSync_Performance_100Workspaces(t *testing.T) {
	// Create temporary directory structure
	parent := t.TempDir()

	// Setup parent repo
	exec.Command("git", "-C", parent, "init").Run()
	exec.Command("git", "-C", parent, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", parent, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", parent, "commit", "--allow-empty", "-m", "init").Run()

	// Create 100 workspace directories with .git
	t.Log("Creating 100 test workspaces...")
	for i := 0; i < 100; i++ {
		workspacePath := filepath.Join(parent, fmt.Sprintf("workspace-%03d", i))
		os.MkdirAll(workspacePath, 0755)

		// Initialize git repo in workspace
		exec.Command("git", "-C", workspacePath, "init").Run()
		exec.Command("git", "-C", workspacePath, "config", "user.email", "test@test.com").Run()
		exec.Command("git", "-C", workspacePath, "config", "user.name", "Test").Run()
		exec.Command("git", "-C", workspacePath, "remote", "add", "origin", fmt.Sprintf("https://example.com/repo-%03d.git", i)).Run()
		exec.Command("git", "-C", workspacePath, "commit", "--allow-empty", "-m", "init").Run()

		// Create some modified files (simulating real workspaces)
		configFile := filepath.Join(workspacePath, "config.yml")
		os.WriteFile(configFile, []byte("original"), 0644)
		exec.Command("git", "-C", workspacePath, "add", "config.yml").Run()
		exec.Command("git", "-C", workspacePath, "commit", "-m", "Add config").Run()
		os.WriteFile(configFile, []byte("modified"), 0644) // Modified but not committed
	}

	// Test different worker counts
	workerCounts := []int{1, 2, 4, 8, 16, 32, 64}
	results := make(map[int]time.Duration)

	for _, numWorkers := range workerCounts {
		t.Logf("\n=== Testing with %d workers ===", numWorkers)

		// Phase 1: Discover workspaces
		discoveries, err := discoverWorkspaces(parent, parent)
		if err != nil {
			t.Fatalf("discoverWorkspaces failed: %v", err)
		}

		// Phase 2: Process workspaces in parallel with specified worker count
		ctx := context.Background()
		processingStart := time.Now()

		workspaces, err := processWorkspacesParallelWithWorkers(ctx, discoveries, numWorkers)
		if err != nil {
			t.Fatalf("processWorkspacesParallelWithWorkers failed: %v", err)
		}

		processingTime := time.Since(processingStart)
		results[numWorkers] = processingTime

		// Report results for this worker count
		t.Logf("Workers: %d, Processing time: %v, Avg per workspace: %v",
			numWorkers, processingTime, processingTime/time.Duration(len(workspaces)))

		// Validation
		if len(workspaces) != 100 {
			t.Errorf("Expected 100 workspaces, got %d", len(workspaces))
		}
	}

	// Print summary table
	t.Logf("\n=== Performance Summary ===")
	t.Logf("Workers | Processing Time | Speed vs Sequential")
	t.Logf("--------|-----------------|-------------------")
	baseline := results[1]
	for _, numWorkers := range workerCounts {
		duration := results[numWorkers]
		speedup := float64(baseline) / float64(duration)
		t.Logf("%7d | %15v | %.2fx", numWorkers, duration, speedup)
	}

	// Find fastest configuration
	fastest := workerCounts[0]
	fastestTime := results[fastest]
	for _, numWorkers := range workerCounts {
		if results[numWorkers] < fastestTime {
			fastest = numWorkers
			fastestTime = results[numWorkers]
		}
	}

	t.Logf("\n✓ Fastest configuration: %d workers (%v)", fastest, fastestTime)
}

// BenchmarkWorkspaceProcessing benchmarks the parallel processing with different worker pool sizes
func BenchmarkWorkspaceProcessing(b *testing.B) {
	// Setup test data once
	parent := b.TempDir()

	exec.Command("git", "-C", parent, "init").Run()
	exec.Command("git", "-C", parent, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", parent, "config", "user.name", "Test").Run()
	exec.Command("git", "-C", parent, "commit", "--allow-empty", "-m", "init").Run()

	// Create test workspaces
	for i := 0; i < 50; i++ {
		workspacePath := filepath.Join(parent, fmt.Sprintf("ws-%02d", i))
		os.MkdirAll(workspacePath, 0755)
		exec.Command("git", "-C", workspacePath, "init").Run()
		exec.Command("git", "-C", workspacePath, "config", "user.email", "test@test.com").Run()
		exec.Command("git", "-C", workspacePath, "config", "user.name", "Test").Run()
		exec.Command("git", "-C", workspacePath, "remote", "add", "origin", fmt.Sprintf("https://example.com/repo-%02d.git", i)).Run()
		exec.Command("git", "-C", workspacePath, "commit", "--allow-empty", "-m", "init").Run()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		discoveries, _ := discoverWorkspaces(parent, parent)
		ctx := context.Background()
		processWorkspacesParallel(ctx, discoveries)
	}
}
