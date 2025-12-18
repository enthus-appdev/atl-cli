package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ConfluenceService handles Confluence API operations.
// Always uses the v2 API as v1 has been deprecated and removed by Atlassian.
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
func (s *ConfluenceService) GetPages(ctx context.Context, spaceID string, limit int, cursor string) (*PagesResponse, error) {
	path := fmt.Sprintf("%s/spaces/%s/pages", s.baseURL(), spaceID)

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	params.Set("status", "current")
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
func (s *ConfluenceService) GetPagesAll(ctx context.Context, spaceID string) ([]*Page, error) {
	var allPages []*Page
	cursor := ""

	for {
		result, err := s.GetPages(ctx, spaceID, 100, cursor)
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
func (s *ConfluenceService) GetPage(ctx context.Context, pageID string) (*Page, error) {
	path := fmt.Sprintf("%s/pages/%s", s.baseURL(), pageID)

	params := url.Values{}
	params.Set("body-format", "storage")

	var page Page
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &page); err != nil {
		return nil, err
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
func (s *ConfluenceService) CreatePage(ctx context.Context, spaceID, title, content string, parentID string) (*Page, error) {
	path := fmt.Sprintf("%s/pages", s.baseURL())

	reqBody := CreatePageRequest{
		SpaceID:  spaceID,
		Title:    title,
		ParentID: parentID,
		Status:   "current",
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

// DeletePage deletes a page.
func (s *ConfluenceService) DeletePage(ctx context.Context, pageID string) error {
	path := fmt.Sprintf("%s/pages/%s", s.baseURL(), pageID)
	return s.client.Delete(ctx, path)
}

// baseURLV1 returns the base URL for Confluence v1 API.
// Some endpoints like archive only exist in v1.
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
// Note: There's no dedicated unarchive endpoint in the REST API.
// This uses the update page endpoint to change status back to current.
func (s *ConfluenceService) UnarchivePage(ctx context.Context, pageID string) error {
	// First get the archived page to get its current version
	path := fmt.Sprintf("%s/content/%s?status=archived", s.baseURLV1(), pageID)
	var page struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Type    string `json:"type"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
	}
	if err := s.client.Get(ctx, path, &page); err != nil {
		return fmt.Errorf("failed to get archived page: %w", err)
	}

	// Update the page status to current
	updatePath := fmt.Sprintf("%s/content/%s", s.baseURLV1(), pageID)
	updateBody := map[string]interface{}{
		"id":     pageID,
		"type":   page.Type,
		"title":  page.Title,
		"status": "current",
		"version": map[string]int{
			"number": page.Version.Number + 1,
		},
	}
	return s.client.Put(ctx, updatePath, updateBody, nil)
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

// SearchPages searches for pages by title (exact match).
func (s *ConfluenceService) SearchPages(ctx context.Context, query string, limit int) (*PagesResponse, error) {
	path := fmt.Sprintf("%s/pages", s.baseURL())

	params := url.Values{}
	params.Set("title", query)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	params.Set("status", "current")

	var result PagesResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ConfluenceSearchResult represents a search result from CQL search.
type ConfluenceSearchResult struct {
	ID         string                     `json:"id"`
	Type       string                     `json:"type"`
	Status     string                     `json:"status"`
	Title      string                     `json:"title"`
	SpaceID    string                     `json:"spaceId,omitempty"`
	ParentID   string                     `json:"parentId,omitempty"`
	ParentType string                     `json:"parentType,omitempty"`
	Excerpt    string                     `json:"excerpt,omitempty"`
	Links      *ConfluenceSearchResultLink `json:"_links,omitempty"`
}

// ConfluenceSearchResultLink contains links for a search result.
type ConfluenceSearchResultLink struct {
	WebUI string `json:"webui,omitempty"`
}

// ConfluenceSearchResponse represents a paginated search response.
type ConfluenceSearchResponse struct {
	Results []*ConfluenceSearchResult `json:"results"`
	Links   *PaginationLinks          `json:"_links,omitempty"`
}

// SearchWithCQL searches for content using CQL (Confluence Query Language).
// Example CQL: "title ~ 'keyword'" or "space = 'SPACEKEY' AND title ~ 'keyword'"
func (s *ConfluenceService) SearchWithCQL(ctx context.Context, cql string, limit int, cursor string) (*ConfluenceSearchResponse, error) {
	path := fmt.Sprintf("%s/search", s.baseURL())

	params := url.Values{}
	params.Set("cql", cql)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	var result ConfluenceSearchResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
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
		params.Set("limit", strconv.Itoa(limit))
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
		params.Set("limit", strconv.Itoa(limit))
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
