package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestPrintKeepFileList(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantLines int
		wantMore  bool
	}{
		{
			name:      "empty list",
			files:     []string{},
			wantLines: 0,
			wantMore:  false,
		},
		{
			name:      "single file",
			files:     []string{".env.local"},
			wantLines: 1,
			wantMore:  false,
		},
		{
			name:      "three files",
			files:     []string{".env.local", "config/local.json", "data/cache.db"},
			wantLines: 3,
			wantMore:  false,
		},
		{
			name: "exactly 10 files",
			files: []string{
				"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt",
				"file6.txt", "file7.txt", "file8.txt", "file9.txt", "file10.txt",
			},
			wantLines: 10,
			wantMore:  false,
		},
		{
			name: "11 files (should show all)",
			files: []string{
				"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt",
				"file6.txt", "file7.txt", "file8.txt", "file9.txt", "file10.txt",
				"file11.txt",
			},
			wantLines: 11, // all 11 files
			wantMore:  false,
		},
		{
			name: "15 files (should show all)",
			files: []string{
				"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt",
				"file6.txt", "file7.txt", "file8.txt", "file9.txt", "file10.txt",
				"file11.txt", "file12.txt", "file13.txt", "file14.txt", "file15.txt",
			},
			wantLines: 15, // all 15 files
			wantMore:  false,
		},
		{
			name: "88 files (should show all)",
			files: func() []string {
				files := make([]string, 88)
				for i := 0; i < 88; i++ {
					files[i] = fmt.Sprintf("file%d.txt", i+1)
				}
				return files
			}(),
			wantLines: 88, // all 88 files
			wantMore:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use bytes.Buffer to capture output
			var buf bytes.Buffer

			printKeepFileList(&buf, tt.files)

			output := buf.String()

			// Count lines (excluding trailing newline)
			lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
			if output == "" {
				lines = []string{}
			}
			gotLines := len(lines)

			if gotLines != tt.wantLines {
				t.Errorf("printKeepFileList() output %d lines, want %d\nOutput:\n%s", gotLines, tt.wantLines, output)
			}

			// Check for "... (N more)" line if expected
			hasMore := strings.Contains(output, "more files")
			if hasMore != tt.wantMore {
				t.Errorf("printKeepFileList() hasMore=%v, want %v\nOutput:\n%s", hasMore, tt.wantMore, output)
			}

			// Verify indentation and bullet points
			if tt.wantLines > 0 {
				for _, line := range lines {
					if !strings.Contains(line, "more files") {
						if !strings.HasPrefix(line, "      •") {
							t.Errorf("line should start with '      •', got: %q", line)
						}
					}
				}
			}

			// Verify specific content
			if tt.wantMore {
				// Test case for truncation (if we add it back later)
				expectedMsg := "more files"
				if !strings.Contains(output, expectedMsg) {
					t.Errorf("expected to find %q in output, but didn't\nOutput:\n%s", expectedMsg, output)
				}
				// Verify first 10 files are shown
				for i := 0; i < 10 && i < len(tt.files); i++ {
					if !strings.Contains(output, tt.files[i]) {
						t.Errorf("expected to find %q in output (first 10 files), but didn't", tt.files[i])
					}
				}
				// Verify files beyond 10 are NOT shown
				for i := 10; i < len(tt.files); i++ {
					if strings.Contains(output, tt.files[i]) {
						t.Errorf("file %q should not appear in truncated output", tt.files[i])
					}
				}
			} else if len(tt.files) > 10 {
				// Verbose mode: verify ALL files are shown
				for i, file := range tt.files {
					if !strings.Contains(output, file) {
						t.Errorf("expected to find file %q (index %d) in output, but didn't\nOutput:\n%s", file, i, output)
					}
				}
				// Verify NO "more files" message
				if strings.Contains(output, "more files") {
					t.Errorf("should NOT contain 'more files' message in verbose mode\nOutput:\n%s", output)
				}
			}
		})
	}
}

func TestPrintKeepFileListFormatting(t *testing.T) {
	files := []string{".env.local", "config/local.json"}

	// Use bytes.Buffer to capture output
	var buf bytes.Buffer

	printKeepFileList(&buf, files)

	output := buf.String()

	// Verify exact formatting
	expectedLines := []string{
		"      • .env.local",
		"      • config/local.json",
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for i, expected := range expectedLines {
		if i >= len(lines) {
			t.Errorf("missing line %d: want %q", i, expected)
			continue
		}
		if lines[i] != expected {
			t.Errorf("line %d:\ngot:  %q\nwant: %q", i, lines[i], expected)
		}
	}
}
