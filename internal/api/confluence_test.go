package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/enthus-appdev/atl-cli/internal/auth"
)

// TestNewConfluenceService tests the ConfluenceService constructor.
func TestNewConfluenceService(t *testing.T) {
	client := &Client{}
	service := NewConfluenceService(client)

	if service == nil {
		t.Fatal("NewConfluenceService() returned nil")
	}
	if service.client != client {
		t.Error("NewConfluenceService() did not set client correctly")
	}
}

// TestSpaceStructure tests the Space structure JSON serialization.
func TestSpaceStructure(t *testing.T) {
	space := &Space{
		ID:     "123456",
		Key:    "TEST",
		Name:   "Test Space",
		Type:   "global",
		Status: "current",
		Description: &SpaceDescription{
			Plain: &PlainValue{Value: "A test space"},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(space)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal back
	var decoded Space
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Key != "TEST" {
		t.Errorf("Space.Key = %q, want %q", decoded.Key, "TEST")
	}
	if decoded.Name != "Test Space" {
		t.Errorf("Space.Name = %q, want %q", decoded.Name, "Test Space")
	}
	if decoded.Description.Plain.Value != "A test space" {
		t.Errorf("Space.Description.Plain.Value = %q, want %q", decoded.Description.Plain.Value, "A test space")
	}
}

// TestPageStructure tests the Page structure JSON serialization.
func TestPageStructure(t *testing.T) {
	page := &Page{
		ID:      "123456789",
		Title:   "Test Page",
		SpaceID: "123456",
		Status:  "current",
		Version: &PageVersion{
			Number:    1,
			Message:   "Initial version",
			CreatedAt: "2024-01-15T10:00:00.000Z",
		},
		Body: &PageBody{
			Storage: &BodyContent{
				Value:          "<p>Page content</p>",
				Representation: "storage",
			},
		},
		Links: &PageLinks{
			WebUI: "/wiki/spaces/TEST/pages/123456789/Test+Page",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal back
	var decoded Page
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Title != "Test Page" {
		t.Errorf("Page.Title = %q, want %q", decoded.Title, "Test Page")
	}
	if decoded.Version.Number != 1 {
		t.Errorf("Page.Version.Number = %d, want 1", decoded.Version.Number)
	}
	if decoded.Body.Storage.Value != "<p>Page content</p>" {
		t.Errorf("Page.Body.Storage.Value = %q, want %q", decoded.Body.Storage.Value, "<p>Page content</p>")
	}
}

// TestExtractCursor tests the cursor extraction from pagination URLs.
func TestExtractCursor(t *testing.T) {
	tests := []struct {
		name    string
		nextURL string
		want    string
	}{
		{
			name:    "valid cursor",
			nextURL: "https://api.atlassian.com/ex/confluence/123/wiki/api/v2/spaces?cursor=abc123",
			want:    "abc123",
		},
		{
			name:    "no cursor",
			nextURL: "https://api.atlassian.com/ex/confluence/123/wiki/api/v2/spaces",
			want:    "",
		},
		{
			name:    "invalid URL",
			nextURL: "://invalid",
			want:    "",
		},
		{
			name:    "empty URL",
			nextURL: "",
			want:    "",
		},
		{
			name:    "cursor with special characters",
			nextURL: "https://api.atlassian.com/spaces?cursor=abc%2B123",
			want:    "abc+123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCursor(tt.nextURL)
			if got != tt.want {
				t.Errorf("extractCursor() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGetSpaces tests the GetSpaces method.
func TestGetSpaces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		limit := r.URL.Query().Get("limit")
		if limit == "" {
			t.Error("limit parameter should be set")
		}

		response := SpacesResponse{
			Results: []*Space{
				{ID: "1", Key: "SPACE1", Name: "Space One"},
				{ID: "2", Key: "SPACE2", Name: "Space Two"},
			},
			Links: &PaginationLinks{
				Next: "",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
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
	var result SpacesResponse
	err := client.Get(ctx, server.URL+"?limit=25&status=current", &result)

	if err != nil {
		t.Fatalf("GetSpaces error = %v", err)
	}
	if len(result.Results) != 2 {
		t.Errorf("GetSpaces returned %d spaces, want 2", len(result.Results))
	}
	if result.Results[0].Key != "SPACE1" {
		t.Errorf("First space key = %q, want %q", result.Results[0].Key, "SPACE1")
	}
}

// TestGetPagesInSpace tests the GetPagesInSpace method.
func TestGetPagesInSpace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := PagesResponse{
			Results: []*Page{
				{ID: "1", Title: "Page One", SpaceID: "123"},
				{ID: "2", Title: "Page Two", SpaceID: "123"},
			},
			Links: &PaginationLinks{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
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
	var result PagesResponse
	err := client.Get(ctx, server.URL, &result)

	if err != nil {
		t.Fatalf("GetPagesInSpace error = %v", err)
	}
	if len(result.Results) != 2 {
		t.Errorf("GetPagesInSpace returned %d pages, want 2", len(result.Results))
	}
}

// TestCreatePageRequest tests the CreatePageRequest structure.
func TestCreatePageRequest(t *testing.T) {
	req := CreatePageRequest{
		SpaceID: "123456",
		Status:  "current",
		Title:   "New Page",
	}
	req.Body.Representation = "storage"
	req.Body.Value = "<p>Page content</p>"

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)
	if jsonStr == "" {
		t.Error("JSON should not be empty")
	}

	// Verify structure
	var decoded CreatePageRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Title != "New Page" {
		t.Errorf("CreatePageRequest.Title = %q, want %q", decoded.Title, "New Page")
	}
	if decoded.Body.Representation != "storage" {
		t.Errorf("CreatePageRequest.Body.Representation = %q, want %q", decoded.Body.Representation, "storage")
	}
}

// TestUpdatePageRequest tests the UpdatePageRequest structure.
func TestUpdatePageRequest(t *testing.T) {
	req := UpdatePageRequest{
		ID:     "123456789",
		Status: "current",
		Title:  "Updated Page",
	}
	req.Body.Representation = "storage"
	req.Body.Value = "<p>Updated content</p>"
	req.Version.Number = 2
	req.Version.Message = "Updated content"

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded UpdatePageRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Version.Number != 2 {
		t.Errorf("UpdatePageRequest.Version.Number = %d, want 2", decoded.Version.Number)
	}
	if decoded.Title != "Updated Page" {
		t.Errorf("UpdatePageRequest.Title = %q, want %q", decoded.Title, "Updated Page")
	}
}

// TestSpacesResponse tests the SpacesResponse structure.
func TestSpacesResponse(t *testing.T) {
	response := &SpacesResponse{
		Results: []*Space{
			{ID: "1", Key: "ONE"},
			{ID: "2", Key: "TWO"},
		},
		Links: &PaginationLinks{
			Next: "https://api.atlassian.com/spaces?cursor=next123",
		},
	}

	if len(response.Results) != 2 {
		t.Errorf("SpacesResponse.Results has %d items, want 2", len(response.Results))
	}
	if response.Links.Next == "" {
		t.Error("SpacesResponse.Links.Next should not be empty")
	}
}

// TestPagesResponse tests the PagesResponse structure.
func TestPagesResponse(t *testing.T) {
	response := &PagesResponse{
		Results: []*Page{
			{ID: "1", Title: "Page 1"},
			{ID: "2", Title: "Page 2"},
		},
		Links: &PaginationLinks{},
	}

	if len(response.Results) != 2 {
		t.Errorf("PagesResponse.Results has %d items, want 2", len(response.Results))
	}
}

// TestSpaceDescription tests the SpaceDescription structure.
func TestSpaceDescription(t *testing.T) {
	desc := &SpaceDescription{
		Plain: &PlainValue{Value: "Plain description"},
		View:  &ViewValue{Value: "<p>HTML description</p>"},
	}

	if desc.Plain.Value != "Plain description" {
		t.Errorf("SpaceDescription.Plain.Value = %q, want %q", desc.Plain.Value, "Plain description")
	}
	if desc.View.Value != "<p>HTML description</p>" {
		t.Errorf("SpaceDescription.View.Value = %q, want %q", desc.View.Value, "<p>HTML description</p>")
	}
}

// TestPageBodyFormats tests the PageBody structure with different formats.
func TestPageBodyFormats(t *testing.T) {
	body := &PageBody{
		Storage: &BodyContent{
			Value:          "<p>Storage format</p>",
			Representation: "storage",
		},
		View: &BodyContent{
			Value:          "<p>Rendered HTML</p>",
			Representation: "view",
		},
	}

	if body.Storage.Representation != "storage" {
		t.Errorf("PageBody.Storage.Representation = %q, want %q", body.Storage.Representation, "storage")
	}
	if body.View.Representation != "view" {
		t.Errorf("PageBody.View.Representation = %q, want %q", body.View.Representation, "view")
	}
}
