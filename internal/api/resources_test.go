package api

import "testing"

func TestDedupeResources(t *testing.T) {
	// Mirrors an observed accessible-resources payload: each site repeated once.
	in := []*AccessibleResource{
		{ID: "sandbox-id", URL: "https://example-sandbox.atlassian.net", Name: "example-sandbox"},
		{ID: "prod-id", URL: "https://example.atlassian.net", Name: "example"},
		{ID: "sandbox-id", URL: "https://example-sandbox.atlassian.net", Name: "example-sandbox"},
		{ID: "prod-id", URL: "https://example.atlassian.net", Name: "example"},
	}

	got := dedupeResources(in)

	if len(got) != 2 {
		t.Fatalf("expected 2 unique sites, got %d", len(got))
	}
	// First-seen order must be preserved.
	if got[0].ID != "sandbox-id" || got[1].ID != "prod-id" {
		t.Fatalf("unexpected order: %q, %q", got[0].ID, got[1].ID)
	}
}

func TestDedupeResourcesNoDuplicates(t *testing.T) {
	in := []*AccessibleResource{
		{ID: "prod-id", URL: "https://example.atlassian.net", Name: "example"},
	}
	if got := dedupeResources(in); len(got) != 1 || got[0].ID != "prod-id" {
		t.Fatalf("expected input returned unchanged, got %+v", got)
	}
}
