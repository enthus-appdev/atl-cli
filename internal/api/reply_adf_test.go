package api

import (
	"net/url"
	"testing"
)

func TestBuildReplyADF_MentionsAuthorAndLinksOriginalWithoutQuoting(t *testing.T) {
	body := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type:    "paragraph",
			Content: []ADFContent{{Type: "text", Text: "The actual reply"}},
		}},
	}
	author := &User{AccountID: " account-123 ", DisplayName: " Alex Example "}

	doc := BuildReplyADF("example.atlassian.net", "PROJ-42", "987:654", author, body)

	if len(doc.Content) != 2 {
		t.Fatalf("expected reply header and body only, got %d top-level nodes", len(doc.Content))
	}
	if doc.Content[0].Type != "paragraph" {
		t.Fatalf("expected reply header paragraph, got %q", doc.Content[0].Type)
	}
	if len(doc.Content[0].Content) != 3 {
		t.Fatalf("expected mention, separator, and link, got %+v", doc.Content[0].Content)
	}

	mention := doc.Content[0].Content[0]
	if mention.Type != "mention" || mention.Attrs == nil {
		t.Fatalf("expected real ADF mention, got %+v", mention)
	}
	if mention.Attrs.ID != "account-123" || mention.Attrs.Text != "@Alex Example" {
		t.Errorf("unexpected mention attrs: %+v", mention.Attrs)
	}

	separator := doc.Content[0].Content[1]
	if separator.Type != "text" || separator.Text != replySeparator {
		t.Errorf("unexpected reply separator: %+v", separator)
	}

	link := doc.Content[0].Content[2]
	if link.Text != "Replying to comment 987:654" || len(link.Marks) != 1 || link.Marks[0].Type != "link" || link.Marks[0].Attrs == nil {
		t.Fatalf("expected focused-comment link, got %+v", link)
	}
	parsedURL, err := url.Parse(link.Marks[0].Attrs.Href)
	if err != nil {
		t.Fatalf("parse reply URL: %v", err)
	}
	if parsedURL.Scheme != "https" || parsedURL.Host != "example.atlassian.net" || parsedURL.Path != "/browse/PROJ-42" {
		t.Errorf("unexpected reply URL: %s", parsedURL)
	}
	if got := parsedURL.Query().Get("focusedCommentId"); got != "987:654" {
		t.Errorf("unexpected focused comment ID: %q", got)
	}
	if got := doc.Content[1].Content[0].Text; got != "The actual reply" {
		t.Errorf("expected reply body unchanged, got %q", got)
	}
	for _, node := range doc.Content {
		if node.Type == "blockquote" {
			t.Fatal("reply must not copy the original comment into a blockquote")
		}
	}
}

func TestBuildReplyADF_StillLinksWhenAuthorUnavailable(t *testing.T) {
	doc := BuildReplyADF("example.atlassian.net", "PROJ-42", "987", &User{AccountID: "  "}, nil)

	if len(doc.Content) != 1 || len(doc.Content[0].Content) != 1 {
		t.Fatalf("expected link-only header, got %+v", doc.Content)
	}
	link := doc.Content[0].Content[0]
	if link.Type != "text" || len(link.Marks) != 1 || link.Marks[0].Type != "link" {
		t.Fatalf("expected focused-comment link without a mention, got %+v", link)
	}
}
