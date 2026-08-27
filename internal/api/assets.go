package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
		baseURL:     fmt.Sprintf("%s/ex/jira/%s", AtlassianAPIURL, client.CloudID()),
	}
}

func (c *AssetsClient) do(ctx context.Context, method, fullURL string, body, out interface{}) error {
	return c.client.Request(ctx, method, fullURL, body, out)
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
	Value        interface{} `json:"value,omitempty"`
	DisplayValue string      `json:"displayValue,omitempty"`
	SearchValue  string      `json:"searchValue,omitempty"`
}

type aqlPage struct {
	Values []AssetObject `json:"values"`
	IsLast bool          `json:"isLast"`
}

// AQLPage runs an AQL query and returns one page of objects.
func (c *AssetsClient) AQLPage(ctx context.Context, ql string, startAt, maxResults int) ([]AssetObject, bool, error) {
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
