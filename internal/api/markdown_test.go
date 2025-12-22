package api

import (
	"encoding/json"
	"testing"
)

func TestMarkdownToADF_Empty(t *testing.T) {
	adf := MarkdownToADF("")
	if adf.Type != "doc" {
		t.Errorf("expected type 'doc', got %q", adf.Type)
	}
	if adf.Version != 1 {
		t.Errorf("expected version 1, got %d", adf.Version)
	}
	if len(adf.Content) != 0 {
		t.Errorf("expected 0 content blocks, got %d", len(adf.Content))
	}
}

func TestMarkdownToADF_PlainText(t *testing.T) {
	adf := MarkdownToADF("Hello, World!")

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	para := adf.Content[0]
	if para.Type != "paragraph" {
		t.Errorf("expected paragraph, got %q", para.Type)
	}

	if len(para.Content) != 1 {
		t.Fatalf("expected 1 text node, got %d", len(para.Content))
	}

	if para.Content[0].Text != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", para.Content[0].Text)
	}
}

func TestMarkdownToADF_MultipleParagraphs(t *testing.T) {
	adf := MarkdownToADF("First paragraph\n\nSecond paragraph")

	if len(adf.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(adf.Content))
	}

	if adf.Content[0].Content[0].Text != "First paragraph" {
		t.Errorf("expected 'First paragraph', got %q", adf.Content[0].Content[0].Text)
	}

	if adf.Content[1].Content[0].Text != "Second paragraph" {
		t.Errorf("expected 'Second paragraph', got %q", adf.Content[1].Content[0].Text)
	}
}

func TestMarkdownToADF_Headings(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLevel int
		wantText  string
	}{
		{"h1", "# Heading 1", 1, "Heading 1"},
		{"h2", "## Heading 2", 2, "Heading 2"},
		{"h3", "### Heading 3", 3, "Heading 3"},
		{"h4", "#### Heading 4", 4, "Heading 4"},
		{"h5", "##### Heading 5", 5, "Heading 5"},
		{"h6", "###### Heading 6", 6, "Heading 6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adf := MarkdownToADF(tt.input)

			if len(adf.Content) != 1 {
				t.Fatalf("expected 1 content block, got %d", len(adf.Content))
			}

			heading := adf.Content[0]
			if heading.Type != "heading" {
				t.Errorf("expected heading, got %q", heading.Type)
			}

			if heading.Attrs == nil || heading.Attrs.Level != tt.wantLevel {
				t.Errorf("expected level %d, got %v", tt.wantLevel, heading.Attrs)
			}

			if len(heading.Content) != 1 || heading.Content[0].Text != tt.wantText {
				t.Errorf("expected text %q, got %v", tt.wantText, heading.Content)
			}
		})
	}
}

func TestMarkdownToADF_Bold(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"asterisks", "**bold text**"},
		{"underscores", "__bold text__"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adf := MarkdownToADF(tt.input)

			if len(adf.Content) != 1 {
				t.Fatalf("expected 1 content block, got %d", len(adf.Content))
			}

			para := adf.Content[0]
			if len(para.Content) != 1 {
				t.Fatalf("expected 1 text node, got %d", len(para.Content))
			}

			textNode := para.Content[0]
			if textNode.Text != "bold text" {
				t.Errorf("expected 'bold text', got %q", textNode.Text)
			}

			if len(textNode.Marks) != 1 || textNode.Marks[0].Type != "strong" {
				t.Errorf("expected strong mark, got %v", textNode.Marks)
			}
		})
	}
}

func TestMarkdownToADF_Italic(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"asterisks", "*italic text*"},
		{"underscores", "_italic text_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adf := MarkdownToADF(tt.input)

			if len(adf.Content) != 1 {
				t.Fatalf("expected 1 content block, got %d", len(adf.Content))
			}

			para := adf.Content[0]
			if len(para.Content) != 1 {
				t.Fatalf("expected 1 text node, got %d", len(para.Content))
			}

			textNode := para.Content[0]
			if textNode.Text != "italic text" {
				t.Errorf("expected 'italic text', got %q", textNode.Text)
			}

			if len(textNode.Marks) != 1 || textNode.Marks[0].Type != "em" {
				t.Errorf("expected em mark, got %v", textNode.Marks)
			}
		})
	}
}

func TestMarkdownToADF_InlineCode(t *testing.T) {
	adf := MarkdownToADF("Use `code` here")

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	para := adf.Content[0]
	if len(para.Content) != 3 {
		t.Fatalf("expected 3 text nodes, got %d", len(para.Content))
	}

	// Check the code node
	codeNode := para.Content[1]
	if codeNode.Text != "code" {
		t.Errorf("expected 'code', got %q", codeNode.Text)
	}

	if len(codeNode.Marks) != 1 || codeNode.Marks[0].Type != "code" {
		t.Errorf("expected code mark, got %v", codeNode.Marks)
	}
}

func TestMarkdownToADF_Link(t *testing.T) {
	adf := MarkdownToADF("Check [this link](https://example.com)")

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	para := adf.Content[0]

	// Find the link node
	var linkNode *ADFContent
	for i := range para.Content {
		if len(para.Content[i].Marks) > 0 && para.Content[i].Marks[0].Type == "link" {
			linkNode = &para.Content[i]
			break
		}
	}

	if linkNode == nil {
		t.Fatal("expected to find a link node")
	}

	if linkNode.Text != "this link" {
		t.Errorf("expected 'this link', got %q", linkNode.Text)
	}

	if linkNode.Marks[0].Attrs == nil || linkNode.Marks[0].Attrs.Href != "https://example.com" {
		t.Errorf("expected href 'https://example.com', got %v", linkNode.Marks[0].Attrs)
	}
}

func TestMarkdownToADF_CodeBlock(t *testing.T) {
	input := "```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```"
	adf := MarkdownToADF(input)

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	codeBlock := adf.Content[0]
	if codeBlock.Type != "codeBlock" {
		t.Errorf("expected codeBlock, got %q", codeBlock.Type)
	}

	if codeBlock.Attrs == nil || codeBlock.Attrs.Language != "go" {
		t.Errorf("expected language 'go', got %v", codeBlock.Attrs)
	}

	expectedCode := "func main() {\n\tfmt.Println(\"Hello\")\n}"
	if len(codeBlock.Content) != 1 || codeBlock.Content[0].Text != expectedCode {
		t.Errorf("expected code %q, got %v", expectedCode, codeBlock.Content)
	}
}

func TestMarkdownToADF_BulletList(t *testing.T) {
	input := "- Item 1\n- Item 2\n- Item 3"
	adf := MarkdownToADF(input)

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	list := adf.Content[0]
	if list.Type != "bulletList" {
		t.Errorf("expected bulletList, got %q", list.Type)
	}

	if len(list.Content) != 3 {
		t.Fatalf("expected 3 list items, got %d", len(list.Content))
	}

	for i, item := range list.Content {
		if item.Type != "listItem" {
			t.Errorf("item %d: expected listItem, got %q", i, item.Type)
		}
	}
}

func TestMarkdownToADF_OrderedList(t *testing.T) {
	input := "1. First\n2. Second\n3. Third"
	adf := MarkdownToADF(input)

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	list := adf.Content[0]
	if list.Type != "orderedList" {
		t.Errorf("expected orderedList, got %q", list.Type)
	}

	if len(list.Content) != 3 {
		t.Fatalf("expected 3 list items, got %d", len(list.Content))
	}
}

func TestMarkdownToADF_Blockquote(t *testing.T) {
	input := "> This is a quote\n> with multiple lines"
	adf := MarkdownToADF(input)

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	quote := adf.Content[0]
	if quote.Type != "blockquote" {
		t.Errorf("expected blockquote, got %q", quote.Type)
	}
}

func TestMarkdownToADF_HorizontalRule(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"dashes", "---"},
		{"asterisks", "***"},
		{"underscores", "___"},
		{"spaced dashes", "- - -"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adf := MarkdownToADF(tt.input)

			if len(adf.Content) != 1 {
				t.Fatalf("expected 1 content block, got %d", len(adf.Content))
			}

			rule := adf.Content[0]
			if rule.Type != "rule" {
				t.Errorf("expected rule, got %q", rule.Type)
			}
		})
	}
}

func TestMarkdownToADF_Strikethrough(t *testing.T) {
	adf := MarkdownToADF("~~deleted text~~")

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	para := adf.Content[0]
	if len(para.Content) != 1 {
		t.Fatalf("expected 1 text node, got %d", len(para.Content))
	}

	textNode := para.Content[0]
	if textNode.Text != "deleted text" {
		t.Errorf("expected 'deleted text', got %q", textNode.Text)
	}

	if len(textNode.Marks) != 1 || textNode.Marks[0].Type != "strike" {
		t.Errorf("expected strike mark, got %v", textNode.Marks)
	}
}

func TestMarkdownToADF_Complex(t *testing.T) {
	input := `# Goals

- Goal 1
- Goal 2

## Implementation

This is a **bold** statement with *italic* text.

` + "```" + `javascript
console.log("Hello");
` + "```" + `

> Important note here

---

Check [the docs](https://example.com) for more info.`

	adf := MarkdownToADF(input)

	// Should have: heading, bullet list, heading, paragraph, code block, blockquote, rule, paragraph
	if len(adf.Content) < 7 {
		t.Errorf("expected at least 7 content blocks, got %d", len(adf.Content))
	}

	// Verify first heading
	if adf.Content[0].Type != "heading" {
		t.Errorf("expected heading first, got %q", adf.Content[0].Type)
	}
	if adf.Content[0].Attrs.Level != 1 {
		t.Errorf("expected h1, got level %d", adf.Content[0].Attrs.Level)
	}
}

func TestMarkdownToADF_JSONOutput(t *testing.T) {
	input := `## Goals

- Goal 1`

	adf := MarkdownToADF(input)

	// Verify it produces valid JSON
	jsonBytes, err := json.MarshalIndent(adf, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal ADF to JSON: %v", err)
	}

	// Verify structure matches expected ADF format
	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if result["type"] != "doc" {
		t.Errorf("expected type 'doc', got %v", result["type"])
	}

	if result["version"].(float64) != 1 {
		t.Errorf("expected version 1, got %v", result["version"])
	}

	content := result["content"].([]interface{})
	if len(content) != 2 {
		t.Errorf("expected 2 content blocks, got %d", len(content))
	}
}

// TestTextToADFBackwardCompatibility ensures the updated TextToADF
// still handles plain text correctly (backward compatibility).
func TestTextToADFBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{"single paragraph", "Hello, World!", 1},
		{"multiple paragraphs", "First\n\nSecond", 2},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adf := TextToADF(tt.input)

			if adf.Type != "doc" {
				t.Errorf("expected type 'doc', got %q", adf.Type)
			}
			if adf.Version != 1 {
				t.Errorf("expected version 1, got %d", adf.Version)
			}
			if len(adf.Content) != tt.wantLen {
				t.Errorf("expected %d content blocks, got %d", tt.wantLen, len(adf.Content))
			}
		})
	}
}

func TestMarkdownToADF_Table(t *testing.T) {
	input := `| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
| Cell 3   | Cell 4   |`

	adf := MarkdownToADF(input)

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	table := adf.Content[0]
	if table.Type != "table" {
		t.Errorf("expected type 'table', got %q", table.Type)
	}

	// Should have 3 rows: 1 header + 2 data rows
	if len(table.Content) != 3 {
		t.Errorf("expected 3 rows, got %d", len(table.Content))
	}

	// First row should be header
	headerRow := table.Content[0]
	if headerRow.Type != "tableRow" {
		t.Errorf("expected tableRow, got %q", headerRow.Type)
	}

	// Header cells should be tableHeader
	if len(headerRow.Content) != 2 {
		t.Errorf("expected 2 header cells, got %d", len(headerRow.Content))
	}
	if headerRow.Content[0].Type != "tableHeader" {
		t.Errorf("expected tableHeader, got %q", headerRow.Content[0].Type)
	}

	// Data cells should be tableCell
	dataRow := table.Content[1]
	if dataRow.Content[0].Type != "tableCell" {
		t.Errorf("expected tableCell, got %q", dataRow.Content[0].Type)
	}
}

func TestMarkdownToADF_Panel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		panelType string
	}{
		{"info", ":::info\nContent\n:::", "info"},
		{"warning", ":::warning\nContent\n:::", "warning"},
		{"error", ":::error\nContent\n:::", "error"},
		{"note", ":::note\nContent\n:::", "note"},
		{"success", ":::success\nContent\n:::", "success"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adf := MarkdownToADF(tt.input)

			if len(adf.Content) != 1 {
				t.Fatalf("expected 1 content block, got %d", len(adf.Content))
			}

			panel := adf.Content[0]
			if panel.Type != "panel" {
				t.Errorf("expected type 'panel', got %q", panel.Type)
			}

			if panel.Attrs == nil {
				t.Fatal("expected attrs to be set")
			}

			if panel.Attrs.PanelType != tt.panelType {
				t.Errorf("expected panelType %q, got %q", tt.panelType, panel.Attrs.PanelType)
			}
		})
	}
}

func TestMarkdownToADF_Expand(t *testing.T) {
	input := `+++Click to expand
Hidden content here
+++`

	adf := MarkdownToADF(input)

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	expand := adf.Content[0]
	if expand.Type != "expand" {
		t.Errorf("expected type 'expand', got %q", expand.Type)
	}

	if expand.Attrs == nil {
		t.Fatal("expected attrs to be set")
	}

	if expand.Attrs.Title != "Click to expand" {
		t.Errorf("expected title 'Click to expand', got %q", expand.Attrs.Title)
	}

	// Should have inner content
	if len(expand.Content) == 0 {
		t.Error("expected expand to have content")
	}
}

func TestMarkdownToADF_Media(t *testing.T) {
	input := `Check this: !media[abc123]`

	adf := MarkdownToADF(input)

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	para := adf.Content[0]
	if para.Type != "paragraph" {
		t.Errorf("expected paragraph, got %q", para.Type)
	}

	// Should have text and mediaSingle
	if len(para.Content) < 2 {
		t.Fatalf("expected at least 2 inline elements, got %d", len(para.Content))
	}

	// Find the mediaSingle
	var media *ADFContent
	for i := range para.Content {
		if para.Content[i].Type == "mediaSingle" {
			media = &para.Content[i]
			break
		}
	}

	if media == nil {
		t.Fatal("expected to find mediaSingle")
	}

	if len(media.Content) != 1 {
		t.Fatalf("expected 1 media child, got %d", len(media.Content))
	}

	if media.Content[0].Type != "media" {
		t.Errorf("expected media, got %q", media.Content[0].Type)
	}

	if media.Content[0].Attrs == nil || media.Content[0].Attrs.ID != "abc123" {
		t.Error("expected media ID to be 'abc123'")
	}
}

func TestMarkdownToADF_MediaWithCollection(t *testing.T) {
	input := `!media[my-collection:abc123]`

	adf := MarkdownToADF(input)

	para := adf.Content[0]
	media := para.Content[0]

	if media.Type != "mediaSingle" {
		t.Errorf("expected mediaSingle, got %q", media.Type)
	}

	innerMedia := media.Content[0]
	if innerMedia.Attrs.Collection != "my-collection" {
		t.Errorf("expected collection 'my-collection', got %q", innerMedia.Attrs.Collection)
	}
	if innerMedia.Attrs.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %q", innerMedia.Attrs.ID)
	}
}

func TestMarkdownToADF_Combined(t *testing.T) {
	input := `# Test Document

:::info
Important information
:::

| Name | Value |
|------|-------|
| Foo  | Bar   |

+++Details
More info
+++`

	adf := MarkdownToADF(input)

	// Should have: heading, panel, table, expand
	if len(adf.Content) != 4 {
		t.Fatalf("expected 4 content blocks, got %d", len(adf.Content))
	}

	if adf.Content[0].Type != "heading" {
		t.Errorf("expected heading, got %q", adf.Content[0].Type)
	}
	if adf.Content[1].Type != "panel" {
		t.Errorf("expected panel, got %q", adf.Content[1].Type)
	}
	if adf.Content[2].Type != "table" {
		t.Errorf("expected table, got %q", adf.Content[2].Type)
	}
	if adf.Content[3].Type != "expand" {
		t.Errorf("expected expand, got %q", adf.Content[3].Type)
	}
}
