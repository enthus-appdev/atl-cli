package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// mentionDocJSON is an ADF document containing a single unresolved mention
// inside a paragraph, as it would arrive from Jira. Unmarshalling it (rather
// than building an ADFContent literal in Go) is what populates RawAttrs.
const mentionDocJSON = `{
	"type": "doc",
	"version": 1,
	"content": [
		{
			"type": "paragraph",
			"content": [
				{
					"type": "mention",
					"attrs": {"id": "", "text": "@Jane Doe"}
				}
			]
		}
	]
}`

func TestResolveMentions_RawAttrs_ResolvedIDSurvivesMarshal(t *testing.T) {
	var doc ADF
	if err := json.Unmarshal([]byte(mentionDocJSON), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	resolver := func(ctx context.Context, name string) (string, error) {
		return "acct-123", nil
	}

	if err := ResolveMentions(context.Background(), &doc, resolver); err != nil {
		t.Fatalf("ResolveMentions: %v", err)
	}

	out, err := json.Marshal(&doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(out), "acct-123") {
		t.Errorf("expected marshaled output to contain resolved account id, got %s", out)
	}
}

func TestResolveMentions_RawAttrs_UnresolvedDropsAttrs(t *testing.T) {
	var doc ADF
	if err := json.Unmarshal([]byte(mentionDocJSON), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	resolver := func(ctx context.Context, name string) (string, error) {
		return "", nil
	}

	if err := ResolveMentions(context.Background(), &doc, resolver); err != nil {
		t.Fatalf("ResolveMentions: %v", err)
	}

	out, err := json.Marshal(&doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed struct {
		Content []struct {
			Content []map[string]interface{} `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}

	node := parsed.Content[0].Content[0]
	if node["type"] != "text" {
		t.Errorf("expected node type 'text', got %v", node["type"])
	}
	if node["text"] != "@Jane Doe" {
		t.Errorf("expected text '@Jane Doe', got %v", node["text"])
	}
	if _, ok := node["attrs"]; ok {
		t.Errorf("expected no 'attrs' key on text node, got %s", out)
	}
}
