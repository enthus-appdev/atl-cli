package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

const (
	// ConfluenceMaxLimit is the maximum allowed limit for Confluence API pagination.
	// The API returns 400 if you exceed this.
	ConfluenceMaxLimit = 250
)

// capLimit ensures the limit doesn't exceed the API maximum.
// Returns the capped limit and logs a debug message if capping occurred.
func capLimit(limit, max int) int {
	if limit > max {
		debugLog("Limit %d exceeds max %d, capping", limit, max)
		return max
	}
	return limit
}

// ConfluenceService handles Confluence API operations.
//
// # API Version Strategy
//
// This service uses a mix of Confluence REST API v1 and v2:
//
//   - v2 API (/wiki/api/v2): Used for most operations (pages, spaces, children).
//     Better performance, cleaner response format, and actively developed.
//
//   - v1 API (/wiki/rest/api): Required for operations not yet in v2:
//   - Search (CQL): No v2 equivalent exists (as of Dec 2024)
//   - Archive/Unarchive: Only available in v1
//   - Move page: Only available in v1
//
// When Atlassian adds these endpoints to v2, we should migrate.
// Track progress: https://developer.atlassian.com/cloud/confluence/rest/v2/intro/
//
// # Required OAuth Scopes
//
// Classic scopes (for v1 operations):
//   - read:confluence-content.all
//   - write:confluence-content
//   - search:confluence (required for CQL search)
//
// Granular scopes (for v2 operations):
//   - read:page:confluence, write:page:confluence
//   - read:space:confluence
//   - read:content:confluence, write:content:confluence
type ConfluenceService struct {
	client *Client
}

// NewConfluenceService creates a new Confluence service.
func NewConfluenceService(client *Client) *ConfluenceService {
	return &ConfluenceService{client: client}
}

// Space represents a Confluence space.
type Space struct {
	ID          string            `json:"id"`
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description *SpaceDescription `json:"description,omitempty"`
	Status      string            `json:"status"`
	HomepageID  string            `json:"homepageId,omitempty"`
}

// SpaceDescription represents a space description.
type SpaceDescription struct {
	Plain *PlainValue `json:"plain,omitempty"`
	View  *ViewValue  `json:"view,omitempty"`
}

// PlainValue represents plain text content.
type PlainValue struct {
	Value string `json:"value"`
}

// ViewValue represents rendered content.
type ViewValue struct {
	Value string `json:"value"`
}

// Page represents a Confluence page.
type Page struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	SpaceID   string       `json:"spaceId,omitempty"`
	Status    string       `json:"status"`
	ParentID  string       `json:"parentId,omitempty"`
	AuthorID  string       `json:"authorId,omitempty"`
	CreatedAt string       `json:"createdAt,omitempty"`
	Version   *PageVersion `json:"version,omitempty"`
	Body      *PageBody    `json:"body,omitempty"`
	Links     *PageLinks   `json:"_links,omitempty"`
}

// PageVersion represents page version information.
type PageVersion struct {
	Number    int    `json:"number"`
	Message   string `json:"message,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	AuthorID  string `json:"authorId,omitempty"`
}

// PageBody represents the body content of a page.
type PageBody struct {
	Storage        *BodyContent `json:"storage,omitempty"`
	AtlasDocFormat *BodyContent `json:"atlas_doc_format,omitempty"`
	View           *BodyContent `json:"view,omitempty"`
}

// BodyContent represents body content in a specific format.
type BodyContent struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// PageLinks represents page links.
type PageLinks struct {
	WebUI  string `json:"webui,omitempty"`
	EditUI string `json:"editui,omitempty"`
	TinyUI string `json:"tinyui,omitempty"`
}

// SpacesResponse represents a paginated list of spaces.
type SpacesResponse struct {
	Results []*Space         `json:"results"`
	Links   *PaginationLinks `json:"_links,omitempty"`
}

// PagesResponse represents a paginated list of pages.
type PagesResponse struct {
	Results []*Page          `json:"results"`
	Links   *PaginationLinks `json:"_links,omitempty"`
}

// PaginationLinks represents pagination links.
type PaginationLinks struct {
	Next string `json:"next,omitempty"`
	Base string `json:"base,omitempty"`
}

// baseURL returns the base URL for Confluence v2 API.
func (s *ConfluenceService) baseURL() string {
	return s.client.ConfluenceBaseURLV2()
}

// GetSpaces gets a list of spaces.
func (s *ConfluenceService) GetSpaces(ctx context.Context, limit int, cursor string) (*SpacesResponse, error) {
	path := fmt.Sprintf("%s/spaces", s.baseURL())

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(capLimit(limit, ConfluenceMaxLimit)))
	}
	params.Set("status", "current")
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	var result SpacesResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetSpacesAll gets all spaces by following pagination.
func (s *ConfluenceService) GetSpacesAll(ctx context.Context) ([]*Space, error) {
	var allSpaces []*Space
	cursor := ""

	for {
		result, err := s.GetSpaces(ctx, 100, cursor)
		if err != nil {
			return nil, err
		}
		allSpaces = append(allSpaces, result.Results...)

		if result.Links == nil || result.Links.Next == "" {
			break
		}
		cursor = extractCursor(result.Links.Next)
		if cursor == "" {
			break
		}
	}

	return allSpaces, nil
}

// extractCursor extracts the cursor parameter from a pagination URL.
func extractCursor(nextURL string) string {
	parsed, err := url.Parse(nextURL)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("cursor")
}

// GetSpace gets a space by ID.
func (s *ConfluenceService) GetSpace(ctx context.Context, spaceID string) (*Space, error) {
	path := fmt.Sprintf("%s/spaces/%s", s.baseURL(), spaceID)

	var space Space
	if err := s.client.Get(ctx, path, &space); err != nil {
		return nil, err
	}

	return &space, nil
}

// GetSpaceByKey gets a space by its key.
func (s *ConfluenceService) GetSpaceByKey(ctx context.Context, key string) (*Space, error) {
	path := fmt.Sprintf("%s/spaces", s.baseURL())

	params := url.Values{}
	params.Set("keys", key)
	params.Set("limit", "1")

	var result SpacesResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("space not found: %s", key)
	}

	return result.Results[0], nil
}

// GetPages gets pages in a space.
// status can be: "current", "draft", "archived", or empty for current.
func (s *ConfluenceService) GetPages(ctx context.Context, spaceID string, limit int, cursor string, status string) (*PagesResponse, error) {
	path := fmt.Sprintf("%s/spaces/%s/pages", s.baseURL(), spaceID)

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(capLimit(limit, ConfluenceMaxLimit)))
	}
	if status != "" {
		params.Set("status", status)
	} else {
		params.Set("status", "current")
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	var result PagesResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetPagesAll gets all pages in a space.
// status can be "current", "draft", "archived", or empty for current.
func (s *ConfluenceService) GetPagesAll(ctx context.Context, spaceID string, status string) ([]*Page, error) {
	var allPages []*Page
	cursor := ""

	for {
		result, err := s.GetPages(ctx, spaceID, 100, cursor, status)
		if err != nil {
			return nil, err
		}
		allPages = append(allPages, result.Results...)

		if result.Links == nil || result.Links.Next == "" {
			break
		}
		cursor = extractCursor(result.Links.Next)
		if cursor == "" {
			break
		}
	}

	return allPages, nil
}

// GetPage gets a page by ID.
// Requests both storage and atlas_doc_format to handle both old and new editor pages.
func (s *ConfluenceService) GetPage(ctx context.Context, pageID string) (*Page, error) {
	path := fmt.Sprintf("%s/pages/%s", s.baseURL(), pageID)

	// Try to get storage format first
	params := url.Values{}
	params.Set("body-format", "storage")

	var page Page
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &page); err != nil {
		return nil, err
	}

	// If storage body is empty, try atlas_doc_format (new editor)
	if page.Body == nil || page.Body.Storage == nil || page.Body.Storage.Value == "" {
		params.Set("body-format", "atlas_doc_format")
		var adfPage Page
		if err := s.client.Get(ctx, path+"?"+params.Encode(), &adfPage); err == nil {
			page.Body = adfPage.Body
		}
	}

	return &page, nil
}

// CreatePageRequest represents a request to create a page.
type CreatePageRequest struct {
	SpaceID  string `json:"spaceId"`
	Title    string `json:"title"`
	ParentID string `json:"parentId,omitempty"`
	Status   string `json:"status,omitempty"`
	Body     struct {
		Representation string `json:"representation"`
		Value          string `json:"value"`
	} `json:"body"`
}

// CreatePage creates a new page.
// status can be "current" or "draft". Empty defaults to "current".
func (s *ConfluenceService) CreatePage(ctx context.Context, spaceID, title, content string, parentID string, status string) (*Page, error) {
	path := fmt.Sprintf("%s/pages", s.baseURL())

	if status == "" {
		status = "current"
	}

	reqBody := CreatePageRequest{
		SpaceID:  spaceID,
		Title:    title,
		ParentID: parentID,
		Status:   status,
	}
	reqBody.Body.Representation = "storage"
	reqBody.Body.Value = content

	var page Page
	if err := s.client.Post(ctx, path, reqBody, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// UpdatePageRequest represents a request to update a page.
type UpdatePageRequest struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Title   string `json:"title"`
	Version struct {
		Number  int    `json:"number"`
		Message string `json:"message,omitempty"`
	} `json:"version"`
	Body struct {
		Representation string `json:"representation"`
		Value          string `json:"value"`
	} `json:"body"`
}

// UpdatePage updates an existing page.
func (s *ConfluenceService) UpdatePage(ctx context.Context, pageID, title, content string, version int, message string) (*Page, error) {
	path := fmt.Sprintf("%s/pages/%s", s.baseURL(), pageID)

	reqBody := UpdatePageRequest{
		ID:     pageID,
		Status: "current",
		Title:  title,
	}
	reqBody.Version.Number = version + 1
	reqBody.Version.Message = message
	reqBody.Body.Representation = "storage"
	reqBody.Body.Value = content

	var page Page
	if err := s.client.Put(ctx, path, reqBody, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// DeleteContent deletes a page or folder.
// contentType can be "page", "folder", or empty (auto-detects by trying page then folder).
// Note: v1 /content/{id} DELETE is deprecated (410 Gone), so we only use v2 endpoints.
func (s *ConfluenceService) DeleteContent(ctx context.Context, id string, contentType string) error {
	switch contentType {
	case "folder":
		path := fmt.Sprintf("%s/folders/%s", s.baseURL(), id)
		return s.client.Delete(ctx, path)
	case "page":
		path := fmt.Sprintf("%s/pages/%s", s.baseURL(), id)
		return s.client.Delete(ctx, path)
	default:
		// Auto-detect: try v2 page first, then v2 folder
		pagePath := fmt.Sprintf("%s/pages/%s", s.baseURL(), id)
		err := s.client.Delete(ctx, pagePath)
		if err == nil {
			return nil
		}
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 404 {
			folderPath := fmt.Sprintf("%s/folders/%s", s.baseURL(), id)
			return s.client.Delete(ctx, folderPath)
		}
		return err
	}
}

// PublishPage publishes a draft page by changing its status to current.
func (s *ConfluenceService) PublishPage(ctx context.Context, pageID string) (*Page, error) {
	// First get the draft page
	path := fmt.Sprintf("%s/pages/%s", s.baseURL(), pageID)
	params := url.Values{}
	params.Set("status", "draft")
	params.Set("body-format", "storage")

	var page Page
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &page); err != nil {
		return nil, fmt.Errorf("failed to get draft page: %w", err)
	}

	// Update the page with status=current
	reqBody := UpdatePageRequest{
		ID:     pageID,
		Status: "current",
		Title:  page.Title,
	}
	reqBody.Version.Number = page.Version.Number + 1
	reqBody.Version.Message = "Published via CLI"
	reqBody.Body.Representation = "storage"
	if page.Body != nil && page.Body.Storage != nil {
		reqBody.Body.Value = page.Body.Storage.Value
	} else {
		reqBody.Body.Value = "<p></p>"
	}

	var result Page
	if err := s.client.Put(ctx, path, reqBody, &result); err != nil {
		return nil, fmt.Errorf("failed to publish page: %w", err)
	}

	return &result, nil
}

// baseURLV1 returns the base URL for Confluence v1 API.
//
// V1 is required for: search (CQL), archive, move.
// Note: v1 /content/{id} DELETE and PUT are deprecated (410 Gone).
// Unarchive has no API - must use web UI.
// See ConfluenceService docs for full API version strategy.
func (s *ConfluenceService) baseURLV1() string {
	return s.client.ConfluenceBaseURLV1()
}

// ArchivePage archives a page using the v1 API.
// Note: Archive endpoint only exists in v1 API.
func (s *ConfluenceService) ArchivePage(ctx context.Context, pageID string) error {
	path := fmt.Sprintf("%s/content/archive", s.baseURLV1())
	body := map[string]interface{}{
		"pages": []map[string]string{
			{"id": pageID},
		},
	}
	return s.client.Post(ctx, path, body, nil)
}

// UnarchivePage restores an archived page.
// NOTE: Confluence Cloud has no REST API for unarchiving pages.
// The v1 workaround using PUT /content/{id} was deprecated (410 Gone).
// Users must restore archived pages via the Confluence web UI.
// Feature request: https://jira.atlassian.com/browse/CONFCLOUD-75065
func (s *ConfluenceService) UnarchivePage(ctx context.Context, pageID string) error {
	return fmt.Errorf("unarchive is not supported via API - Confluence has no REST endpoint for restoring archived pages. Please use the Confluence web UI to restore archived pages")
}

// ArchivePages archives multiple pages using the v1 API.
func (s *ConfluenceService) ArchivePages(ctx context.Context, pageIDs []string) error {
	path := fmt.Sprintf("%s/content/archive", s.baseURLV1())
	pages := make([]map[string]string, len(pageIDs))
	for i, id := range pageIDs {
		pages[i] = map[string]string{"id": id}
	}
	body := map[string]interface{}{
		"pages": pages,
	}
	return s.client.Post(ctx, path, body, nil)
}

// MovePosition represents the position for moving a page.
type MovePosition string

const (
	// MovePositionBefore moves the page before the target (same parent as target).
	MovePositionBefore MovePosition = "before"
	// MovePositionAfter moves the page after the target (same parent as target).
	MovePositionAfter MovePosition = "after"
	// MovePositionAppend moves the page to be a child of the target.
	MovePositionAppend MovePosition = "append"
)

// MovePage moves a page to a new location.
// position can be: "before", "after", or "append"
// - "before": move page under same parent as target, before target in list
// - "after": move page under same parent as target, after target in list
// - "append": move page to be a child of the target
// Note: Uses v1 API as this endpoint doesn't exist in v2.
func (s *ConfluenceService) MovePage(ctx context.Context, pageID string, position MovePosition, targetID string) error {
	path := fmt.Sprintf("%s/content/%s/move/%s/%s", s.baseURLV1(), pageID, position, targetID)
	return s.client.Put(ctx, path, nil, nil)
}

// MovePageToSpace moves a page to a different space (makes it a root page in that space).
// spaceKey is the key of the destination space.
func (s *ConfluenceService) MovePageToSpace(ctx context.Context, pageID string, spaceKey string) error {
	// To move to a different space as a root page, we need to get the space's homepage
	// and use "after" position, or use the space root
	space, err := s.GetSpaceByKey(ctx, spaceKey)
	if err != nil {
		return fmt.Errorf("failed to get space: %w", err)
	}

	// Move as child of the space's homepage
	if space.HomepageID != "" {
		return s.MovePage(ctx, pageID, MovePositionAfter, space.HomepageID)
	}

	return fmt.Errorf("space %s has no homepage", spaceKey)
}

// SearchPages searches for pages by title (exact match).
func (s *ConfluenceService) SearchPages(ctx context.Context, query string, limit int) (*PagesResponse, error) {
	path := fmt.Sprintf("%s/pages", s.baseURL())

	params := url.Values{}
	params.Set("title", query)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(capLimit(limit, ConfluenceMaxLimit)))
	}
	params.Set("status", "current")

	var result PagesResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ConfluenceSearchResultV1 represents a search result from v1 CQL search.
// The v1 API wraps content in a "content" field.
type ConfluenceSearchResultV1 struct {
	Content struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Status string `json:"status"`
		Title  string `json:"title"`
		Space  *struct {
			Key string `json:"key"`
		} `json:"space,omitempty"`
	} `json:"content"`
	Excerpt string `json:"excerpt,omitempty"`
	URL     string `json:"url,omitempty"`
}

// ConfluenceSearchResponseV1 represents the v1 search response.
type ConfluenceSearchResponseV1 struct {
	Results []*ConfluenceSearchResultV1 `json:"results"`
	Links   *PaginationLinks            `json:"_links,omitempty"`
	Start   int                         `json:"start"`
	Limit   int                         `json:"limit"`
	Size    int                         `json:"size"`
}

// ConfluenceSearchResult represents a normalized search result.
type ConfluenceSearchResult struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Title    string `json:"title"`
	SpaceKey string `json:"spaceKey,omitempty"`
	Excerpt  string `json:"excerpt,omitempty"`
}

// ConfluenceSearchResponse represents a paginated search response.
type ConfluenceSearchResponse struct {
	Results []*ConfluenceSearchResult `json:"results"`
}

// SearchWithCQL searches for content using CQL (Confluence Query Language).
//
// Example CQL queries:
//   - "title ~ 'keyword'" - search by title
//   - "space = 'SPACEKEY' AND title ~ 'keyword'" - search in specific space
//   - "type = page AND text ~ 'content'" - full-text search
//
// Uses v1 API because search endpoint doesn't exist in v2 (as of Dec 2024).
// Requires OAuth scope: search:confluence (classic scope).
func (s *ConfluenceService) SearchWithCQL(ctx context.Context, cql string, limit int, cursor string) (*ConfluenceSearchResponse, error) {
	path := fmt.Sprintf("%s/search", s.baseURLV1())

	params := url.Values{}
	params.Set("cql", cql)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(capLimit(limit, ConfluenceMaxLimit)))
	}
	if cursor != "" {
		params.Set("start", cursor)
	}

	var v1Result ConfluenceSearchResponseV1
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &v1Result); err != nil {
		return nil, err
	}

	// Convert v1 response to normalized format
	result := &ConfluenceSearchResponse{
		Results: make([]*ConfluenceSearchResult, 0, len(v1Result.Results)),
	}
	for _, r := range v1Result.Results {
		spaceKey := ""
		if r.Content.Space != nil {
			spaceKey = r.Content.Space.Key
		}
		result.Results = append(result.Results, &ConfluenceSearchResult{
			ID:       r.Content.ID,
			Type:     r.Content.Type,
			Status:   r.Content.Status,
			Title:    r.Content.Title,
			SpaceKey: spaceKey,
			Excerpt:  r.Excerpt,
		})
	}

	return result, nil
}

// SearchByTitle searches for pages by title using CQL contains match.
func (s *ConfluenceService) SearchByTitle(ctx context.Context, title string, spaceKey string, limit int) (*ConfluenceSearchResponse, error) {
	var cql string
	if spaceKey != "" {
		cql = fmt.Sprintf("type = page AND space = \"%s\" AND title ~ \"%s\"", spaceKey, title)
	} else {
		cql = fmt.Sprintf("type = page AND title ~ \"%s\"", title)
	}

	return s.SearchWithCQL(ctx, cql, limit, "")
}

// PageChild represents a child or descendant page.
type PageChild struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	Title         string `json:"title"`
	ParentID      string `json:"parentId,omitempty"`
	Depth         int    `json:"depth,omitempty"`
	ChildPosition int    `json:"childPosition,omitempty"`
	Type          string `json:"type"` // "page" or "folder"
}

// ChildrenResponse represents a paginated list of child pages.
type ChildrenResponse struct {
	Results []*PageChild     `json:"results"`
	Links   *PaginationLinks `json:"_links,omitempty"`
}

// GetPageChildren gets immediate children of a page.
func (s *ConfluenceService) GetPageChildren(ctx context.Context, pageID string, limit int, cursor string) (*ChildrenResponse, error) {
	path := fmt.Sprintf("%s/pages/%s/children", s.baseURL(), pageID)

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(capLimit(limit, ConfluenceMaxLimit)))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	var result ChildrenResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetPageDescendants gets all descendants of a page (children, grandchildren, etc.).
func (s *ConfluenceService) GetPageDescendants(ctx context.Context, pageID string, limit int, cursor string) (*ChildrenResponse, error) {
	path := fmt.Sprintf("%s/pages/%s/descendants", s.baseURL(), pageID)

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(capLimit(limit, ConfluenceMaxLimit)))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	var result ChildrenResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetPageDescendantsAll gets all descendants by following pagination.
func (s *ConfluenceService) GetPageDescendantsAll(ctx context.Context, pageID string) ([]*PageChild, error) {
	var all []*PageChild
	cursor := ""

	for {
		result, err := s.GetPageDescendants(ctx, pageID, 100, cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, result.Results...)

		if result.Links == nil || result.Links.Next == "" {
			break
		}
		cursor = extractCursor(result.Links.Next)
		if cursor == "" {
			break
		}
	}

	return all, nil
}

// Template represents a Confluence content template.
type Template struct {
	TemplateID   string        `json:"templateId"`
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	TemplateType string        `json:"templateType"` // "page" or "blogpost"
	Body         *TemplateBody `json:"body,omitempty"`
	Space        *SpaceRef     `json:"space,omitempty"`
	Labels       []Label       `json:"labels,omitempty"`
}

// TemplateBody represents the body of a template.
type TemplateBody struct {
	Storage *BodyContent `json:"storage,omitempty"`
	View    *BodyContent `json:"view,omitempty"`
}

// SpaceRef is a reference to a space in template responses.
type SpaceRef struct {
	Key string `json:"key"`
}

// Label represents a label on content.
type Label struct {
	Name   string `json:"name"`
	Prefix string `json:"prefix,omitempty"`
}

// CreateTemplateRequest represents a request to create a template.
type CreateTemplateRequest struct {
	Name         string `json:"name"`
	TemplateType string `json:"templateType"` // "page"
	Description  string `json:"description,omitempty"`
	Body         struct {
		Storage struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"storage"`
	} `json:"body"`
	Space *struct {
		Key string `json:"key"`
	} `json:"space,omitempty"`
}

// UpdateTemplateRequest represents a request to update a template.
type UpdateTemplateRequest struct {
	TemplateID   string `json:"templateId"`
	Name         string `json:"name"`
	TemplateType string `json:"templateType"`
	Description  string `json:"description,omitempty"`
	Body         struct {
		Storage struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"storage"`
	} `json:"body"`
	Space *struct {
		Key string `json:"key"`
	} `json:"space,omitempty"`
}

// GetTemplate gets a template by ID.
// Uses v1 API as templates are not available in v2.
func (s *ConfluenceService) GetTemplate(ctx context.Context, templateID string) (*Template, error) {
	path := fmt.Sprintf("%s/template/%s", s.baseURLV1(), templateID)

	var template Template
	if err := s.client.Get(ctx, path, &template); err != nil {
		return nil, err
	}

	return &template, nil
}

// CreateTemplate creates a new content template.
// If spaceKey is empty, creates a global template (requires Confluence Administrator permission).
// If spaceKey is provided, creates a space template (requires Space Admin permission).
// Uses v1 API as templates are not available in v2.
func (s *ConfluenceService) CreateTemplate(ctx context.Context, name, body, description, spaceKey string) (*Template, error) {
	path := fmt.Sprintf("%s/template", s.baseURLV1())

	reqBody := CreateTemplateRequest{
		Name:         name,
		TemplateType: "page",
		Description:  description,
	}
	reqBody.Body.Storage.Value = body
	reqBody.Body.Storage.Representation = "storage"

	if spaceKey != "" {
		reqBody.Space = &struct {
			Key string `json:"key"`
		}{Key: spaceKey}
	}

	var template Template
	if err := s.client.Post(ctx, path, reqBody, &template); err != nil {
		return nil, err
	}

	return &template, nil
}

// UpdateTemplate updates an existing content template.
// Uses v1 API as templates are not available in v2.
func (s *ConfluenceService) UpdateTemplate(ctx context.Context, templateID, name, body, description string) (*Template, error) {
	// First get the existing template to preserve space info
	existing, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing template: %w", err)
	}

	path := fmt.Sprintf("%s/template", s.baseURLV1())

	reqBody := UpdateTemplateRequest{
		TemplateID:   templateID,
		Name:         name,
		TemplateType: existing.TemplateType,
		Description:  description,
	}
	reqBody.Body.Storage.Value = body
	reqBody.Body.Storage.Representation = "storage"

	if existing.Space != nil {
		reqBody.Space = &struct {
			Key string `json:"key"`
		}{Key: existing.Space.Key}
	}

	var template Template
	if err := s.client.Put(ctx, path, reqBody, &template); err != nil {
		return nil, err
	}

	return &template, nil
}
