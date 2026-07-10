package auth

import (
	"errors"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/config"
)

// errNoCredentials is returned when no layer supplies a complete credential
// pair. The message lists the resolution order so a user knows every option.
var errNoCredentials = errors.New(`OAuth credentials not configured

Provide them in one of these ways (checked in this order):
  1. Environment: ATLASSIAN_CLIENT_ID and ATLASSIAN_CLIENT_SECRET
  2. OS keychain:  atl auth set-credentials --client-id <id> --client-secret <secret>
  3. Config file:  atl auth setup`)

// resolveClientCredentials resolves the OAuth app credentials for the given
// config, delegating layer precedence to auth.ResolveClientCredentials.
func resolveClientCredentials(cfg *config.Config) (clientID, clientSecret string, source auth.CredentialSource) {
	var configID, configSecret string
	if cfg != nil && cfg.OAuth != nil {
		configID, configSecret = cfg.OAuth.ClientID, cfg.OAuth.ClientSecret
	}
	return auth.ResolveClientCredentials(configID, configSecret)
}

// requireClientCredentials resolves credentials and returns errNoCredentials
// when either half is missing.
func requireClientCredentials(cfg *config.Config) (clientID, clientSecret string, source auth.CredentialSource, err error) {
	clientID, clientSecret, source = resolveClientCredentials(cfg)
	if clientID == "" || clientSecret == "" {
		return "", "", auth.SourceNone, errNoCredentials
	}
	return clientID, clientSecret, source, nil
}
