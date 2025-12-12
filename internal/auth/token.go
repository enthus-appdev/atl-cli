// Package auth provides OAuth 2.0 authentication for Atlassian Cloud APIs.
//
// This package handles:
//   - OAuth 2.0 authorization code flow with browser-based consent
//   - Secure token storage using the system keyring
//   - Token expiration tracking
//
// Tokens are stored per-host in the system keyring, allowing users to
// authenticate with multiple Atlassian instances simultaneously.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	// KeyringService is the service name used for storing credentials in the system keyring.
	// All tokens are stored under this service name with the hostname as the key.
	KeyringService = "atlassian-cli"
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

// StoreToken stores tokens in the system keyring.
func StoreToken(hostname string, tokens *TokenSet) error {
	data, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("failed to serialize tokens: %w", err)
	}

	if err := keyring.Set(KeyringService, hostname, string(data)); err != nil {
		return fmt.Errorf("failed to store tokens in keyring: %w", err)
	}

	return nil
}

// GetToken retrieves tokens from the system keyring.
func GetToken(hostname string) (*TokenSet, error) {
	data, err := keyring.Get(KeyringService, hostname)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve tokens from keyring: %w", err)
	}

	var tokens TokenSet
	if err := json.Unmarshal([]byte(data), &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse stored tokens: %w", err)
	}

	return &tokens, nil
}

// DeleteToken removes tokens from the system keyring.
func DeleteToken(hostname string) error {
	if err := keyring.Delete(KeyringService, hostname); err != nil {
		if err == keyring.ErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete tokens from keyring: %w", err)
	}
	return nil
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
