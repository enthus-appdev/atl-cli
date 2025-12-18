package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// JiraService handles Jira API operations.
type JiraService struct {
	client *Client
}

// NewJiraService creates a new Jira service.
func NewJiraService(client *Client) *JiraService {
	return &JiraService{client: client}
}

// Issue represents a Jira issue.
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

// IssueFields contains the fields of a Jira issue.
type IssueFields struct {
	Summary     string       `json:"summary"`
	Description *ADF         `json:"description,omitempty"`
	Status      *Status      `json:"status,omitempty"`
	Priority    *Priority    `json:"priority,omitempty"`
	IssueType   *IssueType   `json:"issuetype,omitempty"`
	Assignee    *User        `json:"assignee,omitempty"`
	Reporter    *User        `json:"reporter,omitempty"`
	Project     *Project     `json:"project,omitempty"`
	Labels      []string     `json:"labels,omitempty"`
	Created     string       `json:"created,omitempty"`
	Updated     string       `json:"updated,omitempty"`
	Resolution  *Resolution  `json:"resolution,omitempty"`
	Components  []*Component `json:"components,omitempty"`
	Comment     *Comments    `json:"comment,omitempty"`
	Parent      *Issue       `json:"parent,omitempty"`
}

// ADF represents Atlassian Document Format content.
type ADF struct {
	Type    string        `json:"type"`
	Version int           `json:"version,omitempty"`
	Content []ADFContent  `json:"content,omitempty"`
	Text    string        `json:"text,omitempty"`
	Attrs   *ADFAttrs     `json:"attrs,omitempty"`
	Marks   []ADFMark     `json:"marks,omitempty"`
}

// ADFContent represents content within an ADF document.
type ADFContent struct {
	Type    string        `json:"type"`
	Content []ADFContent  `json:"content,omitempty"`
	Text    string        `json:"text,omitempty"`
	Attrs   *ADFAttrs     `json:"attrs,omitempty"`
	Marks   []ADFMark     `json:"marks,omitempty"`
}

// ADFAttrs represents attributes in ADF.
type ADFAttrs struct {
	Level int    `json:"level,omitempty"`
	URL   string `json:"url,omitempty"`
}

// ADFMark represents text marks in ADF.
type ADFMark struct {
	Type  string    `json:"type"`
	Attrs *ADFAttrs `json:"attrs,omitempty"`
}

// Status represents an issue status.
type Status struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	StatusCategory *StatusCategory `json:"statusCategory,omitempty"`
}

// StatusCategory represents a status category.
type StatusCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Priority represents an issue priority.
type Priority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// IssueType represents an issue type.
type IssueType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Subtask     bool   `json:"subtask"`
}

// User represents a Jira user.
type User struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress,omitempty"`
	Active       bool   `json:"active"`
	AvatarUrls   map[string]string `json:"avatarUrls,omitempty"`
}

// Project represents a Jira project.
type Project struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Resolution represents an issue resolution.
type Resolution struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Component represents a project component.
type Component struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Comments represents the comment field on an issue.
type Comments struct {
	Comments   []*Comment `json:"comments"`
	MaxResults int        `json:"maxResults"`
	Total      int        `json:"total"`
	StartAt    int        `json:"startAt"`
}

// Comment represents a Jira comment.
type Comment struct {
	ID      string `json:"id"`
	Author  *User  `json:"author,omitempty"`
	Body    *ADF   `json:"body,omitempty"`
	Created string `json:"created,omitempty"`
	Updated string `json:"updated,omitempty"`
}

// Transition represents a workflow transition.
type Transition struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	To   *Status `json:"to,omitempty"`
}

// SearchResult represents the result of a JQL search.
// Note: The new /search/jql endpoint uses nextPageToken for pagination instead of startAt.
type SearchResult struct {
	Issues        []*Issue `json:"issues"`
	Total         int      `json:"total"`
	MaxResults    int      `json:"maxResults"`
	StartAt       int      `json:"startAt"`       // Deprecated: use NextPageToken
	NextPageToken string   `json:"nextPageToken"` // Token for fetching the next page
	IsLast        bool     `json:"isLast"`        // True if this is the last page
}

// TransitionsResponse represents available transitions for an issue.
type TransitionsResponse struct {
	Transitions []*Transition `json:"transitions"`
}

// GetIssue fetches a single issue by key.
func (s *JiraService) GetIssue(ctx context.Context, key string) (*Issue, error) {
	path := fmt.Sprintf("%s/issue/%s", s.client.JiraBaseURL(), key)

	params := url.Values{}
	params.Set("expand", "renderedFields")
	params.Set("fields", "*all")

	var issue Issue
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

// SearchOptions contains options for searching issues.
type SearchOptions struct {
	JQL           string
	MaxResults    int
	Fields        []string
	NextPageToken string // Token for pagination (replaces startAt)
}

// Search searches for issues using JQL.
// Uses the new /search/jql endpoint which replaces the deprecated /search endpoint.
func (s *JiraService) Search(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	path := fmt.Sprintf("%s/search/jql", s.client.JiraBaseURL())

	params := url.Values{}
	params.Set("jql", opts.JQL)
	if opts.MaxResults > 0 {
		params.Set("maxResults", strconv.Itoa(opts.MaxResults))
	}
	if opts.NextPageToken != "" {
		params.Set("nextPageToken", opts.NextPageToken)
	}
	if len(opts.Fields) > 0 {
		params.Set("fields", strings.Join(opts.Fields, ","))
	} else {
		params.Set("fields", "summary,status,priority,issuetype,assignee,reporter,created,updated,labels,project")
	}

	var result SearchResult
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateIssueRequest represents a request to create an issue.
type CreateIssueRequest struct {
	Fields CreateIssueFields `json:"fields"`
}

// CreateIssueFields contains fields for creating an issue.
type CreateIssueFields struct {
	Project      *ProjectID             `json:"project"`
	Summary      string                 `json:"summary"`
	Description  *ADF                   `json:"description,omitempty"`
	IssueType    *IssueTypeID           `json:"issuetype"`
	Assignee     *AccountID             `json:"assignee,omitempty"`
	Priority     *PriorityID            `json:"priority,omitempty"`
	Labels       []string               `json:"labels,omitempty"`
	Parent       *ParentID              `json:"parent,omitempty"`
	CustomFields map[string]interface{} `json:"-"` // Merged during marshaling
}

// MarshalJSON implements custom JSON marshaling to include custom fields.
func (r *CreateIssueRequest) MarshalJSON() ([]byte, error) {
	// Build the fields map with standard fields
	fields := map[string]interface{}{
		"project":   r.Fields.Project,
		"summary":   r.Fields.Summary,
		"issuetype": r.Fields.IssueType,
	}

	if r.Fields.Description != nil {
		fields["description"] = r.Fields.Description
	}
	if r.Fields.Assignee != nil {
		fields["assignee"] = r.Fields.Assignee
	}
	if r.Fields.Priority != nil {
		fields["priority"] = r.Fields.Priority
	}
	if len(r.Fields.Labels) > 0 {
		fields["labels"] = r.Fields.Labels
	}
	if r.Fields.Parent != nil {
		fields["parent"] = r.Fields.Parent
	}

	// Merge custom fields
	for k, v := range r.Fields.CustomFields {
		fields[k] = v
	}

	return json.Marshal(map[string]interface{}{
		"fields": fields,
	})
}

// ProjectID is used when creating issues.
type ProjectID struct {
	Key string `json:"key"`
}

// IssueTypeID is used when creating issues.
type IssueTypeID struct {
	Name string `json:"name"`
}

// AccountID is used when setting assignee.
type AccountID struct {
	AccountID string `json:"accountId"`
}

// PriorityID is used when setting priority.
type PriorityID struct {
	Name string `json:"name"`
}

// ParentID is used when creating subtasks.
type ParentID struct {
	Key string `json:"key"`
}

// CreateIssueResponse represents the response from creating an issue.
type CreateIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// CreateIssue creates a new issue.
func (s *JiraService) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*CreateIssueResponse, error) {
	path := fmt.Sprintf("%s/issue", s.client.JiraBaseURL())

	var result CreateIssueResponse
	if err := s.client.Post(ctx, path, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateIssueRequest represents a request to update an issue.
type UpdateIssueRequest struct {
	Fields map[string]interface{} `json:"fields,omitempty"`
	Update map[string][]UpdateOp  `json:"update,omitempty"`
}

// UpdateOp represents an update operation.
type UpdateOp struct {
	Add    interface{} `json:"add,omitempty"`
	Remove interface{} `json:"remove,omitempty"`
	Set    interface{} `json:"set,omitempty"`
}

// UpdateIssue updates an existing issue.
func (s *JiraService) UpdateIssue(ctx context.Context, key string, req *UpdateIssueRequest) error {
	path := fmt.Sprintf("%s/issue/%s", s.client.JiraBaseURL(), key)
	return s.client.Put(ctx, path, req, nil)
}

// GetTransitions gets available transitions for an issue.
func (s *JiraService) GetTransitions(ctx context.Context, key string) ([]*Transition, error) {
	path := fmt.Sprintf("%s/issue/%s/transitions", s.client.JiraBaseURL(), key)

	var result TransitionsResponse
	if err := s.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return result.Transitions, nil
}

// TransitionRequest represents a request to transition an issue.
type TransitionRequest struct {
	Transition TransitionID `json:"transition"`
}

// TransitionID identifies a transition.
type TransitionID struct {
	ID string `json:"id"`
}

// TransitionIssue transitions an issue to a new status.
func (s *JiraService) TransitionIssue(ctx context.Context, key string, transitionID string) error {
	path := fmt.Sprintf("%s/issue/%s/transitions", s.client.JiraBaseURL(), key)
	req := &TransitionRequest{
		Transition: TransitionID{ID: transitionID},
	}
	return s.client.Post(ctx, path, req, nil)
}

// CommentVisibility represents visibility restrictions for a comment.
type CommentVisibility struct {
	Type       string `json:"type"`                 // "role" or "group"
	Value      string `json:"value"`                // role name or group name
	Identifier string `json:"identifier,omitempty"` // group ID (for group type)
}

// AddCommentRequest represents a request to add a comment.
type AddCommentRequest struct {
	Body       *ADF               `json:"body"`
	Visibility *CommentVisibility `json:"visibility,omitempty"`
}

// CommentOptions contains options for adding/editing comments.
type CommentOptions struct {
	Body           string
	VisibilityType string // "role" or "group"
	VisibilityName string // role name or group name
}

// AddComment adds a comment to an issue.
func (s *JiraService) AddComment(ctx context.Context, key string, body string) (*Comment, error) {
	return s.AddCommentWithOptions(ctx, key, &CommentOptions{Body: body})
}

// AddCommentWithOptions adds a comment with optional visibility restrictions.
func (s *JiraService) AddCommentWithOptions(ctx context.Context, key string, opts *CommentOptions) (*Comment, error) {
	path := fmt.Sprintf("%s/issue/%s/comment", s.client.JiraBaseURL(), key)

	req := &AddCommentRequest{
		Body: TextToADF(opts.Body),
	}

	if opts.VisibilityType != "" && opts.VisibilityName != "" {
		req.Visibility = &CommentVisibility{
			Type:  opts.VisibilityType,
			Value: opts.VisibilityName,
		}
	}

	var result Comment
	if err := s.client.Post(ctx, path, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetComment gets a single comment by ID.
func (s *JiraService) GetComment(ctx context.Context, key string, commentID string) (*Comment, error) {
	path := fmt.Sprintf("%s/issue/%s/comment/%s", s.client.JiraBaseURL(), key, commentID)

	var result Comment
	if err := s.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetComments gets comments for an issue.
func (s *JiraService) GetComments(ctx context.Context, key string) ([]*Comment, error) {
	path := fmt.Sprintf("%s/issue/%s/comment", s.client.JiraBaseURL(), key)

	var result Comments
	if err := s.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return result.Comments, nil
}

// UpdateComment updates an existing comment.
func (s *JiraService) UpdateComment(ctx context.Context, key string, commentID string, opts *CommentOptions) (*Comment, error) {
	path := fmt.Sprintf("%s/issue/%s/comment/%s", s.client.JiraBaseURL(), key, commentID)

	req := &AddCommentRequest{
		Body: TextToADF(opts.Body),
	}

	if opts.VisibilityType != "" && opts.VisibilityName != "" {
		req.Visibility = &CommentVisibility{
			Type:  opts.VisibilityType,
			Value: opts.VisibilityName,
		}
	}

	var result Comment
	if err := s.client.Put(ctx, path, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteComment deletes a comment.
func (s *JiraService) DeleteComment(ctx context.Context, key string, commentID string) error {
	path := fmt.Sprintf("%s/issue/%s/comment/%s", s.client.JiraBaseURL(), key, commentID)
	return s.client.Delete(ctx, path)
}

// AssignIssue assigns an issue to a user.
func (s *JiraService) AssignIssue(ctx context.Context, key string, accountID string) error {
	path := fmt.Sprintf("%s/issue/%s/assignee", s.client.JiraBaseURL(), key)

	var body interface{}
	if accountID == "" {
		body = map[string]interface{}{"accountId": nil}
	} else {
		body = map[string]string{"accountId": accountID}
	}

	return s.client.Put(ctx, path, body, nil)
}

// GetMyself gets the current user.
func (s *JiraService) GetMyself(ctx context.Context) (*User, error) {
	path := fmt.Sprintf("%s/myself", s.client.JiraBaseURL())

	var user User
	if err := s.client.Get(ctx, path, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// SearchUsers searches for users.
func (s *JiraService) SearchUsers(ctx context.Context, query string) ([]*User, error) {
	path := fmt.Sprintf("%s/user/search", s.client.JiraBaseURL())

	params := url.Values{}
	params.Set("query", query)

	var users []*User
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &users); err != nil {
		return nil, err
	}

	return users, nil
}

// IssueLinkType represents a type of issue link.
type IssueLinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

// IssueLinkTypesResponse represents the response from getting link types.
type IssueLinkTypesResponse struct {
	IssueLinkTypes []*IssueLinkType `json:"issueLinkTypes"`
}

// GetIssueLinkTypes gets all available issue link types.
func (s *JiraService) GetIssueLinkTypes(ctx context.Context) ([]*IssueLinkType, error) {
	path := fmt.Sprintf("%s/issueLinkType", s.client.JiraBaseURL())

	var result IssueLinkTypesResponse
	if err := s.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return result.IssueLinkTypes, nil
}

// CreateIssueLinkRequest represents a request to create an issue link.
type CreateIssueLinkRequest struct {
	Type         *IssueLinkTypeID `json:"type"`
	InwardIssue  *IssueKeyID      `json:"inwardIssue"`
	OutwardIssue *IssueKeyID      `json:"outwardIssue"`
}

// IssueLinkTypeID identifies a link type by name.
type IssueLinkTypeID struct {
	Name string `json:"name"`
}

// IssueKeyID identifies an issue by key.
type IssueKeyID struct {
	Key string `json:"key"`
}

// CreateIssueLink creates a link between two issues.
func (s *JiraService) CreateIssueLink(ctx context.Context, inwardKey, outwardKey, linkTypeName string) error {
	path := fmt.Sprintf("%s/issueLink", s.client.JiraBaseURL())

	req := &CreateIssueLinkRequest{
		Type:         &IssueLinkTypeID{Name: linkTypeName},
		InwardIssue:  &IssueKeyID{Key: inwardKey},
		OutwardIssue: &IssueKeyID{Key: outwardKey},
	}

	return s.client.Post(ctx, path, req, nil)
}

// Field represents a Jira field definition.
type Field struct {
	ID          string       `json:"id"`
	Key         string       `json:"key"`
	Name        string       `json:"name"`
	Custom      bool         `json:"custom"`
	Orderable   bool         `json:"orderable"`
	Navigable   bool         `json:"navigable"`
	Searchable  bool         `json:"searchable"`
	Schema      *FieldSchema `json:"schema,omitempty"`
	ClauseNames []string     `json:"clauseNames,omitempty"`
}

// FieldSchema describes the type of a field.
type FieldSchema struct {
	Type     string `json:"type"`
	System   string `json:"system,omitempty"`
	Custom   string `json:"custom,omitempty"`
	CustomID int    `json:"customId,omitempty"`
}

// GetFields gets all field definitions.
func (s *JiraService) GetFields(ctx context.Context) ([]*Field, error) {
	path := fmt.Sprintf("%s/field", s.client.JiraBaseURL())

	var fields []*Field
	if err := s.client.Get(ctx, path, &fields); err != nil {
		return nil, err
	}

	return fields, nil
}

// GetFieldByName finds a field by name and returns it.
// Returns nil if not found.
func (s *JiraService) GetFieldByName(ctx context.Context, name string) (*Field, error) {
	fields, err := s.GetFields(ctx)
	if err != nil {
		return nil, err
	}

	nameLower := strings.ToLower(name)
	for _, f := range fields {
		if strings.ToLower(f.Name) == nameLower {
			return f, nil
		}
	}

	return nil, nil
}

// Sprint represents a Jira sprint.
type Sprint struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	State         string `json:"state"` // future, active, closed
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	OriginBoardID int    `json:"originBoardId,omitempty"`
	Goal          string `json:"goal,omitempty"`
}

// SprintsResponse represents a paginated list of sprints.
type SprintsResponse struct {
	MaxResults int       `json:"maxResults"`
	StartAt    int       `json:"startAt"`
	IsLast     bool      `json:"isLast"`
	Values     []*Sprint `json:"values"`
}

// Board represents a Jira board.
type Board struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // scrum, kanban
	Location *BoardLocation `json:"location,omitempty"`
}

// BoardLocation represents the location of a board.
type BoardLocation struct {
	ProjectID  int    `json:"projectId"`
	ProjectKey string `json:"projectKey"`
	Name       string `json:"displayName"`
}

// BoardsResponse represents a paginated list of boards.
type BoardsResponse struct {
	MaxResults int      `json:"maxResults"`
	StartAt    int      `json:"startAt"`
	IsLast     bool     `json:"isLast"`
	Values     []*Board `json:"values"`
}

// GetBoards gets all boards, optionally filtered by project.
func (s *JiraService) GetBoards(ctx context.Context, projectKey string) ([]*Board, error) {
	path := fmt.Sprintf("%s/board", s.client.AgileBaseURL())

	params := url.Values{}
	if projectKey != "" {
		params.Set("projectKeyOrId", projectKey)
	}
	params.Set("maxResults", "100")

	var result BoardsResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return result.Values, nil
}

// GetSprints gets sprints for a board.
func (s *JiraService) GetSprints(ctx context.Context, boardID int, state string) ([]*Sprint, error) {
	path := fmt.Sprintf("%s/board/%d/sprint", s.client.AgileBaseURL(), boardID)

	params := url.Values{}
	if state != "" {
		params.Set("state", state)
	}
	params.Set("maxResults", "100")

	var result SprintsResponse
	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return result.Values, nil
}

// MoveIssuesToSprint moves issues to a sprint.
func (s *JiraService) MoveIssuesToSprint(ctx context.Context, sprintID int, issueKeys []string) error {
	path := fmt.Sprintf("%s/sprint/%d/issue", s.client.AgileBaseURL(), sprintID)

	body := map[string]interface{}{
		"issues": issueKeys,
	}

	return s.client.Post(ctx, path, body, nil)
}

// RemoveIssuesFromSprint moves issues to the backlog (removes from sprint).
func (s *JiraService) RemoveIssuesFromSprint(ctx context.Context, issueKeys []string) error {
	path := fmt.Sprintf("%s/backlog/issue", s.client.AgileBaseURL())

	body := map[string]interface{}{
		"issues": issueKeys,
	}

	return s.client.Post(ctx, path, body, nil)
}

// TextToADF converts plain text to Atlassian Document Format.
func TextToADF(text string) *ADF {
	paragraphs := strings.Split(text, "\n\n")
	content := make([]ADFContent, 0, len(paragraphs))

	for _, para := range paragraphs {
		if para == "" {
			continue
		}
		content = append(content, ADFContent{
			Type: "paragraph",
			Content: []ADFContent{
				{Type: "text", Text: para},
			},
		})
	}

	return &ADF{
		Type:    "doc",
		Version: 1,
		Content: content,
	}
}

// ADFToText converts Atlassian Document Format to plain text.
func ADFToText(adf *ADF) string {
	if adf == nil {
		return ""
	}
	var sb strings.Builder
	adfContentToText(&sb, adf.Content)
	return strings.TrimSpace(sb.String())
}

func adfContentToText(sb *strings.Builder, content []ADFContent) {
	for i, c := range content {
		switch c.Type {
		case "text":
			sb.WriteString(c.Text)
		case "paragraph":
			if i > 0 {
				sb.WriteString("\n\n")
			}
			adfContentToText(sb, c.Content)
		case "heading":
			if i > 0 {
				sb.WriteString("\n\n")
			}
			adfContentToText(sb, c.Content)
			sb.WriteString("\n")
		case "bulletList", "orderedList":
			if i > 0 {
				sb.WriteString("\n")
			}
			adfContentToText(sb, c.Content)
		case "listItem":
			sb.WriteString("• ")
			adfContentToText(sb, c.Content)
			sb.WriteString("\n")
		case "codeBlock":
			sb.WriteString("\n```\n")
			adfContentToText(sb, c.Content)
			sb.WriteString("\n```\n")
		case "hardBreak":
			sb.WriteString("\n")
		default:
			adfContentToText(sb, c.Content)
		}
	}
}
