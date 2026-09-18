package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
			Scopes: []string{
				auth.AssetsObjectReadScope,
				auth.AssetsSchemaReadScope,
				auth.AssetsTypeReadScope,
				auth.AssetsAttributeReadScope,
			},
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

func TestAssetsScopesAreOperationSpecific(t *testing.T) {
	client := &AssetsClient{client: &Client{
		hostname: "test.atlassian.net",
		tokens:   &auth.TokenSet{Scopes: []string{auth.AssetsObjectReadScope}},
	}}
	if err := client.requireScopes(auth.AssetsObjectReadScope); err != nil {
		t.Fatalf("object scope rejected: %v", err)
	}
	if err := client.requireScopes(auth.AssetsSchemaReadScope); err == nil {
		t.Fatal("missing schema scope accepted")
	}
}

func TestAssetAttributeValuePreservesLargeNumber(t *testing.T) {
	var value AssetAttributeValue
	if err := json.Unmarshal([]byte(`{"value":9007199254740993}`), &value); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"value":9007199254740993}`; got != want {
		t.Fatalf("round trip = %s, want %s", got, want)
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

func TestAssetsObjectTypeAttributes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requireBearer(t, request)
		if request.URL.Path != "/ex/jira/cloud-123/jsm/assets/workspace/workspace-456/v1/objecttype/9/attributes" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"550","name":"Import-Key","label":false,"type":0,
			 "defaultType":{"id":0,"name":"Text"},"system":true,"editable":false,
			 "minimumCardinality":1,"maximumCardinality":1,"position":0},
			{"id":"561","name":"Status","label":false,"type":0,
			 "defaultType":{"id":0,"name":"Text"},"editable":true,
			 "minimumCardinality":0,"maximumCardinality":1,"position":3}
		]`))
	}))
	defer server.Close()

	client := newTestAssetsClient(server, "workspace-456")
	attributes, err := client.ObjectTypeAttributes(context.Background(), "9")
	if err != nil {
		t.Fatal(err)
	}
	if len(attributes) != 2 {
		t.Fatalf("attributes = %#v", attributes)
	}
	if got, want := attributes[1].Name, "Status"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if !attributes[0].Required() {
		t.Error("minimumCardinality 1 did not read as required")
	}
	if attributes[1].Required() {
		t.Error("minimumCardinality 0 read as required")
	}
}

func TestAssetsObjectType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requireBearer(t, request)
		if request.URL.Path != "/ex/jira/cloud-123/jsm/assets/workspace/workspace-456/v1/objecttype/9" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"9","name":"Mitarbeiter","objectSchemaId":"5"}`))
	}))
	defer server.Close()

	client := newTestAssetsClient(server, "workspace-456")
	objectType, err := client.ObjectType(context.Background(), "9")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := objectType.Name, "Mitarbeiter"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
}

// TestAssetsObjectTypeReadsNeedTypeAndAttributeScopes pins which scope each
// object-type read is gated on, so a token carrying only the object and schema
// scopes fails locally with the missing scope named rather than as an opaque
// 401 "scope does not match" from Atlassian.
func TestAssetsObjectTypeReadsNeedTypeAndAttributeScopes(t *testing.T) {
	client := &AssetsClient{client: &Client{
		hostname: "test.atlassian.net",
		tokens:   &auth.TokenSet{Scopes: []string{auth.AssetsObjectReadScope, auth.AssetsSchemaReadScope}},
	}}
	if _, err := client.ObjectType(context.Background(), "9"); err == nil {
		t.Errorf("ObjectType() succeeded without %s", auth.AssetsTypeReadScope)
	}
	if _, err := client.ObjectTypeAttributes(context.Background(), "9"); err == nil {
		t.Errorf("ObjectTypeAttributes() succeeded without %s", auth.AssetsAttributeReadScope)
	}
}

// TestRequireObjectTypeReadScopesNamesEveryGap is the contract of the combined
// pre-check: one message lists every missing scope, so a caller does not fix one
// gap only to hit the next on the following run.
func TestRequireObjectTypeReadScopesNamesEveryGap(t *testing.T) {
	client := &AssetsClient{client: &Client{
		hostname: "test.atlassian.net",
		tokens:   &auth.TokenSet{Scopes: []string{auth.AssetsObjectReadScope, auth.AssetsSchemaReadScope}},
	}}

	err := client.RequireObjectTypeReadScopes()
	if err == nil {
		t.Fatal("RequireObjectTypeReadScopes() succeeded without the object type scopes")
	}
	for _, scope := range AssetsObjectTypeReadScopes {
		if !strings.Contains(err.Error(), scope) {
			t.Errorf("error %q does not name the missing scope %q", err.Error(), scope)
		}
	}

	granted := &AssetsClient{client: &Client{
		hostname: "test.atlassian.net",
		tokens:   &auth.TokenSet{Scopes: AssetsObjectTypeReadScopes},
	}}
	if err := granted.RequireObjectTypeReadScopes(); err != nil {
		t.Fatalf("RequireObjectTypeReadScopes() rejected a token holding both scopes: %v", err)
	}
}
