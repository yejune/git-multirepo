package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	githubAPIBase = "https://api.github.com"
	timeout       = 30 * time.Second
)

// Client represents a GitHub API client
type Client struct {
	token      string
	org        string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a GitHub API client from a token and organization URL
// Accepts URLs in formats: "https://github.com/org", "http://github.com/org", "github.com/org"
func NewClient(token, orgURL string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}
	if orgURL == "" {
		return nil, fmt.Errorf("organization URL cannot be empty")
	}

	org, err := extractOrgName(orgURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		token:   token,
		org:     org,
		baseURL: githubAPIBase,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// extractOrgName extracts organization name from various URL formats
func extractOrgName(orgURL string) (string, error) {
	// Remove trailing slashes
	orgURL = strings.TrimRight(orgURL, "/")

	// Add scheme if missing for proper parsing
	if !strings.HasPrefix(orgURL, "http://") && !strings.HasPrefix(orgURL, "https://") {
		orgURL = "https://" + orgURL
	}

	parsed, err := url.Parse(orgURL)
	if err != nil {
		return "", fmt.Errorf("invalid organization URL: %w", err)
	}

	// Extract path and remove leading slash
	path := strings.TrimPrefix(parsed.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("no organization name found in URL: %s", orgURL)
	}

	return parts[0], nil
}

// RepositoryExists checks if a repository exists in the organization
func (c *Client) RepositoryExists(repoName string) (bool, error) {
	if repoName == "" {
		return false, fmt.Errorf("repository name cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, c.org, repoName)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusForbidden:
		return false, fmt.Errorf("permission denied. Check organization membership and token scopes")
	default:
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}
}

// CreateRepository creates a private repository in the organization
func (c *Client) CreateRepository(repoName string) error {
	if repoName == "" {
		return fmt.Errorf("repository name cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/orgs/%s/repos", c.baseURL, c.org)

	payload := map[string]interface{}{
		"name":    repoName,
		"private": true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusForbidden:
		return fmt.Errorf("permission denied. Check organization membership and token scopes")
	case http.StatusUnprocessableEntity:
		// Parse GitHub error message
		var errResp struct {
			Message string `json:"message"`
			Errors  []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			if len(errResp.Errors) > 0 {
				return fmt.Errorf("%s: %s", errResp.Message, errResp.Errors[0].Message)
			}
			return fmt.Errorf("%s", errResp.Message)
		}
		return fmt.Errorf("validation error: %s", string(body))
	default:
		return fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}
}

// GetRepoURL returns the Git HTTPS URL for pushing
func (c *Client) GetRepoURL(repoName string) string {
	return fmt.Sprintf("https://github.com/%s/%s.git", c.org, repoName)
}

// setHeaders sets required headers for GitHub API requests
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")
	}
}
