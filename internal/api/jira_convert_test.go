package api

import (
	"strings"
	"testing"
)

func TestConvertAttrs_MentionAttributes(t *testing.T) {
	attrs := &ADFAttrs{
		ID:          "acc-123",
		Text:        "@John Doe",
		AccessLevel: "",
	}

	result := convertAttrs(attrs)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result["id"] != "acc-123" {
		t.Errorf("expected id 'acc-123', got %v", result["id"])
	}
	if result["text"] != "@John Doe" {
		t.Errorf("expected text '@John Doe', got %v", result["text"])
	}
	// Empty accessLevel should not be included
	if _, ok := result["accessLevel"]; ok {
		t.Error("empty accessLevel should not be in result")
	}
}

func TestConvertAttrs_MentionWithAccessLevel(t *testing.T) {
	attrs := &ADFAttrs{
		ID:          "acc-456",
		Text:        "@Jane",
		AccessLevel: "CONTAINER",
	}

	result := convertAttrs(attrs)
	if result["accessLevel"] != "CONTAINER" {
		t.Errorf("expected accessLevel 'CONTAINER', got %v", result["accessLevel"])
	}
}

func TestConvertAttrs_Nil(t *testing.T) {
	result := convertAttrs(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestConvertAttrs_EmptyAttrs(t *testing.T) {
	result := convertAttrs(&ADFAttrs{})
	if result != nil {
		t.Errorf("expected nil for empty attrs, got %v", result)
	}
}

func TestConvertNode_MentionToText(t *testing.T) {
	mention := ADFContent{
		Type: "mention",
		Attrs: &ADFAttrs{
			ID:   "acc-123",
			Text: "@John Doe",
		},
	}

	node := convertNode(mention)
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.Text != "@John Doe" {
		t.Errorf("expected '@John Doe', got %q", node.Text)
	}
}

func TestConvertNode_MentionWithoutText(t *testing.T) {
	mention := ADFContent{
		Type:  "mention",
		Attrs: &ADFAttrs{ID: "acc-123"},
	}

	node := convertNode(mention)
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.Text != "@Unknown" {
		t.Errorf("expected '@Unknown' fallback, got %q", node.Text)
	}
}

func TestADFToText_WithMention(t *testing.T) {
	doc := &ADF{
		Version: 1,
		Type:    "doc",
		Content: []ADFContent{
			{
				Type: "paragraph",
				Content: []ADFContent{
					{Type: "text", Text: "Hello "},
					{
						Type: "mention",
						Attrs: &ADFAttrs{
							ID:   "acc-123",
							Text: "@John Doe",
						},
					},
				},
			},
		},
	}

	result := ADFToText(doc)
	// The library concatenates adjacent text nodes without spacing
	if result != "Hello @John Doe" && result != "Hello@John Doe" {
		t.Errorf("expected mention text in output, got %q", result)
	}
	if !strings.Contains(result, "@John Doe") {
		t.Errorf("expected '@John Doe' in output, got %q", result)
	}
}
