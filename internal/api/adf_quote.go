package api

import (
	"fmt"
	"strings"
)

// blockquoteAllowedChildren is the ADF content model for the blockquote node.
// A child outside this set makes Jira reject the entire document, so anything
// else must be degraded rather than passed through.
var blockquoteAllowedChildren = map[string]bool{
	"paragraph":   true,
	"bulletList":  true,
	"orderedList": true,
	"codeBlock":   true,
	"mediaGroup":  true,
	"mediaSingle": true,
}

// QuoteADF wraps a document's nodes in a blockquote, copying legal children
// verbatim so media, mentions and exact characters survive. Returns nil when
// nothing quotable remains — a blockquote with no children is invalid ADF.
func QuoteADF(original *ADF) *ADFContent {
	if original == nil {
		return nil
	}

	children := make([]ADFContent, 0, len(original.Content))
	for _, node := range original.Content {
		if child, ok := quoteChild(node); ok {
			children = append(children, child)
		}
	}

	if len(children) == 0 {
		return nil
	}

	return &ADFContent{Type: "blockquote", Content: children}
}

// quoteChild returns the node as a legal blockquote child. Illegal nodes are
// flattened to a paragraph of their rendered text; the bool is false when that
// flattening yields nothing worth emitting.
func quoteChild(node ADFContent) (ADFContent, bool) {
	if blockquoteAllowedChildren[node.Type] {
		// The spec forbids node-level marks on a quoted paragraph. node is a
		// copy, so the caller's slice is untouched.
		if node.Type == "paragraph" {
			node.Marks = nil
		}
		return node, true
	}

	text := ADFToText(&ADF{Type: "doc", Version: 1, Content: []ADFContent{node}})
	if strings.TrimSpace(text) == "" {
		return ADFContent{}, false
	}

	return ADFContent{
		Type:    "paragraph",
		Content: []ADFContent{{Type: "text", Text: text}},
	}, true
}

// AttributionParagraph builds the "Replying to <author>:" line that precedes a quote.
func AttributionParagraph(author string) ADFContent {
	return ADFContent{
		Type: "paragraph",
		Content: []ADFContent{{
			Type:  "text",
			Text:  fmt.Sprintf("Replying to %s:", author),
			Marks: []ADFMark{{Type: "em"}},
		}},
	}
}
