package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/enthus-appdev/atl-cli/internal/auth"
)

func newTestAssetsClient(server *httptest.Server, workspaceID string) *AssetsClient {
	client := &Client{
		httpClient: server.Client(),
		hostname:   "test.atlassian.net",
		cloudID:    "cloud-123",
		tokens: &auth.TokenSet{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}
	return &AssetsClient{
		client:      client,
		workspaceID: workspaceID,
		baseURL:     server.URL + "/ex/jira/cloud-123",
	}
}

func requireBearer(t *testing.T, request *http.Request) {
	t.Helper()
	if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Fatalf("Authorization = %q, want Bearer test-token", got)
	}
}

func TestNewAssetsClientUsesCloudGateway(t *testing.T) {
	client := &Client{cloudID: "cloud-123"}
	assets := NewAssetsClient(client, "workspace-456")

	if got, want := assets.baseURL, "https://api.atlassian.com/ex/jira/cloud-123"; got != want {
		t.Fatalf("baseURL = %q, want %q", got, want)
	}
}

func TestAssetsWorkspaceIDUsesOAuthGateway(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requireBearer(t, request)
		if request.URL.Path != "/ex/jira/cloud-123/rest/servicedeskapi/assets/workspace" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values":[{"workspaceId":"workspace-456"}]}`))
	}))
	defer server.Close()

	client := newTestAssetsClient(server, "")
	workspaceID, err := client.WorkspaceID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if workspaceID != "workspace-456" {
		t.Fatalf("WorkspaceID = %q, want workspace-456", workspaceID)
	}
}

func TestAssetsV1EscapesWorkspaceID(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client := newTestAssetsClient(server, "workspace/456")
	base, err := client.v1(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := base, server.URL+"/ex/jira/cloud-123/jsm/assets/workspace/workspace%2F456/v1"; got != want {
		t.Fatalf("v1() = %q, want %q", got, want)
	}
}

func TestAssetsAQLPageUsesOAuthGateway(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requireBearer(t, request)
		if request.URL.Path != "/ex/jira/cloud-123/jsm/assets/workspace/workspace-456/v1/object/aql" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		wantQuery := url.Values{"includeAttributes": {"false"}, "maxResults": {"25"}, "startAt": {"5"}}
		if request.URL.Query().Encode() != wantQuery.Encode() {
			t.Fatalf("query = %q, want %q", request.URL.Query().Encode(), wantQuery.Encode())
		}
		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["qlQuery"] != "objectId > 0" {
			t.Fatalf("qlQuery = %q", body["qlQuery"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values":[{"id":"9244","objectKey":"CUS-9244","label":"Customer"}],"isLast":true}`))
	}))
	defer server.Close()

	client := newTestAssetsClient(server, "workspace-456")
	objects, isLast, err := client.AQLPage(context.Background(), "objectId > 0", 5, 25)
	if err != nil {
		t.Fatal(err)
	}
	if !isLast || len(objects) != 1 || objects[0].ID != "9244" {
		t.Fatalf("objects = %#v, isLast = %v", objects, isLast)
	}
}

func TestAssetsObjectIncludesAttributes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requireBearer(t, request)
		if request.URL.Path != "/ex/jira/cloud-123/jsm/assets/workspace/workspace-456/v1/object/9244" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"9244",
			"objectKey":"CUS-9244",
			"label":"Example customer",
			"attributes":[{
				"objectTypeAttributeId":"77",
				"objectTypeAttribute":{"name":"MTS CustomerID"},
				"objectAttributeValues":[{"value":"145166","displayValue":"145166"}]
			}]
		}`))
	}))
	defer server.Close()

	client := newTestAssetsClient(server, "workspace-456")
	object, err := client.Object(context.Background(), "9244")
	if err != nil {
		t.Fatal(err)
	}
	if len(object.Attributes) != 1 || object.Attributes[0].ObjectTypeAttribute.Name != "MTS CustomerID" {
		t.Fatalf("attributes = %#v", object.Attributes)
	}
	if got := object.Attributes[0].ObjectAttributeValues[0].DisplayValue; got != "145166" {
		t.Fatalf("display value = %q, want 145166", got)
	}
}

func TestAssetsObjectRejectsWorkspaceMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"9244","workspaceId":"other-workspace"}`))
	}))
	defer server.Close()

	client := newTestAssetsClient(server, "workspace-456")
	if _, err := client.Object(context.Background(), "9244"); err == nil {
		t.Fatal("Object() succeeded for a different workspace")
	}
}
