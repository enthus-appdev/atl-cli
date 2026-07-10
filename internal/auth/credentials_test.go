package auth

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func TestClientCredentialsRoundTrip(t *testing.T) {
	keyring.MockInit()

	if _, ok := GetClientCredentials(); ok {
		t.Fatal("expected no credentials before storing")
	}

	want := ClientCredentials{ClientID: "id-123", ClientSecret: "secret-xyz"}
	if err := StoreClientCredentials(want); err != nil {
		t.Fatalf("StoreClientCredentials: %v", err)
	}

	got, ok := GetClientCredentials()
	if !ok {
		t.Fatal("expected credentials after storing")
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	if err := DeleteClientCredentials(); err != nil {
		t.Fatalf("DeleteClientCredentials: %v", err)
	}
	if _, ok := GetClientCredentials(); ok {
		t.Fatal("expected no credentials after delete")
	}
}

func TestStoreClientCredentialsRequiresBoth(t *testing.T) {
	keyring.MockInit()

	if err := StoreClientCredentials(ClientCredentials{ClientID: "id"}); err == nil {
		t.Fatal("expected error when secret missing")
	}
	if err := StoreClientCredentials(ClientCredentials{ClientSecret: "secret"}); err == nil {
		t.Fatal("expected error when client ID missing")
	}
}

func TestGetClientCredentialsInvalidJSON(t *testing.T) {
	keyring.MockInit()

	// A corrupted entry must degrade to ok=false so the resolver falls through,
	// never a panic or a half-populated pair.
	if err := keyring.Set(credentialService, credentialAccount, "{not-valid-json"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, ok := GetClientCredentials(); ok {
		t.Fatal("expected ok=false for malformed keychain JSON")
	}
}

func TestGetClientCredentialsEmptyFields(t *testing.T) {
	keyring.MockInit()

	// Well-formed JSON with blank halves is not a usable pair: the resolver
	// relies on ok=false here to skip the keychain layer.
	if err := keyring.Set(credentialService, credentialAccount, `{"client_id":"","client_secret":""}`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, ok := GetClientCredentials(); ok {
		t.Fatal("expected ok=false for empty credential fields")
	}
}

func TestDeleteClientCredentialsMissingIsNoError(t *testing.T) {
	keyring.MockInit()

	if err := DeleteClientCredentials(); err != nil {
		t.Fatalf("deleting absent credentials should be a no-op, got: %v", err)
	}
}
