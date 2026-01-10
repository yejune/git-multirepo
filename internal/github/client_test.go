package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractOrgName(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "https URL",
			url:  "https://github.com/git-workspaces",
			want: "git-workspaces",
		},
		{
			name: "http URL",
			url:  "http://github.com/git-workspaces",
			want: "git-workspaces",
		},
		{
			name: "URL without scheme",
			url:  "github.com/git-workspaces",
			want: "git-workspaces",
		},
		{
			name: "URL with trailing slash",
			url:  "https://github.com/git-workspaces/",
			want: "git-workspaces",
		},
		{
			name: "URL with path segments",
			url:  "https://github.com/git-workspaces/some/path",
			want: "git-workspaces",
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "URL without org",
			url:     "https://github.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractOrgName(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("extractOrgName() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("extractOrgName() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("extractOrgName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		orgURL  string
		wantOrg string
		wantErr bool
	}{
		{
			name:    "valid input",
			token:   "test-token",
			orgURL:  "https://github.com/git-workspaces",
			wantOrg: "git-workspaces",
		},
		{
			name:    "empty token",
			token:   "",
			orgURL:  "https://github.com/git-workspaces",
			wantErr: true,
		},
		{
			name:    "empty org URL",
			token:   "test-token",
			orgURL:  "",
			wantErr: true,
		},
		{
			name:    "invalid org URL",
			token:   "test-token",
			orgURL:  "https://github.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.token, tt.orgURL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("NewClient() unexpected error: %v", err)
				return
			}
			if client.org != tt.wantOrg {
				t.Errorf("NewClient() org = %v, want %v", client.org, tt.wantOrg)
			}
			if client.token != tt.token {
				t.Errorf("NewClient() token = %v, want %v", client.token, tt.token)
			}
		})
	}
}

func TestRepositoryExists(t *testing.T) {
	tests := []struct {
		name       string
		repoName   string
		statusCode int
		want       bool
		wantErr    bool
	}{
		{
			name:       "repository exists",
			repoName:   "test-repo",
			statusCode: http.StatusOK,
			want:       true,
			wantErr:    false,
		},
		{
			name:       "repository not found",
			repoName:   "test-repo",
			statusCode: http.StatusNotFound,
			want:       false,
			wantErr:    false,
		},
		{
			name:       "permission denied",
			repoName:   "test-repo",
			statusCode: http.StatusForbidden,
			want:       false,
			wantErr:    true,
		},
		{
			name:       "empty repo name",
			repoName:   "",
			statusCode: http.StatusOK,
			want:       false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := &Client{
				token:      "test-token",
				org:        "test-org",
				baseURL:    server.URL,
				httpClient: server.Client(),
			}

			got, err := client.RepositoryExists(tt.repoName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("RepositoryExists() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("RepositoryExists() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("RepositoryExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateRepository(t *testing.T) {
	tests := []struct {
		name       string
		repoName   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful creation",
			repoName:   "test-repo",
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "permission denied",
			repoName:   "test-repo",
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:       "validation error",
			repoName:   "test-repo",
			statusCode: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
		{
			name:       "empty repo name",
			repoName:   "",
			statusCode: http.StatusCreated,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify headers
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("Missing or incorrect Authorization header")
				}
				if r.Header.Get("Accept") != "application/vnd.github+json" {
					t.Errorf("Missing or incorrect Accept header")
				}
				if r.Method == "POST" && r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Missing or incorrect Content-Type header")
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusUnprocessableEntity {
					w.Write([]byte(`{"message":"Validation Failed","errors":[{"message":"name already exists"}]}`))
				}
			}))
			defer server.Close()

			client := &Client{
				token:      "test-token",
				org:        "test-org",
				baseURL:    server.URL,
				httpClient: server.Client(),
			}

			err := client.CreateRepository(tt.repoName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateRepository() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("CreateRepository() unexpected error: %v", err)
			}
		})
	}
}

func TestGetRepoURL(t *testing.T) {
	client := &Client{
		org: "git-workspaces",
	}

	tests := []struct {
		name     string
		repoName string
		want     string
	}{
		{
			name:     "standard repo name",
			repoName: "test-repo",
			want:     "https://github.com/git-workspaces/test-repo.git",
		},
		{
			name:     "repo with dashes",
			repoName: "my-awesome-repo",
			want:     "https://github.com/git-workspaces/my-awesome-repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.GetRepoURL(tt.repoName)
			if got != tt.want {
				t.Errorf("GetRepoURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
