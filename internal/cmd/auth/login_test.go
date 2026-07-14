package auth

import (
	"strings"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

func twoSites() []*api.AccessibleResource {
	return []*api.AccessibleResource{
		{ID: "prod-id", URL: "https://example.atlassian.net", Name: "example"},
		{ID: "sandbox-id", URL: "https://example-sandbox.atlassian.net", Name: "example-sandbox"},
	}
}

func TestSelectResourceNonInteractiveErrors(t *testing.T) {
	ios := iostreams.Test() // IsStdinTTY defaults to false

	_, err := selectResource(ios, twoSites())
	if err == nil {
		t.Fatal("expected an error on non-interactive stdin, got nil")
	}
	// The error must steer the user to the deterministic escape hatch.
	if !strings.Contains(err.Error(), "--hostname") {
		t.Fatalf("error should mention --hostname, got %q", err.Error())
	}
}
