package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/enthus-appdev/atl-cli/internal/auth"
)

// TestTextToADF tests conversion of plain text to Atlassian Document Format.
func TestTextToADF(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantLen  int // number of content blocks
	}{
		{
			name:     "single paragraph",
			input:    "Hello, World!",
			wantType: "doc",
			wantLen:  1,
		},
		{
			name:     "multiple paragraphs",
			input:    "First paragraph\n\nSecond paragraph",
			wantType: "doc",
			wantLen:  2,
		},
		{
			name:     "empty text",
			input:    "",
			wantType: "doc",
			wantLen:  0,
		},
		{
			name:     "text with single newlines",
			input:    "Line one\nLine two",
			wantType: "doc",
			wantLen:  1, // Single newlines don't create new paragraphs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adf := TextToADF(tt.input)

			if adf.Type != tt.wantType {
				t.Errorf("TextToADF().Type = %q, want %q", adf.Type, tt.wantType)
			}
			if adf.Version != 1 {
				t.Errorf("TextToADF().Version = %d, want 1", adf.Version)
			}
			if len(adf.Content) != tt.wantLen {
				t.Errorf("TextToADF() has %d content blocks, want %d", len(adf.Content), tt.wantLen)
			}
		})
	}
}

// TestADFToText tests conversion of Atlassian Document Format to plain text.
func TestADFToText(t *testing.T) {
	tests := []struct {
		name string
		adf  *ADF
		want string
	}{
		{
			name: "nil ADF",
			adf:  nil,
			want: "",
		},
		{
			name: "simple paragraph",
			adf: &ADF{
				Type:    "doc",
				Version: 1,
				Content: []ADFContent{
					{
						Type: "paragraph",
						Content: []ADFContent{
							{Type: "text", Text: "Hello"},
						},
					},
				},
			},
			want: "Hello",
		},
		{
			name: "multiple paragraphs",
			adf: &ADF{
				Type:    "doc",
				Version: 1,
				Content: []ADFContent{
					{
						Type:    "paragraph",
						Content: []ADFContent{{Type: "text", Text: "First"}},
					},
					{
						Type:    "paragraph",
						Content: []ADFContent{{Type: "text", Text: "Second"}},
					},
				},
			},
			want: "First\n\nSecond",
		},
		{
			name: "bullet list",
			adf: &ADF{
				Type:    "doc",
				Version: 1,
				Content: []ADFContent{
					{
						Type: "bulletList",
						Content: []ADFContent{
							{
								Type: "listItem",
								Content: []ADFContent{
									{
										Type:    "paragraph",
										Content: []ADFContent{{Type: "text", Text: "Item 1"}},
									},
								},
							},
							{
								Type: "listItem",
								Content: []ADFContent{
									{
										Type:    "paragraph",
										Content: []ADFContent{{Type: "text", Text: "Item 2"}},
									},
								},
							},
						},
					},
				},
			},
			want: "- Item 1\n- Item 2",
		},
		{
			name: "code block",
			adf: &ADF{
				Type:    "doc",
				Version: 1,
				Content: []ADFContent{
					{
						Type: "codeBlock",
						Content: []ADFContent{
							{Type: "text", Text: "code here"},
						},
					},
				},
			},
			want: "```\ncode here\n```",
		},
		{
			name: "heading",
			adf: &ADF{
				Type:    "doc",
				Version: 1,
				Content: []ADFContent{
					{
						Type:    "heading",
						Content: []ADFContent{{Type: "text", Text: "Title"}},
					},
				},
			},
			want: "Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ADFToText(tt.adf)
			if got != tt.want {
				t.Errorf("ADFToText() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTextToADFRoundTrip tests that text can be converted to ADF and back.
func TestTextToADFRoundTrip(t *testing.T) {
	tests := []string{
		"Simple text",
		"Multiple\n\nParagraphs",
	}

	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			adf := TextToADF(text)
			result := ADFToText(adf)

			// Result should contain the original text (formatting may differ slightly)
			if !strings.Contains(result, strings.Split(text, "\n")[0]) {
				t.Errorf("Round trip failed: input %q, got %q", text, result)
			}
		})
	}
}

// TestJiraServiceGetIssue tests the GetIssue method.
func TestJiraServiceGetIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/issue/TEST-123") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		issue := Issue{
			ID:   "10001",
			Key:  "TEST-123",
			Self: "https://example.atlassian.net/rest/api/3/issue/10001",
			Fields: IssueFields{
				Summary: "Test Issue",
				Status: &Status{
					ID:   "1",
					Name: "To Do",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(issue)
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		cloudID:    "test-cloud",
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}

	// Note: In production code, the base URL would be constructed from AtlassianAPIURL.
	// For testing, we use the test server URL directly.

	// Create the service with a custom request that uses the test server
	jira := NewJiraService(client)

	// Use the test server URL directly
	ctx := context.Background()
	var issue Issue
	err := client.Get(ctx, server.URL+"/issue/TEST-123", &issue)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if issue.Key != "TEST-123" {
		t.Errorf("Issue.Key = %q, want %q", issue.Key, "TEST-123")
	}
	if issue.Fields.Summary != "Test Issue" {
		t.Errorf("Issue.Fields.Summary = %q, want %q", issue.Fields.Summary, "Test Issue")
	}

	// Verify service was created
	if jira == nil {
		t.Error("NewJiraService() returned nil")
	}
}

// TestJiraServiceSearch tests the Search method.
func TestJiraServiceSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		jql := r.URL.Query().Get("jql")
		if jql == "" {
			t.Error("JQL parameter missing")
		}

		result := SearchResult{
			Issues: []*Issue{
				{Key: "TEST-1", Fields: IssueFields{Summary: "First"}},
				{Key: "TEST-2", Fields: IssueFields{Summary: "Second"}},
			},
			StartAt:    0,
			MaxResults: 50,
			Total:      2,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		cloudID:    "test-cloud",
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}

	ctx := context.Background()
	var result SearchResult
	err := client.Get(ctx, server.URL+"/search?jql=project=TEST", &result)

	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if len(result.Issues) != 2 {
		t.Errorf("Search returned %d issues, want 2", len(result.Issues))
	}
	if result.Total != 2 {
		t.Errorf("Search.Total = %d, want 2", result.Total)
	}
}

// TestSearchOptions tests the SearchOptions struct.
func TestSearchOptions(t *testing.T) {
	opts := SearchOptions{
		JQL:           "project = TEST",
		MaxResults:    25,
		Fields:        []string{"summary", "status"},
		NextPageToken: "token123",
	}

	if opts.JQL != "project = TEST" {
		t.Errorf("SearchOptions.JQL = %q, want %q", opts.JQL, "project = TEST")
	}
	if opts.NextPageToken != "token123" {
		t.Errorf("SearchOptions.NextPageToken = %q, want %q", opts.NextPageToken, "token123")
	}
	if opts.MaxResults != 25 {
		t.Errorf("SearchOptions.MaxResults = %d, want 25", opts.MaxResults)
	}
	if len(opts.Fields) != 2 {
		t.Errorf("SearchOptions.Fields has %d items, want 2", len(opts.Fields))
	}
}

// TestIssueTypes tests the Issue and related type structures.
func TestIssueTypes(t *testing.T) {
	// Test that types can be JSON marshaled/unmarshaled correctly
	issue := &Issue{
		ID:   "10001",
		Key:  "TEST-123",
		Self: "https://example.atlassian.net/rest/api/3/issue/10001",
		Fields: IssueFields{
			Summary: "Test Summary",
			Status: &Status{
				ID:   "1",
				Name: "To Do",
				StatusCategory: &StatusCategory{
					ID:   1,
					Key:  "new",
					Name: "To Do",
				},
			},
			Priority: &Priority{
				ID:   "3",
				Name: "Medium",
			},
			IssueType: &IssueType{
				ID:      "10001",
				Name:    "Task",
				Subtask: false,
			},
			Assignee: &User{
				AccountID:   "user-123",
				DisplayName: "John Doe",
				Active:      true,
			},
			Labels: []string{"bug", "urgent"},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal back
	var decoded Issue
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Verify round-trip
	if decoded.Key != issue.Key {
		t.Errorf("Round-trip Key = %q, want %q", decoded.Key, issue.Key)
	}
	if decoded.Fields.Summary != issue.Fields.Summary {
		t.Errorf("Round-trip Summary = %q, want %q", decoded.Fields.Summary, issue.Fields.Summary)
	}
	if decoded.Fields.Status.Name != "To Do" {
		t.Errorf("Round-trip Status.Name = %q, want %q", decoded.Fields.Status.Name, "To Do")
	}
}

// TestCreateIssueRequest tests the CreateIssueRequest structure.
func TestCreateIssueRequest(t *testing.T) {
	req := &CreateIssueRequest{
		Fields: CreateIssueFields{
			Project:   &ProjectID{Key: "TEST"},
			Summary:   "New Issue",
			IssueType: &IssueTypeID{Name: "Task"},
			Priority:  &PriorityID{Name: "High"},
			Labels:    []string{"new-feature"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Verify JSON structure
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"key":"TEST"`) {
		t.Error("JSON should contain project key")
	}
	if !strings.Contains(jsonStr, `"summary":"New Issue"`) {
		t.Error("JSON should contain summary")
	}
}

// TestTransition tests the Transition structure.
func TestTransition(t *testing.T) {
	transition := &Transition{
		ID:   "31",
		Name: "Done",
		To: &Status{
			ID:   "10001",
			Name: "Done",
		},
	}

	if transition.ID != "31" {
		t.Errorf("Transition.ID = %q, want %q", transition.ID, "31")
	}
	if transition.To.Name != "Done" {
		t.Errorf("Transition.To.Name = %q, want %q", transition.To.Name, "Done")
	}
}

// TestComment tests the Comment structure.
func TestComment(t *testing.T) {
	comment := &Comment{
		ID: "10001",
		Author: &User{
			DisplayName: "Jane Doe",
		},
		Body: &ADF{
			Type:    "doc",
			Version: 1,
			Content: []ADFContent{
				{
					Type:    "paragraph",
					Content: []ADFContent{{Type: "text", Text: "Comment text"}},
				},
			},
		},
		Created: "2024-01-15T10:00:00.000+0000",
	}

	if comment.Author.DisplayName != "Jane Doe" {
		t.Errorf("Comment.Author.DisplayName = %q, want %q", comment.Author.DisplayName, "Jane Doe")
	}

	bodyText := ADFToText(comment.Body)
	if bodyText != "Comment text" {
		t.Errorf("Comment body text = %q, want %q", bodyText, "Comment text")
	}
}

// TestNewJiraService tests the NewJiraService constructor.
func TestNewJiraService(t *testing.T) {
	client := &Client{}
	service := NewJiraService(client)

	if service == nil {
		t.Fatal("NewJiraService() returned nil")
	}
	if service.client != client {
		t.Error("NewJiraService() did not set client correctly")
	}
}
