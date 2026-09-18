package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/enthus-appdev/atl-cli/internal/auth"
)

// AssetsClient talks to the Jira Service Management Assets (CMDB) REST API.
//
// Assets lives below a different path than the Jira platform API, but supports
// the same OAuth 2.0 access token when requests use the cloud gateway URL.
type AssetsClient struct {
	client      *Client
	workspaceID string
	baseURL     string
}

// NewAssetsClient builds an OAuth-backed Assets client. workspaceID may be
// empty, in which case it is discovered through Jira Service Management.
func NewAssetsClient(client *Client, workspaceID string) *AssetsClient {
	return &AssetsClient{
		client:      client,
		workspaceID: workspaceID,
		baseURL:     client.JiraGatewayBaseURL(),
	}
}

func (c *AssetsClient) do(ctx context.Context, method, fullURL string, body, out interface{}) error {
	return c.client.Request(ctx, method, fullURL, body, out)
}

func (c *AssetsClient) requireScopes(required ...string) error {
	missing := c.client.MissingScopes(required...)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("OAuth token is missing Assets scopes %s; re-authenticate with 'atl auth login --hostname %s'",
		strings.Join(missing, ", "), c.client.Hostname())
}

// WorkspaceID returns the resolved workspace id, discovering it from the site if
// it was not supplied.
func (c *AssetsClient) WorkspaceID(ctx context.Context) (string, error) {
	if c.workspaceID != "" {
		return c.workspaceID, nil
	}
	var out struct {
		Values []struct {
			WorkspaceID string `json:"workspaceId"`
		} `json:"values"`
	}
	if err := c.do(ctx, http.MethodGet, c.baseURL+"/rest/servicedeskapi/assets/workspace", nil, &out); err != nil {
		return "", fmt.Errorf("discovering assets workspace: %w", err)
	}
	if len(out.Values) == 0 {
		return "", fmt.Errorf("no assets workspace found for %s", c.client.Hostname())
	}
	c.workspaceID = out.Values[0].WorkspaceID
	return c.workspaceID, nil
}

func (c *AssetsClient) v1(ctx context.Context) (string, error) {
	ws, err := c.WorkspaceID(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/jsm/assets/workspace/%s/v1", c.baseURL, url.PathEscape(ws)), nil
}

// AssetSchema is one object schema with its current object count.
type AssetSchema struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ObjectSchemaKey string `json:"objectSchemaKey"`
	ObjectCount     int    `json:"objectCount"`
	ObjectTypeCount int    `json:"objectTypeCount"`
}

// Schemas returns all object schemas in the workspace.
func (c *AssetsClient) Schemas(ctx context.Context) ([]AssetSchema, error) {
	if err := c.requireScopes(auth.AssetsSchemaReadScope); err != nil {
		return nil, err
	}
	base, err := c.v1(ctx)
	if err != nil {
		return nil, err
	}
	var out struct {
		Values []AssetSchema `json:"values"`
	}
	if err := c.do(ctx, http.MethodGet, base+"/objectschema/list", nil, &out); err != nil {
		return nil, err
	}
	return out.Values, nil
}

// AssetObject is a single Assets object (trimmed to the useful fields).
type AssetObject struct {
	WorkspaceID string `json:"workspaceId,omitempty"`
	GlobalID    string `json:"globalId,omitempty"`
	ID          string `json:"id"`
	ObjectKey   string `json:"objectKey"`
	Label       string `json:"label"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
	ObjectType  struct {
		ID   string `json:"id,omitempty"`
		Name string `json:"name"`
	} `json:"objectType"`
	Attributes []AssetAttribute `json:"attributes,omitempty"`
}

// AssetAttribute is one named attribute and its values on an Assets object.
type AssetAttribute struct {
	ID                    string `json:"id,omitempty"`
	ObjectTypeAttributeID string `json:"objectTypeAttributeId"`
	ObjectTypeAttribute   struct {
		Name string `json:"name"`
	} `json:"objectTypeAttribute"`
	ObjectAttributeValues []AssetAttributeValue `json:"objectAttributeValues,omitempty"`
}

// AssetAttributeValue preserves both the API value and its display form.
type AssetAttributeValue struct {
	Value        json.RawMessage `json:"value,omitempty"`
	DisplayValue string          `json:"displayValue,omitempty"`
	SearchValue  string          `json:"searchValue,omitempty"`
}

type aqlPage struct {
	Values []AssetObject `json:"values"`
	IsLast bool          `json:"isLast"`
}

// AQLPage runs an AQL query and returns one page of objects.
func (c *AssetsClient) AQLPage(ctx context.Context, ql string, startAt, maxResults int) ([]AssetObject, bool, error) {
	if err := c.requireScopes(auth.AssetsObjectReadScope); err != nil {
		return nil, false, err
	}
	base, err := c.v1(ctx)
	if err != nil {
		return nil, false, err
	}
	q := url.Values{}
	q.Set("startAt", fmt.Sprint(startAt))
	q.Set("maxResults", fmt.Sprint(maxResults))
	q.Set("includeAttributes", "false")
	var page aqlPage
	if err := c.do(ctx, http.MethodPost, base+"/object/aql?"+q.Encode(), map[string]string{"qlQuery": ql}, &page); err != nil {
		return nil, false, err
	}
	return page.Values, page.IsLast, nil
}

// Object loads an Assets object and all attributes returned by the API.
func (c *AssetsClient) Object(ctx context.Context, objectID string) (*AssetObject, error) {
	if err := c.requireScopes(auth.AssetsObjectReadScope); err != nil {
		return nil, err
	}
	workspaceID, err := c.WorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	base, err := c.v1(ctx)
	if err != nil {
		return nil, err
	}
	var object AssetObject
	if err := c.do(ctx, http.MethodGet, base+"/object/"+url.PathEscape(objectID), nil, &object); err != nil {
		return nil, err
	}
	if object.WorkspaceID != "" && object.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("assets object %s belongs to workspace %s, expected %s", objectID, object.WorkspaceID, workspaceID)
	}
	return &object, nil
}

// AQLCount returns the exact number of objects matching an AQL query by
// paginating through every page. The object/aql endpoint caps its reported
// `total` at 1000, so it cannot be trusted for counting — this walks instead.
func (c *AssetsClient) AQLCount(ctx context.Context, ql string) (int, error) {
	total, start := 0, 0
	for {
		vals, isLast, err := c.AQLPage(ctx, ql, start, 500)
		if err != nil {
			return 0, err
		}
		total += len(vals)
		if isLast || len(vals) == 0 {
			return total, nil
		}
		start += len(vals)
	}
}

// AssetObjectType is one object type (the "class" an Assets object belongs to).
type AssetObjectType struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	ObjectSchemaID     string `json:"objectSchemaId,omitempty"`
	ParentObjectTypeID string `json:"parentObjectTypeId,omitempty"`
	ObjectCount        int    `json:"objectCount,omitempty"`
	Inherited          bool   `json:"inherited,omitempty"`
}

// AssetObjectTypeAttribute is one attribute definition on an object type: the
// field itself, independent of any object's value for it. An object omits every
// attribute it holds no value for, so reading one object can never prove an
// attribute absent from its type; this is what does.
type AssetObjectTypeAttribute struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Label       bool   `json:"label,omitempty"`
	Type        int    `json:"type"`
	DefaultType struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"defaultType"`
	// Options carries an attribute's predefined value list as one comma-separated
	// string, for the attributes that have one. It is not confined to Select: a
	// Text attribute can carry a list too, so the presence of options says what
	// values are expected without implying the kind.
	Options            string `json:"options,omitempty"`
	System             bool   `json:"system,omitempty"`
	Editable           bool   `json:"editable,omitempty"`
	Hidden             bool   `json:"hidden,omitempty"`
	UniqueAttribute    bool   `json:"uniqueAttribute,omitempty"`
	MinimumCardinality int    `json:"minimumCardinality"`
	MaximumCardinality int    `json:"maximumCardinality"`
	Position           int    `json:"position"`
}

// TypeName names the attribute's kind for display. Only a Default attribute
// (type 0) carries its concrete kind in DefaultType; a reference, user, group,
// or project attribute leaves DefaultType empty and is identified by the numeric
// type alone. Assets does not publish that enum, so an unnamed kind is reported
// as its number rather than mapped to a label that cannot be verified.
func (a AssetObjectTypeAttribute) TypeName() string {
	if a.DefaultType.Name != "" {
		return a.DefaultType.Name
	}
	return fmt.Sprintf("type %d", a.Type)
}

// Required reports whether the attribute must carry at least one value.
func (a AssetObjectTypeAttribute) Required() bool {
	return a.MinimumCardinality > 0
}

// AssetsObjectTypeReadScopes are the scopes the two object type reads need
// between them. Assets gates the type and its attributes separately, so a
// caller that will make both requests checks both up front rather than
// discovering the second gap only after fixing the first.
var AssetsObjectTypeReadScopes = []string{auth.AssetsTypeReadScope, auth.AssetsAttributeReadScope}

// RequireObjectTypeReadScopes reports every scope missing for reading an object
// type together with its attributes.
func (c *AssetsClient) RequireObjectTypeReadScopes() error {
	return c.requireScopes(AssetsObjectTypeReadScopes...)
}

// ObjectType loads one object type by id. Its name is what tells a caller the id
// addresses the type they meant: a wrong id and a type without the attribute
// being looked for are otherwise indistinguishable.
func (c *AssetsClient) ObjectType(ctx context.Context, objectTypeID string) (*AssetObjectType, error) {
	if err := c.requireScopes(auth.AssetsTypeReadScope); err != nil {
		return nil, err
	}
	base, err := c.v1(ctx)
	if err != nil {
		return nil, err
	}
	var objectType AssetObjectType
	if err := c.do(ctx, http.MethodGet, base+"/objecttype/"+url.PathEscape(objectTypeID), nil, &objectType); err != nil {
		return nil, err
	}
	return &objectType, nil
}

// ObjectTypeAttributes returns every attribute defined on an object type,
// including the ones inherited from a parent type.
func (c *AssetsClient) ObjectTypeAttributes(ctx context.Context, objectTypeID string) ([]AssetObjectTypeAttribute, error) {
	if err := c.requireScopes(auth.AssetsAttributeReadScope); err != nil {
		return nil, err
	}
	base, err := c.v1(ctx)
	if err != nil {
		return nil, err
	}
	var attributes []AssetObjectTypeAttribute
	if err := c.do(ctx, http.MethodGet, base+"/objecttype/"+url.PathEscape(objectTypeID)+"/attributes", nil, &attributes); err != nil {
		return nil, err
	}
	return attributes, nil
}
