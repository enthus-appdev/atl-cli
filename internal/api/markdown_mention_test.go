package api

import (
	"testing"
)

func TestMarkdownToADF_MentionByID(t *testing.T) {
	adf := MarkdownToADF("Hello @[id:5f1234567890abcdef1234567] world")

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	para := adf.Content[0]
	if para.Type != "paragraph" {
		t.Errorf("expected paragraph, got %q", para.Type)
	}

	// Should have: text("Hello "), mention, text(" world")
	if len(para.Content) != 3 {
		t.Fatalf("expected 3 inline nodes, got %d", len(para.Content))
	}

	// Verify the mention node
	mention := para.Content[1]
	if mention.Type != "mention" {
		t.Errorf("expected mention node, got %q", mention.Type)
	}
	if mention.Attrs == nil {
		t.Fatal("expected mention to have attrs")
	}
	if mention.Attrs.ID != "5f1234567890abcdef1234567" {
		t.Errorf("expected ID '5f1234567890abcdef1234567', got %q", mention.Attrs.ID)
	}
	if mention.Attrs.Text != "@5f1234567890abcdef1234567" {
		t.Errorf("expected Text '@5f1234567890abcdef1234567', got %q", mention.Attrs.Text)
	}
	if mention.Attrs.AccessLevel != "" {
		t.Errorf("expected empty AccessLevel, got %q", mention.Attrs.AccessLevel)
	}

	// Verify surrounding text
	if para.Content[0].Text != "Hello " {
		t.Errorf("expected 'Hello ', got %q", para.Content[0].Text)
	}
	if para.Content[2].Text != " world" {
		t.Errorf("expected ' world', got %q", para.Content[2].Text)
	}
}

func TestMarkdownToADF_MentionByName(t *testing.T) {
	adf := MarkdownToADF("cc @[John Doe] for review")

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	para := adf.Content[0]

	// Should have: text("cc "), mention, text(" for review")
	if len(para.Content) != 3 {
		t.Fatalf("expected 3 inline nodes, got %d", len(para.Content))
	}

	mention := para.Content[1]
	if mention.Type != "mention" {
		t.Errorf("expected mention node, got %q", mention.Type)
	}
	if mention.Attrs == nil {
		t.Fatal("expected mention to have attrs")
	}
	// Name mentions have no ID (will be resolved later)
	if mention.Attrs.ID != "" {
		t.Errorf("expected empty ID for name mention, got %q", mention.Attrs.ID)
	}
	if mention.Attrs.Text != "@John Doe" {
		t.Errorf("expected Text '@John Doe', got %q", mention.Attrs.Text)
	}
	if mention.Attrs.AccessLevel != "" {
		t.Errorf("expected empty AccessLevel, got %q", mention.Attrs.AccessLevel)
	}
}

func TestMarkdownToADF_MultipleMentions(t *testing.T) {
	adf := MarkdownToADF("@[id:abc123] and @[Jane Smith] please review")

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	para := adf.Content[0]

	// Should have: mention, text(" and "), mention, text(" please review")
	if len(para.Content) != 4 {
		t.Fatalf("expected 4 inline nodes, got %d", len(para.Content))
	}

	// First mention - by ID
	m1 := para.Content[0]
	if m1.Type != "mention" {
		t.Errorf("expected first node to be mention, got %q", m1.Type)
	}
	if m1.Attrs == nil {
		t.Fatal("expected first mention to have attrs")
	}
	if m1.Attrs.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %q", m1.Attrs.ID)
	}

	// Text between mentions
	if para.Content[1].Text != " and " {
		t.Errorf("expected ' and ', got %q", para.Content[1].Text)
	}

	// Second mention - by name
	m2 := para.Content[2]
	if m2.Type != "mention" {
		t.Errorf("expected third node to be mention, got %q", m2.Type)
	}
	if m2.Attrs == nil {
		t.Fatal("expected second mention to have attrs")
	}
	if m2.Attrs.ID != "" {
		t.Errorf("expected empty ID for name mention, got %q", m2.Attrs.ID)
	}
	if m2.Attrs.Text != "@Jane Smith" {
		t.Errorf("expected Text '@Jane Smith', got %q", m2.Attrs.Text)
	}

	// Trailing text
	if para.Content[3].Text != " please review" {
		t.Errorf("expected ' please review', got %q", para.Content[3].Text)
	}
}

func TestMarkdownToADF_MentionNotTriggeredByPlainAt(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"email address", "user@example.com"},
		{"at sign alone", "@ something"},
		{"at sign in text", "reach me at user@domain.com please"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adf := MarkdownToADF(tt.input)

			if len(adf.Content) != 1 {
				t.Fatalf("expected 1 content block, got %d", len(adf.Content))
			}

			para := adf.Content[0]
			for _, node := range para.Content {
				if node.Type == "mention" {
					t.Errorf("plain @ should not create mention node, but found one in %q", tt.input)
				}
			}
		})
	}
}
