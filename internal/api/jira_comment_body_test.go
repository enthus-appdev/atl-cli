package api

import (
	"context"
	"testing"
)

func TestCommentBodyADF_PrefersBodyADF(t *testing.T) {
	prebuilt := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type:    "paragraph",
			Content: []ADFContent{{Type: "text", Text: "prebuilt"}},
		}},
	}

	var s *JiraService
	got, err := s.commentBodyADF(context.Background(), &CommentOptions{
		Body:    "this markdown must be ignored",
		BodyADF: prebuilt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != prebuilt {
		t.Fatal("expected BodyADF to be returned unchanged")
	}
}
