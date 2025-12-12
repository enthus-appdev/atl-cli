// Package api provides HTTP clients for Atlassian Cloud APIs.
//
// This package implements clients for:
//   - Jira Cloud REST API v3
//   - Confluence Cloud REST API v2
//
// All API calls use OAuth 2.0 Bearer token authentication. Tokens are
// automatically retrieved from the system keyring based on the configured host.
//
// Example usage:
//
//	client, err := api.NewClientFromConfig()
//	if err != nil {
//	    return err
//	}
//	jira := api.NewJiraService(client)
//	issue, err := jira.GetIssue(ctx, "TEST-123")
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/config"
)

const (
	// AtlassianAPIURL is the base URL for Atlassian cloud API requests.
	// All Atlassian Cloud REST APIs are accessed through this gateway.
	AtlassianAPIURL = "https://api.atlassian.com"

	// DefaultTimeout is the default HTTP client timeout for API requests.
	DefaultTimeout = 30 * time.Second
)

// Client is an HTTP client for Atlassian APIs.
type Client struct {
	httpClient *http.Client
	hostname   string
	cloudID    string
	tokens     *auth.TokenSet
	config     *config.Config
}

// ClientOption configures the API client.
type ClientOption func(*Client)

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// NewClient creates a new API client for the given hostname.
func NewClient(hostname string, opts ...ClientOption) (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	tokens, err := auth.GetToken(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}
	if tokens == nil {
		return nil, fmt.Errorf("not authenticated. Run 'atl auth login' first")
	}

	hostConfig := cfg.GetHost(hostname)
	if hostConfig == nil {
		return nil, fmt.Errorf("no configuration found for host %s", hostname)
	}

	client := &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		hostname:   hostname,
		cloudID:    hostConfig.CloudID,
		tokens:     tokens,
		config:     cfg,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// NewClientFromConfig creates a new API client using the current host from config.
func NewClientFromConfig() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.CurrentHost == "" {
		return nil, fmt.Errorf("no host configured. Run 'atl auth login' first")
	}

	return NewClient(cfg.CurrentHost)
}

// Hostname returns the configured hostname.
func (c *Client) Hostname() string {
	return c.hostname
}

// CloudID returns the cloud ID for the host.
func (c *Client) CloudID() string {
	return c.cloudID
}

// BaseURL returns the base URL for Jira API requests.
func (c *Client) JiraBaseURL() string {
	return fmt.Sprintf("%s/ex/jira/%s/rest/api/3", AtlassianAPIURL, c.cloudID)
}

// ConfluenceBaseURL returns the base URL for Confluence API requests.
func (c *Client) ConfluenceBaseURL() string {
	return fmt.Sprintf("%s/ex/confluence/%s/wiki/api/v2", AtlassianAPIURL, c.cloudID)
}

// Request makes an HTTP request to the API.
func (c *Client) Request(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.tokens.AccessToken))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// Get makes a GET request.
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	return c.Request(ctx, http.MethodGet, path, nil, result)
}

// Post makes a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.Request(ctx, http.MethodPost, path, body, result)
}

// Put makes a PUT request.
func (c *Client) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.Request(ctx, http.MethodPut, path, body, result)
}

// Delete makes a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.Request(ctx, http.MethodDelete, path, nil, nil)
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: %s (status %d): %s", e.Status, e.StatusCode, e.Body)
}

// BuildQueryString builds a URL query string from parameters.
func BuildQueryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	values := url.Values{}
	for k, v := range params {
		if v != "" {
			values.Set(k, v)
		}
	}
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}

// JoinPath joins path segments properly.
func JoinPath(base string, paths ...string) string {
	result := strings.TrimSuffix(base, "/")
	for _, p := range paths {
		result = result + "/" + strings.TrimPrefix(p, "/")
	}
	return result
}
