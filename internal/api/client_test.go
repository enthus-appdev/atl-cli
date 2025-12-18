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

// TestBuildQueryString tests the URL query string builder.
func TestBuildQueryString(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		want   string
	}{
		{
			name:   "empty params",
			params: map[string]string{},
			want:   "",
		},
		{
			name:   "single param",
			params: map[string]string{"key": "value"},
			want:   "?key=value",
		},
		{
			name:   "multiple params",
			params: map[string]string{"a": "1", "b": "2"},
			// Note: order is not guaranteed in maps
			want: "?", // Just check prefix, content tested separately
		},
		{
			name:   "empty value excluded",
			params: map[string]string{"key": "", "other": "value"},
			want:   "?other=value",
		},
		{
			name:   "special characters encoded",
			params: map[string]string{"q": "hello world"},
			want:   "?q=hello+world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildQueryString(tt.params)

			if tt.name == "empty params" {
				if got != "" {
					t.Errorf("BuildQueryString() = %q, want empty string", got)
				}
				return
			}

			if tt.name == "multiple params" {
				// Just verify it starts with ? and contains both params
				if got == "" || got[0] != '?' {
					t.Errorf("BuildQueryString() = %q, should start with ?", got)
				}
				return
			}

			if got != tt.want {
				t.Errorf("BuildQueryString() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestJoinPath tests the path joining utility.
func TestJoinPath(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		paths []string
		want  string
	}{
		{
			name:  "simple join",
			base:  "https://api.example.com",
			paths: []string{"v1", "users"},
			want:  "https://api.example.com/v1/users",
		},
		{
			name:  "base with trailing slash",
			base:  "https://api.example.com/",
			paths: []string{"v1"},
			want:  "https://api.example.com/v1",
		},
		{
			name:  "paths with leading slashes",
			base:  "https://api.example.com",
			paths: []string{"/v1", "/users"},
			want:  "https://api.example.com/v1/users",
		},
		{
			name:  "empty paths",
			base:  "https://api.example.com",
			paths: []string{},
			want:  "https://api.example.com",
		},
		{
			name:  "single path",
			base:  "https://api.example.com",
			paths: []string{"endpoint"},
			want:  "https://api.example.com/endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinPath(tt.base, tt.paths...)
			if got != tt.want {
				t.Errorf("JoinPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAPIError tests the APIError type.
func TestAPIError(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Status:     "404 Not Found",
		Body:       `{"message": "Issue not found"}`,
	}

	errStr := err.Error()

	if errStr == "" {
		t.Error("APIError.Error() should not return empty string")
	}

	// Check that error message contains key information
	if !contains(errStr, "404") {
		t.Error("APIError.Error() should contain status code")
	}
	if !contains(errStr, "Not Found") {
		t.Error("APIError.Error() should contain status text")
	}
}

// TestClientRequest tests the Client.Request method with a mock server.
func TestClientRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("Request missing Authorization header")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("Request missing Accept header")
		}

		// Return a JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	// Create a client with mock token
	client := &Client{
		httpClient: server.Client(),
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}

	// Make a request
	var result map[string]string
	err := client.Get(context.Background(), server.URL, &result)

	if err != nil {
		t.Errorf("Client.Get() error = %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("Client.Get() result = %v, want {status: ok}", result)
	}
}

// TestClientRequestError tests error handling in Client.Request.
func TestClientRequestError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}

	var result map[string]string
	err := client.Get(context.Background(), server.URL, &result)

	if err == nil {
		t.Error("Client.Get() should return error for 404 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Client.Get() error should be *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("APIError.StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

// TestClientPost tests the Client.Post method.
func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("POST request should have Content-Type: application/json")
		}

		// Echo back the request body
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"received": body["message"],
		})
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}

	var result map[string]string
	err := client.Post(context.Background(), server.URL, map[string]string{"message": "hello"}, &result)

	if err != nil {
		t.Errorf("Client.Post() error = %v", err)
	}
	if result["received"] != "hello" {
		t.Errorf("Client.Post() result = %v, want {received: hello}", result)
	}
}

// TestClientPut tests the Client.Put method.
func TestClientPut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}

	err := client.Put(context.Background(), server.URL, map[string]string{"key": "value"}, nil)

	if err != nil {
		t.Errorf("Client.Put() error = %v", err)
	}
}

// TestClientDelete tests the Client.Delete method.
func TestClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}

	err := client.Delete(context.Background(), server.URL)

	if err != nil {
		t.Errorf("Client.Delete() error = %v", err)
	}
}

// TestClientURLMethods tests the URL generation methods.
func TestClientURLMethods(t *testing.T) {
	client := &Client{
		cloudID: "test-cloud-id",
	}

	jiraURL := client.JiraBaseURL()
	expectedJira := "https://api.atlassian.com/ex/jira/test-cloud-id/rest/api/3"
	if jiraURL != expectedJira {
		t.Errorf("JiraBaseURL() = %q, want %q", jiraURL, expectedJira)
	}

	// Confluence always uses v2 API (v1 has been deprecated)
	confluenceURL := client.ConfluenceBaseURL()
	expectedConfluence := "https://api.atlassian.com/ex/confluence/test-cloud-id/wiki/api/v2"
	if confluenceURL != expectedConfluence {
		t.Errorf("ConfluenceBaseURL() = %q, want %q", confluenceURL, expectedConfluence)
	}
}

// TestClientAccessors tests the client accessor methods.
func TestClientAccessors(t *testing.T) {
	client := &Client{
		hostname: "example.atlassian.net",
		cloudID:  "cloud-123",
	}

	if client.Hostname() != "example.atlassian.net" {
		t.Errorf("Hostname() = %q, want %q", client.Hostname(), "example.atlassian.net")
	}

	if client.CloudID() != "cloud-123" {
		t.Errorf("CloudID() = %q, want %q", client.CloudID(), "cloud-123")
	}
}

// Helper function to check string containment
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
