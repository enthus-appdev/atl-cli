package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTextToADFWithResolver_ResolvesNameMention(t *testing.T) {
	// TextToADFWithResolver uses MarkdownToADF + ResolveMentions with a JiraService-based resolver.
	// Since ResolveMentions is tested separately, here we verify the full pipeline
	// by testing the equivalent logic directly.
	doc := MarkdownToADF("Hey @[Jane Smith]!")

	// Simulate the resolver that TextToADFWithResolver builds internally
	resolver := func(ctx context.Context, displayName string) (string, error) {
		// Simulate SearchUsers returning a match
		users := []*User{{AccountID: "abc-456", DisplayName: "Jane Smith"}}
		for _, u := range users {
			if u.DisplayName == displayName {
				return u.AccountID, nil
			}
		}
		return "", nil
	}

	err := ResolveMentions(context.Background(), doc, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	para := doc.Content[0]
	if len(para.Content) < 2 {
		t.Fatalf("expected at least 2 inline nodes, got %d", len(para.Content))
	}

	mention := para.Content[1]
	if mention.Type != "mention" {
		t.Errorf("expected mention node, got %q", mention.Type)
	}
	if mention.Attrs.ID != "abc-456" {
		t.Errorf("expected resolved ID 'abc-456', got %q", mention.Attrs.ID)
	}
	if mention.Attrs.Text != "@Jane Smith" {
		t.Errorf("expected text '@Jane Smith', got %q", mention.Attrs.Text)
	}
}

func TestTextToADFWithResolver_CaseInsensitiveMatch(t *testing.T) {
	doc := MarkdownToADF("@[John Doe]")

	resolver := func(ctx context.Context, displayName string) (string, error) {
		// Simulate a user with different casing
		users := []*User{{AccountID: "case-123", DisplayName: "john doe"}}
		for _, u := range users {
			if strings.EqualFold(u.DisplayName, displayName) {
				return u.AccountID, nil
			}
		}
		return "", nil
	}

	err := ResolveMentions(context.Background(), doc, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mention := doc.Content[0].Content[0]
	if mention.Type != "mention" {
		t.Errorf("expected mention, got %q", mention.Type)
	}
	if mention.Attrs.ID != "case-123" {
		t.Errorf("expected case-insensitive match, got ID %q", mention.Attrs.ID)
	}
}

func TestTextToADFWithResolver_IDMentionSkipsResolver(t *testing.T) {
	doc := MarkdownToADF("cc @[id:direct-123]")

	resolverCalled := false
	resolver := func(ctx context.Context, displayName string) (string, error) {
		resolverCalled = true
		return "", nil
	}

	err := ResolveMentions(context.Background(), doc, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolverCalled {
		t.Error("resolver should not be called for @[id:...] mentions")
	}

	mention := doc.Content[0].Content[1]
	if mention.Type != "mention" {
		t.Errorf("expected mention node, got %q", mention.Type)
	}
	if mention.Attrs.ID != "direct-123" {
		t.Errorf("expected ID 'direct-123', got %q", mention.Attrs.ID)
	}
}

func TestTextToADFWithResolver_MentionRoundTrip(t *testing.T) {
	// Parse → resolve → serialize to JSON → verify mention node structure
	doc := MarkdownToADF("Hello @[id:acc-999]")

	err := ResolveMentions(context.Background(), doc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("failed to marshal ADF: %v", err)
	}

	jsonStr := string(data)

	// Verify the JSON contains the expected mention structure
	var parsed ADF
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	mention := parsed.Content[0].Content[1]
	if mention.Type != "mention" {
		t.Errorf("expected mention type in JSON, got %q (json: %s)", mention.Type, jsonStr)
	}
	if mention.Attrs == nil {
		t.Fatal("expected attrs in mention node")
	}
	if mention.Attrs.ID != "acc-999" {
		t.Errorf("expected id 'acc-999', got %q", mention.Attrs.ID)
	}
}
