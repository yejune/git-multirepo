package cmd

import (
	"os"
	"testing"
)

// TestWorkerCount_Integration verifies worker count determination in realistic scenarios
func TestWorkerCount_Integration(t *testing.T) {
	// Save and restore env var
	originalEnv := os.Getenv("GIT_MULTIREPO_WORKERS")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GIT_MULTIREPO_WORKERS", originalEnv)
		} else {
			os.Unsetenv("GIT_MULTIREPO_WORKERS")
		}
	}()

	t.Run("default behavior - CPU based", func(t *testing.T) {
		os.Unsetenv("GIT_MULTIREPO_WORKERS")
		workers := getOptimalWorkerCount()
		
		// Log for manual verification
		t.Logf("Default worker count: %d (based on runtime.NumCPU())", workers)
		
		// Verify constraints
		if workers < 1 {
			t.Errorf("workers must be >= 1, got %d", workers)
		}
		if workers > 32 {
			t.Errorf("workers must be <= 32, got %d", workers)
		}
	})

	t.Run("user override - custom value", func(t *testing.T) {
		os.Setenv("GIT_MULTIREPO_WORKERS", "10")
		workers := getOptimalWorkerCount()
		
		if workers != 10 {
			t.Errorf("expected 10 workers from env var, got %d", workers)
		}
		t.Logf("Custom worker count: %d", workers)
	})

	t.Run("production scenario - high concurrency", func(t *testing.T) {
		os.Setenv("GIT_MULTIREPO_WORKERS", "24")
		workers := getOptimalWorkerCount()
		
		if workers != 24 {
			t.Errorf("expected 24 workers for high concurrency, got %d", workers)
		}
		t.Logf("High concurrency worker count: %d", workers)
	})

	t.Run("resource constrained - low workers", func(t *testing.T) {
		os.Setenv("GIT_MULTIREPO_WORKERS", "4")
		workers := getOptimalWorkerCount()
		
		if workers != 4 {
			t.Errorf("expected 4 workers for constrained environment, got %d", workers)
		}
		t.Logf("Resource constrained worker count: %d", workers)
	})
}
