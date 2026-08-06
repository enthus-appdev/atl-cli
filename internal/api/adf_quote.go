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

	children := quoteNodes(original.Content)

	if len(children) == 0 {
		return nil
	}

	return &ADFContent{Type: "blockquote", Content: children}
}

// quoteNodes returns zero or more legal blockquote children from a slice of nodes.
func quoteNodes(nodes []ADFContent) []ADFContent {
	out := make([]ADFContent, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, quoteNode(node)...)
	}
	return out
}

// quoteNode returns the node as zero or more legal blockquote children. A node
// the content model rejects contributes its legal descendants instead, so media
// and text nested inside a container survive; only a subtree with nothing legal
// in it collapses to rendered text.
func quoteNode(node ADFContent) []ADFContent {
	if blockquoteAllowedChildren[node.Type] {
		// The spec forbids node-level marks on a quoted paragraph. node is a
		// copy, so the caller's slice is untouched.
		if node.Type == "paragraph" {
			node.Marks = nil
		}
		return []ADFContent{node}
	}

	if hoisted := quoteNodes(node.Content); len(hoisted) > 0 {
		return hoisted
	}

	text := ADFToText(&ADF{Type: "doc", Version: 1, Content: []ADFContent{node}})
	if strings.TrimSpace(text) == "" {
		return nil
	}

	return []ADFContent{{
		Type:    "paragraph",
		Content: []ADFContent{{Type: "text", Text: text}},
	}}
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
