package cmd

import (
	"bytes"
	"io"
	"os"
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
			name: "11 files (should truncate)",
			files: []string{
				"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt",
				"file6.txt", "file7.txt", "file8.txt", "file9.txt", "file10.txt",
				"file11.txt",
			},
			wantLines: 11, // 10 files + 1 "... (N more)" line
			wantMore:  true,
		},
		{
			name: "15 files (should show 10 + more)",
			files: []string{
				"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt",
				"file6.txt", "file7.txt", "file8.txt", "file9.txt", "file10.txt",
				"file11.txt", "file12.txt", "file13.txt", "file14.txt", "file15.txt",
			},
			wantLines: 11, // 10 files + 1 "... (N more)" line
			wantMore:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printKeepFileList(tt.files)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
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

			// Verify specific content for truncation test
			if tt.wantMore {
				expectedCount := len(tt.files) - 10
				expectedMsg := "more files"
				if !strings.Contains(output, expectedMsg) {
					t.Errorf("expected to find %q in output, but didn't\nOutput:\n%s", expectedMsg, output)
				}
				// Verify the count
				for i := 0; i < 10; i++ {
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
				// Verify count message
				if expectedCount > 0 && !strings.Contains(output, "... (") {
					t.Errorf("expected count message format '... (N more files)' not found")
				}
			}
		})
	}
}

func TestPrintKeepFileListFormatting(t *testing.T) {
	files := []string{".env.local", "config/local.json"}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printKeepFileList(files)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
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
