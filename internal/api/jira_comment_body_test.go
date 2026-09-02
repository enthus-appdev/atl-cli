package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/enthus-appdev/atl-cli/internal/auth"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

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

func TestAddCommentWithOptionsPostsPrebuiltADF(t *testing.T) {
	prebuilt := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{{
			Type: "paragraph",
			Content: []ADFContent{
				{Type: "mention", Attrs: &ADFAttrs{ID: "account-123", Text: "@Alex Example"}},
				{Type: "text", Text: "Replying to comment 987", Marks: []ADFMark{{Type: "link", Attrs: &ADFAttrs{Href: "https://example.atlassian.net/browse/PROJ-42?focusedCommentId=987"}}}},
			},
		}},
	}

	client := &Client{
		cloudID: "test-cloud",
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
		httpClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", request.Method)
			}
			if got := request.URL.String(); got != "https://api.atlassian.com/ex/jira/test-cloud/rest/api/3/issue/PROJ-42/comment" {
				t.Errorf("unexpected URL: %s", got)
			}

			var posted AddCommentRequest
			if err := json.NewDecoder(request.Body).Decode(&posted); err != nil {
				t.Fatalf("decode posted comment: %v", err)
			}
			if posted.Body == nil || len(posted.Body.Content) != 1 || len(posted.Body.Content[0].Content) != 2 {
				t.Fatalf("unexpected posted ADF structure: %#v", posted.Body)
			}
			mention := posted.Body.Content[0].Content[0]
			if mention.Attrs == nil || mention.Attrs.ID != "account-123" || mention.Attrs.Text != "@Alex Example" {
				t.Errorf("unexpected posted mention: %#v", mention)
			}
			link := posted.Body.Content[0].Content[1]
			if len(link.Marks) != 1 || link.Marks[0].Attrs == nil || link.Marks[0].Attrs.Href != "https://example.atlassian.net/browse/PROJ-42?focusedCommentId=987" {
				t.Errorf("unexpected posted link: %#v", link)
			}

			return &http.Response{
				StatusCode: http.StatusCreated,
				Status:     "201 Created",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"id":"123"}`)),
				Request:    request,
			}, nil
		})},
	}

	comment, err := NewJiraService(client).AddCommentWithOptions(context.Background(), "PROJ-42", &CommentOptions{BodyADF: prebuilt})
	if err != nil {
		t.Fatalf("AddCommentWithOptions returned an error: %v", err)
	}
	if comment.ID != "123" {
		t.Errorf("unexpected comment ID: %q", comment.ID)
	}
}
