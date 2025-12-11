package auth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	// KeyringService is the service name used for storing credentials in the keyring.
	KeyringService = "atlassian-cli"
)

// TokenSet represents OAuth tokens for an Atlassian host.
type TokenSet struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scopes       []string  `json:"scopes,omitempty"`
}

// IsExpired returns true if the access token has expired.
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
