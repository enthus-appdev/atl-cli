package auth

import (
	"errors"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/zalando/go-keyring"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

func TestRunSetCredentialsFromStdin(t *testing.T) {
	keyring.MockInit()

	ios := iostreams.Test()
	ios.In = strings.NewReader("my-secret-from-stdin\n")

	opts := &SetCredentialsOptions{IO: ios, ClientID: "some-client-id", FromStdin: true}
	if err := runSetCredentials(opts); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	creds, ok := auth.GetClientCredentials()
	if !ok || creds.ClientID != "some-client-id" || creds.ClientSecret != "my-secret-from-stdin" {
		t.Fatalf("credentials not stored correctly: %+v ok=%v", creds, ok)
	}
}

func TestRunSetCredentialsInline(t *testing.T) {
	keyring.MockInit()

	opts := &SetCredentialsOptions{IO: iostreams.Test(), ClientID: "id", ClientSecret: "secret"}
	if err := runSetCredentials(opts); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if creds, ok := auth.GetClientCredentials(); !ok || creds.ClientSecret != "secret" {
		t.Fatalf("credentials not stored: %+v ok=%v", creds, ok)
	}
}

func TestRunSetCredentialsMissingClientID(t *testing.T) {
	keyring.MockInit()

	opts := &SetCredentialsOptions{IO: iostreams.Test(), FromStdin: true}
	err := runSetCredentials(opts)
	if err == nil || !strings.Contains(err.Error(), "client-id") {
		t.Fatalf("expected client-id error, got %v", err)
	}
}

func TestRunSetCredentialsStdinReadError(t *testing.T) {
	keyring.MockInit()

	ios := iostreams.Test()
	ios.In = iotest.ErrReader(errors.New("forced read error"))

	opts := &SetCredentialsOptions{IO: ios, ClientID: "some-id", FromStdin: true}
	err := runSetCredentials(opts)
	if err == nil || !strings.Contains(err.Error(), "failed to read secret from stdin") {
		t.Fatalf("expected stdin read error, got %v", err)
	}
}

func TestRunSetCredentialsStdinConflictsWithSecret(t *testing.T) {
	keyring.MockInit()

	opts := &SetCredentialsOptions{IO: iostreams.Test(), ClientID: "id", ClientSecret: "secret", FromStdin: true}
	if err := runSetCredentials(opts); err == nil {
		t.Fatal("expected error when --from-stdin combined with --client-secret")
	}
}

func TestRunSetCredentialsDelete(t *testing.T) {
	keyring.MockInit()

	if err := auth.StoreClientCredentials(auth.ClientCredentials{ClientID: "id", ClientSecret: "secret"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := runSetCredentials(&SetCredentialsOptions{IO: iostreams.Test(), Delete: true}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := auth.GetClientCredentials(); ok {
		t.Fatal("expected credentials removed after --delete")
	}
}

func TestRunSetCredentialsDeleteConflicts(t *testing.T) {
	keyring.MockInit()

	opts := &SetCredentialsOptions{IO: iostreams.Test(), Delete: true, ClientID: "id"}
	if err := runSetCredentials(opts); err == nil {
		t.Fatal("expected error when --delete combined with storage flags")
	}
}
