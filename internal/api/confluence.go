package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/enthus-appdev/atl-cli/internal/config"
)

// ConfluenceService handles Confluence API operations.
// Supports both v1 (classic scopes) and v2 (granular scopes) APIs.
type ConfluenceService struct {
	client *Client
}

// NewConfluenceService creates a new Confluence service.
func NewConfluenceService(client *Client) *ConfluenceService {
	return &ConfluenceService{client: client}
}

// Space represents a Confluence space (unified for v1 and v2).
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

// Page represents a Confluence page (unified for v1 and v2).
type Page struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	SpaceID   string       `json:"spaceId,omitempty"`   // v2
	SpaceKey  string       `json:"spaceKey,omitempty"`  // derived from v1
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
	Start   int              `json:"start,omitempty"` // v1
	Limit   int              `json:"limit,omitempty"` // v1
	Size    int              `json:"size,omitempty"`  // v1
}

// PagesResponse represents a paginated list of pages.
type PagesResponse struct {
	Results []*Page          `json:"results"`
	Links   *PaginationLinks `json:"_links,omitempty"`
	Start   int              `json:"start,omitempty"` // v1
	Limit   int              `json:"limit,omitempty"` // v1
	Size    int              `json:"size,omitempty"`  // v1
}

// PaginationLinks represents pagination links.
type PaginationLinks struct {
	Next string `json:"next,omitempty"`
}

// ========== SPACE OPERATIONS ==========

// GetSpaces gets a list of spaces.
func (s *ConfluenceService) GetSpaces(ctx context.Context, limit int, cursor string) (*SpacesResponse, error) {
	if s.client.APIVersion() == config.APIVersionV2 {
		return s.getSpacesV2(ctx, limit, cursor)
	}
	return s.getSpacesV1(ctx, limit, cursor)
}

func (s *ConfluenceService) getSpacesV1(ctx context.Context, limit int, startAt string) (*SpacesResponse, error) {
	path := fmt.Sprintf("%s/space", s.client.ConfluenceBaseURLV1())

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if startAt != "" {
		params.Set("start", startAt)
	}
	params.Set("type", "global")
	params.Set("status", "current")

	var result SpacesResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *ConfluenceService) getSpacesV2(ctx context.Context, limit int, cursor string) (*SpacesResponse, error) {
	path := fmt.Sprintf("%s/spaces", s.client.ConfluenceBaseURLV2())

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
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
		// Extract cursor/start from next URL
		cursor = extractPaginationParam(result.Links.Next, s.client.APIVersion())
		if cursor == "" {
			break
		}
	}

	return allSpaces, nil
}

// extractPaginationParam extracts the pagination parameter from a next URL.
func extractPaginationParam(nextURL string, apiVersion config.APIVersion) string {
	parsed, err := url.Parse(nextURL)
	if err != nil {
		return ""
	}
	if apiVersion == config.APIVersionV2 {
		return parsed.Query().Get("cursor")
	}
	return parsed.Query().Get("start")
}

// extractCursor extracts the cursor parameter from a pagination URL (v2).
func extractCursor(nextURL string) string {
	parsed, err := url.Parse(nextURL)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("cursor")
}

// GetSpace gets a space by ID (v2) or key (v1).
func (s *ConfluenceService) GetSpace(ctx context.Context, spaceIDOrKey string) (*Space, error) {
	if s.client.APIVersion() == config.APIVersionV2 {
		return s.getSpaceV2(ctx, spaceIDOrKey)
	}
	return s.getSpaceV1(ctx, spaceIDOrKey)
}

func (s *ConfluenceService) getSpaceV1(ctx context.Context, spaceKey string) (*Space, error) {
	path := fmt.Sprintf("%s/space/%s", s.client.ConfluenceBaseURLV1(), spaceKey)

	var space Space
	if err := s.client.Get(ctx, path, &space); err != nil {
		return nil, err
	}

	return &space, nil
}

func (s *ConfluenceService) getSpaceV2(ctx context.Context, spaceID string) (*Space, error) {
	path := fmt.Sprintf("%s/spaces/%s", s.client.ConfluenceBaseURLV2(), spaceID)

	var space Space
	if err := s.client.Get(ctx, path, &space); err != nil {
		return nil, err
	}

	return &space, nil
}

// GetSpaceByKey gets a space by its key.
func (s *ConfluenceService) GetSpaceByKey(ctx context.Context, key string) (*Space, error) {
	if s.client.APIVersion() == config.APIVersionV2 {
		return s.getSpaceByKeyV2(ctx, key)
	}
	return s.getSpaceV1(ctx, key) // v1 uses key directly
}

func (s *ConfluenceService) getSpaceByKeyV2(ctx context.Context, key string) (*Space, error) {
	path := fmt.Sprintf("%s/spaces", s.client.ConfluenceBaseURLV2())

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

// ========== PAGE OPERATIONS ==========

// GetPagesInSpace gets pages in a space.
func (s *ConfluenceService) GetPagesInSpace(ctx context.Context, spaceIDOrKey string, limit int, cursor string) (*PagesResponse, error) {
	if s.client.APIVersion() == config.APIVersionV2 {
		return s.getPagesInSpaceV2(ctx, spaceIDOrKey, limit, cursor)
	}
	return s.getPagesInSpaceV1(ctx, spaceIDOrKey, limit, cursor)
}

func (s *ConfluenceService) getPagesInSpaceV1(ctx context.Context, spaceKey string, limit int, startAt string) (*PagesResponse, error) {
	path := fmt.Sprintf("%s/content", s.client.ConfluenceBaseURLV1())

	params := url.Values{}
	params.Set("spaceKey", spaceKey)
	params.Set("type", "page")
	params.Set("status", "current")
	params.Set("orderby", "history.lastUpdated desc")
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if startAt != "" {
		params.Set("start", startAt)
	}

	// V1 returns content with different structure, need to transform
	var v1Result struct {
		Results []*v1Content     `json:"results"`
		Links   *PaginationLinks `json:"_links,omitempty"`
		Start   int              `json:"start"`
		Limit   int              `json:"limit"`
		Size    int              `json:"size"`
	}

	if err := s.client.Get(ctx, path+"?"+params.Encode(), &v1Result); err != nil {
		return nil, err
	}

	// Transform v1 content to Page
	result := &PagesResponse{
		Links: v1Result.Links,
		Start: v1Result.Start,
		Limit: v1Result.Limit,
		Size:  v1Result.Size,
	}
	for _, c := range v1Result.Results {
		result.Results = append(result.Results, c.toPage())
	}

	return result, nil
}

// v1Content represents a content item in v1 API response
type v1Content struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Title   string `json:"title"`
	Space   *struct {
		Key string `json:"key"`
	} `json:"space,omitempty"`
	Version *PageVersion `json:"version,omitempty"`
	Body    *struct {
		Storage *BodyContent `json:"storage,omitempty"`
		View    *BodyContent `json:"view,omitempty"`
	} `json:"body,omitempty"`
	Links *PageLinks `json:"_links,omitempty"`
}

func (c *v1Content) toPage() *Page {
	page := &Page{
		ID:      c.ID,
		Title:   c.Title,
		Status:  c.Status,
		Version: c.Version,
		Links:   c.Links,
	}
	if c.Space != nil {
		page.SpaceKey = c.Space.Key
	}
	if c.Body != nil {
		page.Body = &PageBody{
			Storage: c.Body.Storage,
			View:    c.Body.View,
		}
	}
	return page
}

func (s *ConfluenceService) getPagesInSpaceV2(ctx context.Context, spaceID string, limit int, cursor string) (*PagesResponse, error) {
	path := fmt.Sprintf("%s/spaces/%s/pages", s.client.ConfluenceBaseURLV2(), spaceID)

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	params.Set("status", "current")
	params.Set("sort", "-modified-date")
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	var result PagesResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetPagesInSpaceAll gets all pages in a space by following pagination.
func (s *ConfluenceService) GetPagesInSpaceAll(ctx context.Context, spaceIDOrKey string) ([]*Page, error) {
	var allPages []*Page
	cursor := ""

	for {
		result, err := s.GetPagesInSpace(ctx, spaceIDOrKey, 100, cursor)
		if err != nil {
			return nil, err
		}
		allPages = append(allPages, result.Results...)

		if result.Links == nil || result.Links.Next == "" {
			break
		}
		cursor = extractPaginationParam(result.Links.Next, s.client.APIVersion())
		if cursor == "" {
			break
		}
	}

	return allPages, nil
}

// GetPage gets a page by ID.
func (s *ConfluenceService) GetPage(ctx context.Context, pageID string) (*Page, error) {
	if s.client.APIVersion() == config.APIVersionV2 {
		return s.getPageV2(ctx, pageID)
	}
	return s.getPageV1(ctx, pageID)
}

func (s *ConfluenceService) getPageV1(ctx context.Context, pageID string) (*Page, error) {
	path := fmt.Sprintf("%s/content/%s", s.client.ConfluenceBaseURLV1(), pageID)

	params := url.Values{}
	params.Set("expand", "body.storage,version,space")

	var content v1Content
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &content); err != nil {
		return nil, err
	}

	return content.toPage(), nil
}

func (s *ConfluenceService) getPageV2(ctx context.Context, pageID string) (*Page, error) {
	path := fmt.Sprintf("%s/pages/%s", s.client.ConfluenceBaseURLV2(), pageID)

	params := url.Values{}
	params.Set("body-format", "storage")

	var page Page
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// ========== CREATE/UPDATE OPERATIONS ==========

// CreatePageRequest represents a request to create a page.
type CreatePageRequest struct {
	SpaceID  string          `json:"spaceId,omitempty"`  // v2
	SpaceKey string          `json:"spaceKey,omitempty"` // v1 (derived)
	Status   string          `json:"status"`
	Title    string          `json:"title"`
	ParentID string          `json:"parentId,omitempty"`
	Body     *CreatePageBody `json:"body"`
}

// CreatePageBody represents the body for creating a page.
type CreatePageBody struct {
	Representation string `json:"representation"`
	Value          string `json:"value"`
}

// CreatePage creates a new page.
func (s *ConfluenceService) CreatePage(ctx context.Context, req *CreatePageRequest) (*Page, error) {
	if s.client.APIVersion() == config.APIVersionV2 {
		return s.createPageV2(ctx, req)
	}
	return s.createPageV1(ctx, req)
}

func (s *ConfluenceService) createPageV1(ctx context.Context, req *CreatePageRequest) (*Page, error) {
	path := fmt.Sprintf("%s/content", s.client.ConfluenceBaseURLV1())

	// Transform request for v1 API
	v1Req := map[string]interface{}{
		"type":   "page",
		"title":  req.Title,
		"status": req.Status,
		"space": map[string]string{
			"key": req.SpaceKey,
		},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          req.Body.Value,
				"representation": req.Body.Representation,
			},
		},
	}
	if req.ParentID != "" {
		v1Req["ancestors"] = []map[string]string{{"id": req.ParentID}}
	}

	var content v1Content
	if err := s.client.Post(ctx, path, v1Req, &content); err != nil {
		return nil, err
	}

	return content.toPage(), nil
}

func (s *ConfluenceService) createPageV2(ctx context.Context, req *CreatePageRequest) (*Page, error) {
	path := fmt.Sprintf("%s/pages", s.client.ConfluenceBaseURLV2())

	var page Page
	if err := s.client.Post(ctx, path, req, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// UpdatePageRequest represents a request to update a page.
type UpdatePageRequest struct {
	ID      string          `json:"id"`
	Status  string          `json:"status"`
	Title   string          `json:"title"`
	SpaceID string          `json:"spaceId,omitempty"`
	Body    *CreatePageBody `json:"body"`
	Version *UpdateVersion  `json:"version"`
}

// UpdateVersion represents the version for updating a page.
type UpdateVersion struct {
	Number  int    `json:"number"`
	Message string `json:"message,omitempty"`
}

// UpdatePage updates an existing page.
func (s *ConfluenceService) UpdatePage(ctx context.Context, pageID string, req *UpdatePageRequest) (*Page, error) {
	if s.client.APIVersion() == config.APIVersionV2 {
		return s.updatePageV2(ctx, pageID, req)
	}
	return s.updatePageV1(ctx, pageID, req)
}

func (s *ConfluenceService) updatePageV1(ctx context.Context, pageID string, req *UpdatePageRequest) (*Page, error) {
	path := fmt.Sprintf("%s/content/%s", s.client.ConfluenceBaseURLV1(), pageID)

	// Transform request for v1 API
	v1Req := map[string]interface{}{
		"type":   "page",
		"title":  req.Title,
		"status": req.Status,
		"version": map[string]interface{}{
			"number": req.Version.Number,
		},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          req.Body.Value,
				"representation": req.Body.Representation,
			},
		},
	}

	var content v1Content
	if err := s.client.Put(ctx, path, v1Req, &content); err != nil {
		return nil, err
	}

	return content.toPage(), nil
}

func (s *ConfluenceService) updatePageV2(ctx context.Context, pageID string, req *UpdatePageRequest) (*Page, error) {
	path := fmt.Sprintf("%s/pages/%s", s.client.ConfluenceBaseURLV2(), pageID)

	var page Page
	if err := s.client.Put(ctx, path, req, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// SearchPages searches for pages using CQL (v1) or filters (v2).
func (s *ConfluenceService) SearchPages(ctx context.Context, spaceKey, title string, limit int) (*PagesResponse, error) {
	if s.client.APIVersion() == config.APIVersionV2 {
		// For v2, we need to get space ID first
		space, err := s.GetSpaceByKey(ctx, spaceKey)
		if err != nil {
			return nil, err
		}
		return s.searchPagesV2(ctx, space.ID, title, limit)
	}
	return s.searchPagesV1(ctx, spaceKey, title, limit)
}

func (s *ConfluenceService) searchPagesV1(ctx context.Context, spaceKey, title string, limit int) (*PagesResponse, error) {
	path := fmt.Sprintf("%s/content", s.client.ConfluenceBaseURLV1())

	params := url.Values{}
	params.Set("spaceKey", spaceKey)
	params.Set("type", "page")
	if title != "" {
		params.Set("title", title)
	}
	params.Set("status", "current")
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	var v1Result struct {
		Results []*v1Content     `json:"results"`
		Links   *PaginationLinks `json:"_links,omitempty"`
	}

	if err := s.client.Get(ctx, path+"?"+params.Encode(), &v1Result); err != nil {
		return nil, err
	}

	result := &PagesResponse{Links: v1Result.Links}
	for _, c := range v1Result.Results {
		result.Results = append(result.Results, c.toPage())
	}

	return result, nil
}

func (s *ConfluenceService) searchPagesV2(ctx context.Context, spaceID, title string, limit int) (*PagesResponse, error) {
	path := fmt.Sprintf("%s/spaces/%s/pages", s.client.ConfluenceBaseURLV2(), spaceID)

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if title != "" {
		params.Set("title", title)
	}
	params.Set("status", "current")

	var result PagesResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}
