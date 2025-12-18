// Package auth provides OAuth 2.0 authentication for Atlassian Cloud APIs.
//
// This package handles:
//   - OAuth 2.0 authorization code flow with browser-based consent
//   - Secure token storage (file-based with restricted permissions)
//   - Token expiration tracking
//
// Tokens are stored per-host in ~/.config/atlassian/tokens/, allowing users to
// authenticate with multiple Atlassian instances simultaneously.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// KeyringService was the service name used for keyring storage (deprecated).
	// Now tokens are stored in files due to keyring size limitations.
	KeyringService = "atlassian-cli"

	// tokenDirName is the directory name for token storage within the config directory.
	tokenDirName = "tokens"
)

// TokenSet represents OAuth 2.0 tokens for an Atlassian host.
// These tokens are obtained via the OAuth authorization code flow
// and stored securely in the system keyring.
type TokenSet struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scopes       []string  `json:"scopes,omitempty"`
}

// IsExpired returns true if the access token has expired or is about to expire.
// Tokens are considered expired 5 minutes before their actual expiry time
// to provide a buffer for token refresh operations.
func (t *TokenSet) IsExpired() bool {
	// Consider token expired 5 minutes before actual expiry
	return time.Now().Add(5 * time.Minute).After(t.ExpiresAt)
}

// tokenDir returns the directory path for token storage.
// Creates the directory if it doesn't exist with secure permissions (0700).
func tokenDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	dir := filepath.Join(homeDir, ".config", "atlassian", tokenDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create token directory: %w", err)
	}

	return dir, nil
}

// tokenFilePath returns the file path for a hostname's tokens.
// Hostname is sanitized to be filesystem-safe.
func tokenFilePath(hostname string) (string, error) {
	dir, err := tokenDir()
	if err != nil {
		return "", err
	}

	// Sanitize hostname for use as filename
	safeHostname := strings.ReplaceAll(hostname, "/", "_")
	safeHostname = strings.ReplaceAll(safeHostname, "\\", "_")
	safeHostname = strings.ReplaceAll(safeHostname, ":", "_")

	return filepath.Join(dir, safeHostname+".json"), nil
}

// StoreToken stores tokens in a secure file.
// Tokens are stored in ~/.config/atlassian/tokens/<hostname>.json with 0600 permissions.
func StoreToken(hostname string, tokens *TokenSet) error {
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize tokens: %w", err)
	}

	filePath, err := tokenFilePath(hostname)
	if err != nil {
		return err
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// GetToken retrieves tokens from file storage.
// Returns nil, nil if no tokens exist for the hostname.
func GetToken(hostname string) (*TokenSet, error) {
	filePath, err := tokenFilePath(hostname)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var tokens TokenSet
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse stored tokens: %w", err)
	}

	return &tokens, nil
}

// DeleteToken removes tokens from file storage.
// Returns nil if no tokens exist for the hostname.
func DeleteToken(hostname string) error {
	filePath, err := tokenFilePath(hostname)
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete token file: %w", err)
	}
	return nil
}

// ListStoredHosts returns a list of hostnames that have stored tokens.
func ListStoredHosts() ([]string, error) {
	dir, err := tokenDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read token directory: %w", err)
	}

	var hosts []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			// Reverse the sanitization (best effort)
			hostname := strings.TrimSuffix(name, ".json")
			hosts = append(hosts, hostname)
		}
	}

	return hosts, nil
}

// RefreshConfig holds the configuration needed to refresh tokens.
type RefreshConfig struct {
	ClientID     string
	ClientSecret string
}

// RefreshAccessToken refreshes the access token for a given hostname using its stored refresh token.
// It retrieves the current tokens, exchanges the refresh token for new tokens, and stores the result.
// Returns the new TokenSet or an error if refresh fails.
func RefreshAccessToken(ctx context.Context, hostname string, cfg *RefreshConfig) (*TokenSet, error) {
	// Get current tokens
	tokens, err := GetToken(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to get current tokens: %w", err)
	}
	if tokens == nil {
		return nil, fmt.Errorf("no tokens found for %s", hostname)
	}
	if tokens.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available for %s (re-login required)", hostname)
	}

	// Create OAuth flow for refresh
	oauthConfig := &OAuthConfig{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  "http://localhost:8085/callback", // Not used for refresh
		Scopes:       tokens.Scopes,
	}

	flow, err := NewOAuthFlow(oauthConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth flow: %w", err)
	}

	// Refresh tokens
	newTokens, err := flow.RefreshTokens(ctx, tokens.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh tokens: %w", err)
	}

	// Store new tokens
	if err := StoreToken(hostname, newTokens); err != nil {
		return nil, fmt.Errorf("failed to store refreshed tokens: %w", err)
	}

	return newTokens, nil
}
