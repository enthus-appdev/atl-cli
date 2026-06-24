package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AssetsClient talks to the Jira Service Management Assets (CMDB) REST API.
//
// Assets lives on a different base URL than the Jira site API
// (https://api.atlassian.com/jsm/assets/workspace/{workspaceId}/v1) and the
// granular OAuth scopes the rest of atl uses do not cover CMDB objects, so this
// client authenticates with Basic auth (account email + API token) instead of
// the shared OAuth client. The token is read from the environment so it never
// lands in the on-disk config.
type AssetsClient struct {
	httpClient  *http.Client
	email       string
	token       string
	workspaceID string
	siteBase    string // https://<hostname>, used only for workspace discovery
}

const assetsAPIBase = "https://api.atlassian.com/jsm/assets/workspace"

// NewAssetsClient builds an Assets client. email and token are required;
// workspaceID may be empty, in which case it is discovered from the site.
func NewAssetsClient(siteBase, email, token, workspaceID string) *AssetsClient {
	return &AssetsClient{
		httpClient:  &http.Client{Timeout: 60 * time.Second},
		email:       email,
		token:       token,
		workspaceID: workspaceID,
		siteBase:    strings.TrimRight(siteBase, "/"),
	}
}

func (c *AssetsClient) do(ctx context.Context, method, fullURL string, body []byte, out interface{}) error {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, rdr)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		dec := json.NewDecoder(resp.Body)
		var apiErr struct {
			ErrorMessages []string `json:"errorMessages"`
		}
		_ = dec.Decode(&apiErr)
		if len(apiErr.ErrorMessages) > 0 {
			return fmt.Errorf("assets API %s: %d: %s", method, resp.StatusCode, strings.Join(apiErr.ErrorMessages, "; "))
		}
		return fmt.Errorf("assets API %s: unexpected status %d", method, resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// WorkspaceID returns the resolved workspace id, discovering it from the site if
// it was not supplied.
func (c *AssetsClient) WorkspaceID(ctx context.Context) (string, error) {
	if c.workspaceID != "" {
		return c.workspaceID, nil
	}
	if c.siteBase == "" {
		return "", fmt.Errorf("workspace id not set and no site to discover it from (pass --workspace or set ATLASSIAN_ASSETS_WORKSPACE)")
	}
	var out struct {
		Values []struct {
			WorkspaceID string `json:"workspaceId"`
		} `json:"values"`
	}
	if err := c.do(ctx, http.MethodGet, c.siteBase+"/rest/servicedeskapi/assets/workspace", nil, &out); err != nil {
		return "", fmt.Errorf("discovering assets workspace: %w", err)
	}
	if len(out.Values) == 0 {
		return "", fmt.Errorf("no assets workspace found for site %s", c.siteBase)
	}
	c.workspaceID = out.Values[0].WorkspaceID
	return c.workspaceID, nil
}

func (c *AssetsClient) v1(ctx context.Context) (string, error) {
	ws, err := c.WorkspaceID(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/v1", assetsAPIBase, ws), nil
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
	ID         string `json:"id"`
	ObjectKey  string `json:"objectKey"`
	Label      string `json:"label"`
	Created    string `json:"created"`
	Updated    string `json:"updated"`
	ObjectType struct {
		Name string `json:"name"`
	} `json:"objectType"`
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
	body, _ := json.Marshal(map[string]string{"qlQuery": ql})
	var page aqlPage
	if err := c.do(ctx, http.MethodPost, base+"/object/aql?"+q.Encode(), body, &page); err != nil {
		return nil, false, err
	}
	return page.Values, page.IsLast, nil
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
