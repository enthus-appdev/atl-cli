package api

import (
	"context"
	"testing"
)

func TestResolveMentions_ResolvesNameMentions(t *testing.T) {
	doc := MarkdownToADF("Hey @[John Doe]")

	resolver := func(ctx context.Context, name string) (string, error) {
		if name == "John Doe" {
			return "acc-123", nil
		}
		return "", nil
	}

	err := ResolveMentions(context.Background(), doc, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mention := doc.Content[0].Content[1]
	if mention.Attrs.ID != "acc-123" {
		t.Errorf("expected resolved ID 'acc-123', got %q", mention.Attrs.ID)
	}
	if mention.Attrs.Text != "@John Doe" {
		t.Errorf("expected text preserved as '@John Doe', got %q", mention.Attrs.Text)
	}
}

func TestResolveMentions_SkipsIDMentions(t *testing.T) {
	doc := MarkdownToADF("Hey @[id:already-resolved]")

	called := false
	resolver := func(ctx context.Context, name string) (string, error) {
		called = true
		return "", nil
	}

	err := ResolveMentions(context.Background(), doc, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if called {
		t.Error("resolver should not be called for id-based mentions")
	}

	mention := doc.Content[0].Content[1]
	if mention.Attrs.ID != "already-resolved" {
		t.Errorf("expected ID preserved, got %q", mention.Attrs.ID)
	}
}

func TestResolveMentions_UnresolvedBecomesText(t *testing.T) {
	doc := MarkdownToADF("Hey @[Nobody Special]")

	resolver := func(ctx context.Context, name string) (string, error) {
		return "", nil
	}

	err := ResolveMentions(context.Background(), doc, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	node := doc.Content[0].Content[1]
	if node.Type != "text" {
		t.Errorf("expected unresolved mention to become text, got %q", node.Type)
	}
	if node.Text != "@Nobody Special" {
		t.Errorf("expected '@Nobody Special', got %q", node.Text)
	}
}

func TestResolveMentions_NilDoc(t *testing.T) {
	err := ResolveMentions(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error for nil doc: %v", err)
	}
}
