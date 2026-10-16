package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/enthus-appdev/atl-cli/internal/auth"
)

// commentRecorder captures the requests a JiraService sends while adding a comment.
type commentRecorder struct {
	projectLookups int
	posted         []map[string]any
}

// newCommentTestService returns a JiraService whose issue lies in a project of
// projectType and that records every request into rec.
func newCommentTestService(t *testing.T, projectType string, rec *commentRecorder) *JiraService {
	t.Helper()

	client := &Client{
		cloudID: "test-cloud",
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
		httpClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			var body string
			switch request.Method {
			case http.MethodGet:
				if got := request.URL.String(); got != "https://api.atlassian.com/ex/jira/test-cloud/rest/api/3/issue/SD-1?fields=project" {
					t.Errorf("unexpected project lookup URL: %s", got)
				}
				rec.projectLookups++
				body = fmt.Sprintf(`{"fields":{"project":{"key":"SD","projectTypeKey":%q}}}`, projectType)
			case http.MethodPost:
				var posted map[string]any
				if err := json.NewDecoder(request.Body).Decode(&posted); err != nil {
					t.Fatalf("decode posted comment: %v", err)
				}
				rec.posted = append(rec.posted, posted)
				body = `{"id":"123"}`
			default:
				t.Fatalf("unexpected method: %s", request.Method)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    request,
			}, nil
		})},
	}
	return NewJiraService(client)
}

func TestAddCommentWithOptions_InternalSetsServiceDeskProperty(t *testing.T) {
	var rec commentRecorder
	jira := newCommentTestService(t, "service_desk", &rec)

	if _, err := jira.AddCommentWithOptions(context.Background(), "SD-1", &CommentOptions{Body: "note", Internal: true}); err != nil {
		t.Fatalf("AddCommentWithOptions returned an error: %v", err)
	}
	if len(rec.posted) != 1 {
		t.Fatalf("expected one posted comment, got %d", len(rec.posted))
	}

	props, ok := rec.posted[0]["properties"].([]any)
	if !ok || len(props) != 1 {
		t.Fatalf("expected exactly one comment property, got %#v", rec.posted[0]["properties"])
	}
	prop, _ := props[0].(map[string]any)
	if prop["key"] != "sd.public.comment" {
		t.Errorf("expected property key sd.public.comment, got %#v", prop["key"])
	}
	value, _ := prop["value"].(map[string]any)
	if value["internal"] != true {
		t.Errorf("expected property value {internal: true}, got %#v", prop["value"])
	}
}

func TestAddCommentWithOptions_InternalRejectsNonServiceDeskIssue(t *testing.T) {
	var rec commentRecorder
	jira := newCommentTestService(t, "software", &rec)

	_, err := jira.AddCommentWithOptions(context.Background(), "SD-1", &CommentOptions{Body: "note", Internal: true})
	if err == nil {
		t.Fatal("expected an error for an internal comment outside Jira Service Management")
	}
	if !strings.Contains(err.Error(), "Jira Service Management") {
		t.Errorf("expected error to name Jira Service Management, got %q", err.Error())
	}
	if len(rec.posted) != 0 {
		t.Errorf("expected no comment to be posted, got %d", len(rec.posted))
	}
}

func TestAddCommentWithOptions_PublicSkipsProjectLookupAndProperties(t *testing.T) {
	var rec commentRecorder
	jira := newCommentTestService(t, "software", &rec)

	if _, err := jira.AddCommentWithOptions(context.Background(), "SD-1", &CommentOptions{Body: "reply"}); err != nil {
		t.Fatalf("AddCommentWithOptions returned an error: %v", err)
	}
	if rec.projectLookups != 0 {
		t.Errorf("expected no project lookup for a regular comment, got %d", rec.projectLookups)
	}
	if len(rec.posted) != 1 {
		t.Fatalf("expected one posted comment, got %d", len(rec.posted))
	}
	if _, ok := rec.posted[0]["properties"]; ok {
		t.Errorf("expected no properties for a regular comment, got %#v", rec.posted[0]["properties"])
	}
}
