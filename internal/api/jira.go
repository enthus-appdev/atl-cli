package api

import (
	"context"
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
	Project     *ProjectID   `json:"project"`
	Summary     string       `json:"summary"`
	Description *ADF         `json:"description,omitempty"`
	IssueType   *IssueTypeID `json:"issuetype"`
	Assignee    *AccountID   `json:"assignee,omitempty"`
	Priority    *PriorityID  `json:"priority,omitempty"`
	Labels      []string     `json:"labels,omitempty"`
	Parent      *ParentID    `json:"parent,omitempty"`
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

// AddCommentRequest represents a request to add a comment.
type AddCommentRequest struct {
	Body *ADF `json:"body"`
}

// AddComment adds a comment to an issue.
func (s *JiraService) AddComment(ctx context.Context, key string, body string) (*Comment, error) {
	path := fmt.Sprintf("%s/issue/%s/comment", s.client.JiraBaseURL(), key)

	req := &AddCommentRequest{
		Body: TextToADF(body),
	}

	var result Comment
	if err := s.client.Post(ctx, path, req, &result); err != nil {
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
