package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/zalando/go-keyring"
)

const (
	// credentialService is the OS keychain service name under which the OAuth
	// app's client credentials are stored. It is intentionally separate from
	// user tokens, which are file-based (see token.go) because token blobs
	// exceed some keychain backends' size limits; client credentials are small
	// and benefit from the encrypted, OS-gated store.
	credentialService = "atlassian-cli-oauth"

	// credentialAccount holds the client ID and secret as one JSON entry, so a
	// write is atomic — the pair can never be half-stored across two keys.
	credentialAccount = "client-credentials"
)

// ClientCredentials holds OAuth 2.0 application credentials (client ID and
// secret), as opposed to per-user access/refresh tokens.
type ClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// CredentialSource identifies which layer supplied the OAuth client credentials.
type CredentialSource string

const (
	SourceEnv      CredentialSource = "environment variables"
	SourceKeychain CredentialSource = "OS keychain"
	SourceConfig   CredentialSource = "config file"
	SourceNone     CredentialSource = ""
)

// ResolveClientCredentials returns the OAuth app credentials to use, taking the
// highest-precedence layer that supplies BOTH halves: environment variables,
// then OS keychain, then the config-file values passed in. client_id and
// client_secret are a coupled pair (same app), so a layer contributes only when
// complete — the two halves are never mixed across layers.
func ResolveClientCredentials(configID, configSecret string) (clientID, clientSecret string, source CredentialSource) {
	if id, secret := os.Getenv("ATLASSIAN_CLIENT_ID"), os.Getenv("ATLASSIAN_CLIENT_SECRET"); id != "" && secret != "" {
		return id, secret, SourceEnv
	}
	if kc, ok := GetClientCredentials(); ok {
		return kc.ClientID, kc.ClientSecret, SourceKeychain
	}
	if configID != "" && configSecret != "" {
		return configID, configSecret, SourceConfig
	}
	return "", "", SourceNone
}

// StoreClientCredentials saves the OAuth app credentials in the OS keychain as a
// single entry. Both fields are required.
func StoreClientCredentials(creds ClientCredentials) error {
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return errors.New("both client ID and client secret are required")
	}
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}
	if err := keyring.Set(credentialService, credentialAccount, string(data)); err != nil {
		return fmt.Errorf("failed to store credentials in keychain: %w", err)
	}
	return nil
}

// GetClientCredentials reads the OAuth app credentials from the OS keychain.
// It returns ok=false when no complete credential pair is available. A missing
// entry and an unavailable backend (e.g. a headless machine with no Secret
// Service) are both reported as ok=false rather than an error, because the
// keychain is one optional layer in the credential resolver — callers fall
// through to other sources when it yields nothing.
func GetClientCredentials() (creds ClientCredentials, ok bool) {
	data, err := keyring.Get(credentialService, credentialAccount)
	if err != nil {
		return ClientCredentials{}, false
	}
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return ClientCredentials{}, false
	}
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return ClientCredentials{}, false
	}
	return creds, true
}

// DeleteClientCredentials removes the OAuth app credentials from the OS
// keychain. A missing entry is not an error.
func DeleteClientCredentials() error {
	if err := keyring.Delete(credentialService, credentialAccount); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("failed to delete credentials from keychain: %w", err)
	}
	return nil
}
