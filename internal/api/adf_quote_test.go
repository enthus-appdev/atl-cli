package api

import (
	"strings"
	"testing"
)

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

func TestQuoteADF_PreservesBulletList(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type: "bulletList",
			Content: []ADFContent{{
				Type:    "listItem",
				Content: []ADFContent{{Type: "paragraph", Content: []ADFContent{{Type: "text", Text: "item"}}}},
			}},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if len(quote.Content) != 1 || quote.Content[0].Type != "bulletList" {
		t.Fatalf("expected bulletList preserved, got %q", quote.Content[0].Type)
	}
}

func TestQuoteADF_PreservesOrderedList(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type: "orderedList",
			Content: []ADFContent{{
				Type:    "listItem",
				Content: []ADFContent{{Type: "paragraph", Content: []ADFContent{{Type: "text", Text: "item"}}}},
			}},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if len(quote.Content) != 1 || quote.Content[0].Type != "orderedList" {
		t.Fatalf("expected orderedList preserved, got %q", quote.Content[0].Type)
	}
}

func TestQuoteADF_PreservesCodeBlock(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type:    "codeBlock",
			Attrs:   &ADFAttrs{Language: "go"},
			Content: []ADFContent{{Type: "text", Text: "func main() {}"}},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if len(quote.Content) != 1 || quote.Content[0].Type != "codeBlock" {
		t.Fatalf("expected codeBlock preserved, got %q", quote.Content[0].Type)
	}
}

func TestQuoteADF_PreservesMediaGroup(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type: "mediaGroup",
			Content: []ADFContent{{
				Type: "media",
				Attrs: &ADFAttrs{
					ID:         "media-xyz",
					Type:       "file",
					Collection: "coll-2",
				},
			}},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if len(quote.Content) != 1 || quote.Content[0].Type != "mediaGroup" {
		t.Fatalf("expected mediaGroup preserved, got %q", quote.Content[0].Type)
	}
}

func TestQuoteADF_PreservesInlineMarksInParagraph(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type:  "paragraph",
			Marks: []ADFMark{{Type: "em"}}, // Node-level mark (should be cleared)
			Content: []ADFContent{{
				Type:  "text",
				Text:  "bold text",
				Marks: []ADFMark{{Type: "strong"}}, // Inline mark (should survive)
			}},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	para := quote.Content[0]
	if para.Marks != nil {
		t.Errorf("expected node-level marks cleared, got %+v", para.Marks)
	}
	textNode := para.Content[0]
	if len(textNode.Marks) != 1 || textNode.Marks[0].Type != "strong" {
		t.Errorf("expected inline strong mark to survive, got %+v", textNode.Marks)
	}
	if textNode.Text != "bold text" {
		t.Errorf("expected text to survive, got %q", textNode.Text)
	}
}

func TestQuoteADF_HoistsMediaFromIllegalPanel(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type:  "panel",
			Attrs: &ADFAttrs{PanelType: "info"},
			Content: []ADFContent{
				{
					Type:    "paragraph",
					Content: []ADFContent{{Type: "text", Text: "info text"}},
				},
				{
					Type: "mediaSingle",
					Content: []ADFContent{{
						Type: "media",
						Attrs: &ADFAttrs{
							ID:         "media-panel",
							Type:       "file",
							Collection: "panel-coll",
						},
					}},
				},
			},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if len(quote.Content) != 2 {
		t.Fatalf("expected 2 hoisted children, got %d", len(quote.Content))
	}
	if quote.Content[0].Type != "paragraph" {
		t.Fatalf("expected first child to be paragraph, got %q", quote.Content[0].Type)
	}
	if quote.Content[1].Type != "mediaSingle" {
		t.Fatalf("expected second child to be mediaSingle, got %q", quote.Content[1].Type)
	}
	media := quote.Content[1].Content[0]
	if media.Attrs == nil || media.Attrs.ID != "media-panel" || media.Attrs.Collection != "panel-coll" {
		t.Errorf("media attrs lost during hoisting: %+v", media.Attrs)
	}
}

func TestQuoteADF_DegradesIllegalWithoutLegalDescendants(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type:    "panel",
			Attrs:   &ADFAttrs{PanelType: "warning"},
			Content: []ADFContent{{Type: "text", Text: "warning text"}},
		}},
	}

	quote := QuoteADF(original)
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
}

func TestQuoteADF_HeadingInlineRunsStayTogether(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type:  "heading",
			Attrs: &ADFAttrs{Level: 2},
			Content: []ADFContent{
				{Type: "text", Text: "Hello "},
				{Type: "mention", Attrs: &ADFAttrs{ID: "acc-123", Text: "@Bob"}},
				{Type: "text", Text: " World"},
			},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if len(quote.Content) != 1 {
		t.Fatalf("expected 1 child, got %d; heading should not fragment", len(quote.Content))
	}
	child := quote.Content[0]
	if child.Type != "paragraph" {
		t.Fatalf("expected paragraph, got %q", child.Type)
	}
	if len(child.Content) != 1 || child.Content[0].Type != "text" {
		t.Fatalf("expected a single text child, got %+v", child.Content)
	}
	text := child.Content[0].Text
	if !strings.Contains(text, "Hello") || !strings.Contains(text, "Bob") || !strings.Contains(text, "World") {
		t.Errorf("expected all inline runs in one paragraph, got %q", text)
	}
}

func TestQuoteADF_PanelWithMixedContentHoistsAll(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type:  "panel",
			Attrs: &ADFAttrs{PanelType: "info"},
			Content: []ADFContent{
				{
					Type:    "paragraph",
					Content: []ADFContent{{Type: "text", Text: "Info text"}},
				},
				{
					Type: "mediaSingle",
					Content: []ADFContent{{
						Type: "media",
						Attrs: &ADFAttrs{
							ID:         "media-info",
							Type:       "file",
							Collection: "coll-info",
						},
					}},
				},
			},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if len(quote.Content) != 2 {
		t.Fatalf("expected 2 hoisted children, got %d", len(quote.Content))
	}
	if quote.Content[0].Type != "paragraph" || quote.Content[0].Content[0].Text != "Info text" {
		t.Fatalf("expected paragraph with 'Info text' as first child")
	}
	if quote.Content[1].Type != "mediaSingle" {
		t.Fatalf("expected mediaSingle as second child, got %q", quote.Content[1].Type)
	}
}

func TestQuoteADF_PanelWithParagraphAndHeadingFlattenHeading(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type:  "panel",
			Attrs: &ADFAttrs{PanelType: "info"},
			Content: []ADFContent{
				{
					Type:    "paragraph",
					Content: []ADFContent{{Type: "text", Text: "Paragraph text"}},
				},
				{
					Type:    "heading",
					Attrs:   &ADFAttrs{Level: 2},
					Content: []ADFContent{{Type: "text", Text: "Heading text"}},
				},
			},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if len(quote.Content) != 2 {
		t.Fatalf("expected 2 children, got %d", len(quote.Content))
	}
	if quote.Content[0].Type != "paragraph" || quote.Content[0].Content[0].Text != "Paragraph text" {
		t.Fatalf("expected hoisted paragraph as first child")
	}
	if quote.Content[1].Type != "paragraph" {
		t.Fatalf("expected flattened paragraph as second child, got %q", quote.Content[1].Type)
	}
	if !strings.Contains(quote.Content[1].Content[0].Text, "Heading text") {
		t.Fatalf("expected 'Heading text' in flattened paragraph, got %q", quote.Content[1].Content[0].Text)
	}
}

func TestQuoteADF_DeeplyNestedBlockNodeHoisted(t *testing.T) {
	original := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type: "table",
			Content: []ADFContent{{
				Type: "tableRow",
				Content: []ADFContent{{
					Type: "tableCell",
					Content: []ADFContent{{
						Type:    "paragraph",
						Content: []ADFContent{{Type: "text", Text: "Cell text"}},
					}},
				}},
			}},
		}},
	}

	quote := QuoteADF(original)
	if quote == nil {
		t.Fatal("expected a blockquote node")
	}
	if len(quote.Content) != 1 {
		t.Fatalf("expected 1 hoisted paragraph, got %d children", len(quote.Content))
	}
	if quote.Content[0].Type != "paragraph" {
		t.Fatalf("expected paragraph, got %q", quote.Content[0].Type)
	}
	if quote.Content[0].Content[0].Text != "Cell text" {
		t.Fatalf("expected 'Cell text', got %q", quote.Content[0].Content[0].Text)
	}
}
