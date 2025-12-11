package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ConfluenceService handles Confluence API operations.
type ConfluenceService struct {
	client *Client
}

// NewConfluenceService creates a new Confluence service.
func NewConfluenceService(client *Client) *ConfluenceService {
	return &ConfluenceService{client: client}
}

// Space represents a Confluence space.
type Space struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description *SpaceDescription `json:"description,omitempty"`
	Status      string `json:"status"`
	HomepageID  string `json:"homepageId,omitempty"`
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
	SpaceID   string       `json:"spaceId"`
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
	Results []*Space `json:"results"`
	Links   *PaginationLinks `json:"_links,omitempty"`
}

// PagesResponse represents a paginated list of pages.
type PagesResponse struct {
	Results []*Page `json:"results"`
	Links   *PaginationLinks `json:"_links,omitempty"`
}

// PaginationLinks represents pagination links.
type PaginationLinks struct {
	Next string `json:"next,omitempty"`
}

// GetSpaces gets a list of spaces.
func (s *ConfluenceService) GetSpaces(ctx context.Context, limit int, cursor string) (*SpacesResponse, error) {
	path := fmt.Sprintf("%s/spaces", s.client.ConfluenceBaseURL())

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
		// Extract cursor from next URL
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
	path := fmt.Sprintf("%s/spaces/%s", s.client.ConfluenceBaseURL(), spaceID)

	var space Space
	if err := s.client.Get(ctx, path, &space); err != nil {
		return nil, err
	}

	return &space, nil
}

// GetSpaceByKey gets a space by its key.
func (s *ConfluenceService) GetSpaceByKey(ctx context.Context, key string) (*Space, error) {
	path := fmt.Sprintf("%s/spaces", s.client.ConfluenceBaseURL())

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

// GetPagesInSpace gets pages in a space.
func (s *ConfluenceService) GetPagesInSpace(ctx context.Context, spaceID string, limit int, cursor string) (*PagesResponse, error) {
	path := fmt.Sprintf("%s/spaces/%s/pages", s.client.ConfluenceBaseURL(), spaceID)

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
func (s *ConfluenceService) GetPagesInSpaceAll(ctx context.Context, spaceID string) ([]*Page, error) {
	var allPages []*Page
	cursor := ""

	for {
		result, err := s.GetPagesInSpace(ctx, spaceID, 100, cursor)
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
	path := fmt.Sprintf("%s/pages/%s", s.client.ConfluenceBaseURL(), pageID)

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
	SpaceID  string          `json:"spaceId"`
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
	path := fmt.Sprintf("%s/pages", s.client.ConfluenceBaseURL())

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
	path := fmt.Sprintf("%s/pages/%s", s.client.ConfluenceBaseURL(), pageID)

	var page Page
	if err := s.client.Put(ctx, path, req, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// SearchPages searches for pages using CQL.
func (s *ConfluenceService) SearchPages(ctx context.Context, spaceKey, title string, limit int) (*PagesResponse, error) {
	// Use the pages endpoint with filters
	space, err := s.GetSpaceByKey(ctx, spaceKey)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/spaces/%s/pages", s.client.ConfluenceBaseURL(), space.ID)

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
