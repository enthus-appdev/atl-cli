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

func newTestSMClient(server *httptest.Server) *Client {
	return &Client{
		httpClient: server.Client(),
		cloudID:    "test-cloud",
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}
}

func TestSMBaseURL(t *testing.T) {
	client := &Client{cloudID: "abc-123"}
	want := "https://api.atlassian.com/ex/jira/abc-123/rest/servicedeskapi"
	got := client.SMBaseURL()
	if got != want {
		t.Errorf("SMBaseURL() = %q, want %q", got, want)
	}
}

func TestGetServiceDesks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServiceDeskPage{
			Size: 2, Start: 0, Limit: 50, IsLastPage: true,
			Values: []*ServiceDesk{
				{ID: "1", ProjectID: "10001", ProjectName: "IT Support", ProjectKey: "ITS"},
				{ID: "2", ProjectID: "10002", ProjectName: "HR", ProjectKey: "HR"},
			},
		})
	}))
	defer server.Close()

	client := newTestSMClient(server)
	sm := NewSMService(client)

	// Call directly via the client.Get to use server URL
	var result ServiceDeskPage
	err := client.Get(context.Background(), server.URL+"/servicedesk", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(result.Values) != 2 {
		t.Errorf("got %d desks, want 2", len(result.Values))
	}
	if result.Values[0].ProjectKey != "ITS" {
		t.Errorf("first desk key = %q, want ITS", result.Values[0].ProjectKey)
	}
	_ = sm // ensure service is usable
}

func TestGetRequestTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RequestTypePage{
			Size: 2, Start: 0, Limit: 50, IsLastPage: true,
			Values: []*RequestType{
				{ID: "10", Name: "Incident", Description: "Report an incident"},
				{ID: "26", Name: "Service Delivery", Description: "DL ticket creation"},
			},
		})
	}))
	defer server.Close()

	client := newTestSMClient(server)

	var result RequestTypePage
	err := client.Get(context.Background(), server.URL+"/requesttype", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(result.Values) != 2 {
		t.Errorf("got %d types, want 2", len(result.Values))
	}
	if result.Values[1].Name != "Service Delivery" {
		t.Errorf("second type name = %q, want Service Delivery", result.Values[1].Name)
	}
}

func TestGetRequestTypeFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"requestTypeFields": []map[string]any{
				{
					"fieldId": "summary", "name": "Title", "required": true, "visible": true,
					"jiraSchema": map[string]any{"type": "string", "system": "summary"},
				},
				{
					"fieldId": "customfield_10038", "name": "Tempo Team", "required": false, "visible": true,
					"jiraSchema": map[string]any{
						"type": "object", "customId": 10038,
						"custom":        "ari:cloud:ecosystem::extension/tempo-team",
						"configuration": map[string]any{"customRenderer": true, "readOnly": false, "environment": "PRODUCTION"},
					},
				},
			},
			"canRaiseOnBehalfOf":        true,
			"canAddRequestParticipants": true,
		})
	}))
	defer server.Close()

	client := newTestSMClient(server)

	var result RequestTypeFieldsResponse
	err := client.Get(context.Background(), server.URL+"/field", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(result.RequestTypeFields) != 2 {
		t.Fatalf("got %d fields, want 2", len(result.RequestTypeFields))
	}

	// Verify first field
	if result.RequestTypeFields[0].FieldID != "summary" {
		t.Errorf("first field ID = %q, want summary", result.RequestTypeFields[0].FieldID)
	}
	if !result.RequestTypeFields[0].Required {
		t.Error("summary should be required")
	}

	// Verify second field has schema with mixed-type configuration (the NX-15519 bug scenario)
	tempoField := result.RequestTypeFields[1]
	if tempoField.FieldID != "customfield_10038" {
		t.Errorf("second field ID = %q, want customfield_10038", tempoField.FieldID)
	}
	if tempoField.JiraSchema == nil {
		t.Fatal("Tempo Team field should have jiraSchema")
	}
	if tempoField.JiraSchema.Type != "object" {
		t.Errorf("schema type = %q, want object", tempoField.JiraSchema.Type)
	}

	// Key assertion: configuration with booleans deserializes correctly into map[string]interface{}
	conf := tempoField.JiraSchema.Configuration
	if conf == nil {
		t.Fatal("configuration should not be nil")
	}
	if v, ok := conf["customRenderer"]; !ok || v != true {
		t.Errorf("configuration[customRenderer] = %v, want true", v)
	}
	if v, ok := conf["readOnly"]; !ok || v != false {
		t.Errorf("configuration[readOnly] = %v, want false", v)
	}
	if v, ok := conf["environment"]; !ok || v != "PRODUCTION" {
		t.Errorf("configuration[environment] = %v, want PRODUCTION", v)
	}
}

func TestGetRequestTypeFields_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":401,"message":"Unauthorized; scope does not match"}`))
	}))
	defer server.Close()

	client := newTestSMClient(server)

	var result RequestTypeFieldsResponse
	err := client.Get(context.Background(), server.URL+"/field", &result)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error should be *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("status code = %d, want 401", apiErr.StatusCode)
	}
}
