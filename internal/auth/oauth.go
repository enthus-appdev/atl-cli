package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// AtlassianAuthURL is the authorization endpoint for Atlassian OAuth.
	AtlassianAuthURL = "https://auth.atlassian.com/authorize"
	// AtlassianTokenURL is the token endpoint for Atlassian OAuth.
	AtlassianTokenURL = "https://auth.atlassian.com/oauth/token"
	// AtlassianAPIURL is the base URL for Atlassian API requests.
	AtlassianAPIURL = "https://api.atlassian.com"
)

// DefaultScopes returns the default OAuth scopes for API version 1 (classic scopes).
// Classic scopes provide broader access and are recommended by Atlassian
// when multiple permissions are needed.
func DefaultScopes() []string {
	return ScopesV1()
}

// ScopesV1 returns classic OAuth scopes for v1 APIs.
// These work with Confluence REST API v1 and Jira REST API v3.
func ScopesV1() []string {
	return []string{
		// Jira scopes (classic)
		"read:jira-work",
		"write:jira-work",
		"read:jira-user",
		// Confluence scopes (classic) - used with v1 API
		"read:confluence-content.all",
		"write:confluence-content",
		"read:confluence-space.summary",
		// Token refresh
		"offline_access",
	}
}

// ScopesV2 returns granular OAuth scopes for v2 APIs.
// These work with Confluence REST API v2 and Jira REST API v3.
func ScopesV2() []string {
	return []string{
		// Jira scopes (classic - same for both versions)
		"read:jira-work",
		"write:jira-work",
		"read:jira-user",
		// Confluence scopes (granular) - required for v2 API
		"read:space:confluence",
		"read:page:confluence",
		"write:page:confluence",
		"read:content:confluence",
		"write:content:confluence",
		// Token refresh
		"offline_access",
	}
}

// OAuthConfig holds OAuth configuration.
type OAuthConfig struct {
	ClientID    string
	ClientSecret string
	RedirectURI string
	Scopes      []string
}

// OAuthFlow manages the OAuth 2.0 authorization code flow.
type OAuthFlow struct {
	config     *OAuthConfig
	state      string
	httpClient *http.Client
}

// NewOAuthFlow creates a new OAuth flow.
func NewOAuthFlow(config *OAuthConfig) (*OAuthFlow, error) {
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	return &OAuthFlow{
		config:     config,
		state:      state,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// AuthorizationURL returns the URL to redirect the user to for authorization.
func (f *OAuthFlow) AuthorizationURL() string {
	params := url.Values{
		"client_id":     {f.config.ClientID},
		"redirect_uri":  {f.config.RedirectURI},
		"response_type": {"code"},
		"scope":         {strings.Join(f.config.Scopes, " ")},
		"state":         {f.state},
		"audience":      {"api.atlassian.com"},
		"prompt":        {"consent"},
	}

	return fmt.Sprintf("%s?%s", AtlassianAuthURL, params.Encode())
}

// ExchangeCode exchanges an authorization code for tokens.
func (f *OAuthFlow) ExchangeCode(ctx context.Context, code string) (*TokenSet, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {f.config.ClientID},
		"client_secret": {f.config.ClientSecret},
		"code":          {code},
		"redirect_uri":  {f.config.RedirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, AtlassianTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	tokens := &TokenSet{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		Scopes:       strings.Split(tokenResp.Scope, " "),
	}

	return tokens, nil
}

// RefreshTokens exchanges a refresh token for new tokens.
func (f *OAuthFlow) RefreshTokens(ctx context.Context, refreshToken string) (*TokenSet, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {f.config.ClientID},
		"client_secret": {f.config.ClientSecret},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, AtlassianTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh tokens: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	tokens := &TokenSet{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		Scopes:       strings.Split(tokenResp.Scope, " "),
	}

	return tokens, nil
}

// State returns the state parameter used in the authorization request.
func (f *OAuthFlow) State() string {
	return f.state
}

// DefaultCallbackPort is the port used for the OAuth callback server.
const DefaultCallbackPort = 8085

// StartCallbackServer starts a local HTTP server to receive the OAuth callback.
// It listens on the default callback port (8085) which must match the OAuth app configuration.
// Returns the server, the port it's listening on, and any error.
func StartCallbackServer(codeChan chan<- string, errChan chan<- error, expectedState string) (*http.Server, int, error) {
	port := DefaultCallbackPort
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to start callback server on port %d: %w", port, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			errChan <- fmt.Errorf("state mismatch: expected %s, got %s", expectedState, state)
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			errChan <- fmt.Errorf("authorization error: %s - %s", errParam, errDesc)
			http.Error(w, errDesc, http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			http.Error(w, "No authorization code", http.StatusBadRequest)
			return
		}

		// Send success response to browser
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
               display: flex; justify-content: center; align-items: center; height: 100vh;
               margin: 0; background: #f5f5f5; }
        .container { text-align: center; padding: 40px; background: white;
                     border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #0052CC; margin-bottom: 16px; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Authentication Successful!</h1>
        <p>You can close this window and return to the terminal.</p>
    </div>
</body>
</html>
`))

		codeChan <- code
	})

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	return server, port, nil
}

// generateState generates a cryptographically random state parameter.
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
