package auth

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

func TestRunStatusShowsCredentialSource(t *testing.T) {
	keyring.MockInit()
	t.Setenv("ATLASSIAN_CONFIG_DIR", t.TempDir())
	t.Setenv("ATLASSIAN_CLIENT_ID", "")
	t.Setenv("ATLASSIAN_CLIENT_SECRET", "")

	// Keychain outranks the config file in the resolver, so seeding it fixes the
	// reported source to "OS keychain" independent of any real config on disk.
	if err := auth.StoreClientCredentials(auth.ClientCredentials{ClientID: "id", ClientSecret: "secret"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	ios := iostreams.Test()
	var buf bytes.Buffer
	ios.Out = &buf

	if err := runStatus(&StatusOptions{IO: ios}); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if got := buf.String(); !strings.Contains(got, "OAuth credentials: OS keychain") {
		t.Fatalf("expected keychain source in output, got:\n%s", got)
	}
}

func TestRunStatusJSONOmitsCredentialLine(t *testing.T) {
	keyring.MockInit()
	t.Setenv("ATLASSIAN_CONFIG_DIR", t.TempDir())
	t.Setenv("ATLASSIAN_CLIENT_ID", "")
	t.Setenv("ATLASSIAN_CLIENT_SECRET", "")

	if err := auth.StoreClientCredentials(auth.ClientCredentials{ClientID: "id", ClientSecret: "secret"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	ios := iostreams.Test()
	var buf bytes.Buffer
	ios.Out = &buf

	// JSON output is a stable machine-readable shape; the human-readable source
	// line must never leak into it.
	if err := runStatus(&StatusOptions{IO: ios, JSON: true}); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if strings.Contains(buf.String(), "OAuth credentials:") {
		t.Fatalf("JSON output must not contain the credential-source line, got:\n%s", buf.String())
	}
}

func TestAuthStatusJSONIncludesTokenScopes(t *testing.T) {
	data, err := json.Marshal(AuthStatus{
		Hostname:      "sandbox.atlassian.net",
		Authenticated: true,
		Scopes:        []string{"read:cmdb-object:jira", "read:cmdb-schema:jira"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range []string{"read:cmdb-object:jira", "read:cmdb-schema:jira"} {
		if !strings.Contains(string(data), scope) {
			t.Fatalf("JSON missing scope %q: %s", scope, data)
		}
	}
}
