package cmd

import "testing"

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL with .git",
			url:      "https://github.com/user/repo.git",
			expected: "repo",
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/user/repo",
			expected: "repo",
		},
		{
			name:     "SSH URL with .git",
			url:      "git@github.com:user/repo.git",
			expected: "repo",
		},
		{
			name:     "SSH URL without .git",
			url:      "git@github.com:user/repo",
			expected: "repo",
		},
		{
			name:     "GitLab HTTPS",
			url:      "https://gitlab.com/group/subgroup/project.git",
			expected: "project",
		},
		{
			name:     "GitLab SSH",
			url:      "git@gitlab.com:group/subgroup/project.git",
			expected: "project",
		},
		{
			name:     "Simple name",
			url:      "myrepo",
			expected: "myrepo",
		},
		{
			name:     "Local path",
			url:      "/path/to/repo",
			expected: "repo",
		},
		{
			name:     "Bitbucket SSH",
			url:      "git@bitbucket.org:team/repo.git",
			expected: "repo",
		},
		{
			name:     "SSH with nested path",
			url:      "git@github.com:org/team/repo.git",
			expected: "repo",
		},
		{
			name:     "SSH colon only",
			url:      "host:repo",
			expected: "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRepoName(tt.url)
			if result != tt.expected {
				t.Errorf("extractRepoName(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}
