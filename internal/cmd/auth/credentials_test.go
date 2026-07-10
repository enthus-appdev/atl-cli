package auth

import (
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/config"
)

func TestResolveClientCredentialsPrecedence(t *testing.T) {
	keyring.MockInit()
	cfg := &config.Config{OAuth: &config.OAuthConfig{ClientID: "cfg-id", ClientSecret: "cfg-secret"}}

	t.Setenv("ATLASSIAN_CLIENT_ID", "")
	t.Setenv("ATLASSIAN_CLIENT_SECRET", "")

	id, secret, src := resolveClientCredentials(cfg)
	if id != "cfg-id" || secret != "cfg-secret" || src != auth.SourceConfig {
		t.Fatalf("config layer: got (%q,%q,%q), want (cfg-id,cfg-secret,%q)", id, secret, src, auth.SourceConfig)
	}

	if err := auth.StoreClientCredentials(auth.ClientCredentials{ClientID: "kc-id", ClientSecret: "kc-secret"}); err != nil {
		t.Fatalf("StoreClientCredentials: %v", err)
	}
	id, secret, src = resolveClientCredentials(cfg)
	if id != "kc-id" || secret != "kc-secret" || src != auth.SourceKeychain {
		t.Fatalf("keychain layer: got (%q,%q,%q), want (kc-id,kc-secret,%q)", id, secret, src, auth.SourceKeychain)
	}

	t.Setenv("ATLASSIAN_CLIENT_ID", "env-id")
	t.Setenv("ATLASSIAN_CLIENT_SECRET", "env-secret")
	id, secret, src = resolveClientCredentials(cfg)
	if id != "env-id" || secret != "env-secret" || src != auth.SourceEnv {
		t.Fatalf("env layer: got (%q,%q,%q), want (env-id,env-secret,%q)", id, secret, src, auth.SourceEnv)
	}
}

func TestRequireClientCredentialsMissing(t *testing.T) {
	keyring.MockInit()
	t.Setenv("ATLASSIAN_CLIENT_ID", "")
	t.Setenv("ATLASSIAN_CLIENT_SECRET", "")

	_, _, _, err := requireClientCredentials(&config.Config{})
	if err == nil {
		t.Fatal("expected errNoCredentials when nothing is configured")
	}
}

func TestRequireClientCredentialsPartial(t *testing.T) {
	keyring.MockInit()
	t.Setenv("ATLASSIAN_CLIENT_ID", "some-id")
	t.Setenv("ATLASSIAN_CLIENT_SECRET", "")

	// A lone ID with no matching secret is an incomplete pair — must not resolve.
	_, _, _, err := requireClientCredentials(&config.Config{})
	if err == nil {
		t.Fatal("expected errNoCredentials when client secret is missing")
	}
}

func TestResolveClientCredentialsIncompleteLayerFallsThrough(t *testing.T) {
	keyring.MockInit()

	// Env has only an ID; the coupled pair must come whole from the config layer,
	// never a mix of env ID and config secret.
	t.Setenv("ATLASSIAN_CLIENT_ID", "env-id")
	t.Setenv("ATLASSIAN_CLIENT_SECRET", "")
	cfg := &config.Config{OAuth: &config.OAuthConfig{ClientID: "cfg-id", ClientSecret: "cfg-secret"}}

	id, secret, src := resolveClientCredentials(cfg)
	if id != "cfg-id" || secret != "cfg-secret" || src != auth.SourceConfig {
		t.Fatalf("incomplete env should fall through: got (%q,%q,%q), want (cfg-id,cfg-secret,%q)", id, secret, src, auth.SourceConfig)
	}

	// Symmetric: env supplies only the secret — still incomplete, still falls through.
	t.Setenv("ATLASSIAN_CLIENT_ID", "")
	t.Setenv("ATLASSIAN_CLIENT_SECRET", "env-secret")
	id, secret, src = resolveClientCredentials(cfg)
	if id != "cfg-id" || secret != "cfg-secret" || src != auth.SourceConfig {
		t.Fatalf("env-only-secret should fall through: got (%q,%q,%q), want (cfg-id,cfg-secret,%q)", id, secret, src, auth.SourceConfig)
	}
}
