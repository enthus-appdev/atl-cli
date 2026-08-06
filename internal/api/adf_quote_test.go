package api

import "testing"

func TestQuoteADF_PreservesMediaSingle(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type: "mediaSingle",
			Content: []ADFContent{{
				Type: "media",
				Attrs: &ADFAttrs{
					ID:         "media-abc",
					Type:       "file",
					Collection: "coll-1",
				},
			}},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if quote.Type != "blockquote" {
		t.Fatalf("expected blockquote, got %q", quote.Type)
	}
	if len(quote.Content) != 1 {
		t.Fatalf("expected 1 child, got %d", len(quote.Content))
	}
	child := quote.Content[0]
	if child.Type != "mediaSingle" {
		t.Fatalf("expected mediaSingle preserved, got %q", child.Type)
	}
	media := child.Content[0]
	if media.Type != "media" {
		t.Fatalf("expected media node, got %q", media.Type)
	}
	if media.Attrs == nil || media.Attrs.ID != "media-abc" || media.Attrs.Collection != "coll-1" {
		t.Errorf("media attrs lost: %+v", media.Attrs)
	}
}

func TestQuoteADF_PreservesMentionInsideParagraph(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type: "paragraph",
			Content: []ADFContent{{
				Type:  "mention",
				Attrs: &ADFAttrs{ID: "acc-123", Text: "@Bernd Waldmann"},
			}},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	mention := quote.Content[0].Content[0]
	if mention.Type != "mention" {
		t.Fatalf("expected mention preserved, got %q", mention.Type)
	}
	if mention.Attrs.ID != "acc-123" {
		t.Errorf("expected account id acc-123, got %q", mention.Attrs.ID)
	}
}

func TestQuoteADF_StripsParagraphNodeMarks(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type:    "paragraph",
			Marks:   []ADFMark{{Type: "em"}},
			Content: []ADFContent{{Type: "text", Text: "hi"}},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if quote.Content[0].Marks != nil {
		t.Errorf("expected node-level marks cleared, got %+v", quote.Content[0].Marks)
	}
	if quote.Content[0].Content[0].Text != "hi" {
		t.Error("text content must survive mark stripping")
	}
}

func TestQuoteADF_DegradesIllegalChildren(t *testing.T) {
	for _, tc := range []struct {
		name string
		node ADFContent
	}{
		{"heading", ADFContent{
			Type:    "heading",
			Attrs:   &ADFAttrs{Level: 2},
			Content: []ADFContent{{Type: "text", Text: "Section title"}},
		}},
		{"nested blockquote", ADFContent{
			Type: "blockquote",
			Content: []ADFContent{{
				Type:    "paragraph",
				Content: []ADFContent{{Type: "text", Text: "Section title"}},
			}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			quote := QuoteADF(&ADF{Type: "doc", Version: 1, Content: []ADFContent{tc.node}})
			if quote == nil {
				t.Fatal("expected a blockquote node")
			}
			child := quote.Content[0]
			if child.Type != "paragraph" {
				t.Fatalf("expected degrade to paragraph, got %q", child.Type)
			}
			if len(child.Content) != 1 || child.Content[0].Type != "text" {
				t.Fatalf("expected a single text child, got %+v", child.Content)
			}
			if child.Content[0].Text == "" {
				t.Error("degraded paragraph must keep the text")
			}
		})
	}
}

func TestQuoteADF_NilWhenNothingQuotable(t *testing.T) {
	if got := QuoteADF(nil); got != nil {
		t.Errorf("expected nil for nil input, got %+v", got)
	}
	if got := QuoteADF(&ADF{Type: "doc", Version: 1}); got != nil {
		t.Errorf("expected nil for empty document, got %+v", got)
	}
}

func TestAttributionParagraph(t *testing.T) {
	p := AttributionParagraph("Bernd Waldmann")
	if p.Type != "paragraph" {
		t.Fatalf("expected paragraph, got %q", p.Type)
	}
	text := p.Content[0]
	if text.Text != "Replying to Bernd Waldmann:" {
		t.Errorf("unexpected attribution text: %q", text.Text)
	}
	if len(text.Marks) != 1 || text.Marks[0].Type != "em" {
		t.Errorf("expected em mark, got %+v", text.Marks)
	}
}
