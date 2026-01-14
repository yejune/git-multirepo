package cmd

import (
	"os"
	"runtime"
	"testing"
)

// TestGetOptimalWorkerCount tests the worker count calculation logic
func TestGetOptimalWorkerCount(t *testing.T) {
	// Save original env var to restore later
	originalEnv := os.Getenv("GIT_MULTIREPO_WORKERS")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GIT_MULTIREPO_WORKERS", originalEnv)
		} else {
			os.Unsetenv("GIT_MULTIREPO_WORKERS")
		}
	}()

	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			name:     "no env var - defaults to CPU * 2",
			envValue: "",
			expected: func() int {
				workers := runtime.NumCPU() * 2
				if workers < 1 {
					return 1
				}
				if workers > 32 {
					return 32
				}
				return workers
			}(),
		},
		{
			name:     "env var set to 8",
			envValue: "8",
			expected: 8,
		},
		{
			name:     "env var set to 1 (minimum)",
			envValue: "1",
			expected: 1,
		},
		{
			name:     "env var set to 32 (maximum)",
			envValue: "32",
			expected: 32,
		},
		{
			name:     "env var set to 50 (capped to 32)",
			envValue: "50",
			expected: 32,
		},
		{
			name:     "env var invalid - falls back to default",
			envValue: "invalid",
			expected: func() int {
				workers := runtime.NumCPU() * 2
				if workers < 1 {
					return 1
				}
				if workers > 32 {
					return 32
				}
				return workers
			}(),
		},
		{
			name:     "env var negative - falls back to default",
			envValue: "-5",
			expected: func() int {
				workers := runtime.NumCPU() * 2
				if workers < 1 {
					return 1
				}
				if workers > 32 {
					return 32
				}
				return workers
			}(),
		},
		{
			name:     "env var zero - falls back to default",
			envValue: "0",
			expected: func() int {
				workers := runtime.NumCPU() * 2
				if workers < 1 {
					return 1
				}
				if workers > 32 {
					return 32
				}
				return workers
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv("GIT_MULTIREPO_WORKERS", tt.envValue)
			} else {
				os.Unsetenv("GIT_MULTIREPO_WORKERS")
			}

			got := getOptimalWorkerCount()
			if got != tt.expected {
				t.Errorf("getOptimalWorkerCount() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// TestGetOptimalWorkerCount_Boundaries tests edge cases
func TestGetOptimalWorkerCount_Boundaries(t *testing.T) {
	defer os.Unsetenv("GIT_MULTIREPO_WORKERS")

	t.Run("minimum is always 1", func(t *testing.T) {
		// Even if NumCPU() returns an impossibly low value,
		// the implementation guarantees minimum of 1
		workers := getOptimalWorkerCount()
		if workers < 1 {
			t.Errorf("getOptimalWorkerCount() = %d, must be >= 1", workers)
		}
	})

	t.Run("maximum is always 32", func(t *testing.T) {
		os.Setenv("GIT_MULTIREPO_WORKERS", "1000")
		workers := getOptimalWorkerCount()
		if workers > 32 {
			t.Errorf("getOptimalWorkerCount() = %d, must be <= 32", workers)
		}
	})

	t.Run("default is between 1 and 32", func(t *testing.T) {
		os.Unsetenv("GIT_MULTIREPO_WORKERS")
		workers := getOptimalWorkerCount()
		if workers < 1 || workers > 32 {
			t.Errorf("getOptimalWorkerCount() = %d, must be between 1 and 32", workers)
		}
	})
}
