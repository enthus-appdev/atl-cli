package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/jcstorino/jira-cli/pkg/adf"
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
	Summary     string        `json:"summary"`
	Description *ADF          `json:"description,omitempty"`
	Status      *Status       `json:"status,omitempty"`
	Priority    *Priority     `json:"priority,omitempty"`
	IssueType   *IssueType    `json:"issuetype,omitempty"`
	Assignee    *User         `json:"assignee,omitempty"`
	Reporter    *User         `json:"reporter,omitempty"`
	Project     *Project      `json:"project,omitempty"`
	Labels      []string      `json:"labels,omitempty"`
	Created     string        `json:"created,omitempty"`
	Updated     string        `json:"updated,omitempty"`
	Resolution  *Resolution   `json:"resolution,omitempty"`
	Components  []*Component  `json:"components,omitempty"`
	Comment     *Comments     `json:"comment,omitempty"`
	Parent      *Issue        `json:"parent,omitempty"`
	Attachment  []*Attachment `json:"attachment,omitempty"`
}

// Attachment represents an attachment on an issue.
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Author   *User  `json:"author,omitempty"`
	Created  string `json:"created"`
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Content  string `json:"content"` // URL to download the attachment
}

// ADF represents Atlassian Document Format content.
type ADF struct {
	Type    string       `json:"type"`
	Version int          `json:"version,omitempty"`
	Content []ADFContent `json:"content,omitempty"`
	Text    string       `json:"text,omitempty"`
	Attrs   *ADFAttrs    `json:"attrs,omitempty"`
	Marks   []ADFMark    `json:"marks,omitempty"`
}

// ADFContent represents content within an ADF document.
type ADFContent struct {
	Type    string       `json:"type"`
	Content []ADFContent `json:"content,omitempty"`
	Text    string       `json:"text,omitempty"`
	Attrs   *ADFAttrs    `json:"attrs,omitempty"`
	Marks   []ADFMark    `json:"marks,omitempty"`
}

// ADFAttrs represents attributes in ADF.
type ADFAttrs struct {
	Level    int    `json:"level,omitempty"`
	URL      string `json:"url,omitempty"`
	Href     string `json:"href,omitempty"`
	Language string `json:"language,omitempty"`
	// Media attributes
	ID         string `json:"id,omitempty"`
	Type       string `json:"type,omitempty"`
	Collection string `json:"collection,omitempty"`
	Alt        string `json:"alt,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	// Panel attributes
	PanelType string `json:"panelType,omitempty"`
	// Expand attributes
	Title string `json:"title,omitempty"`
	// Table attributes
	Layout string `json:"layout,omitempty"`
	// Table cell attributes
	Colspan  int `json:"colspan,omitempty"`
	Rowspan  int `json:"rowspan,omitempty"`
	Colwidth []int `json:"colwidth,omitempty"`
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
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IconURL     string `json:"iconUrl,omitempty"`
	StatusColor string `json:"statusColor,omitempty"`
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
	AccountID    string            `json:"accountId"`
	DisplayName  string            `json:"displayName"`
	EmailAddress string            `json:"emailAddress,omitempty"`
	Active       bool              `json:"active"`
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

// GetAttachment gets attachment metadata by ID.
func (s *JiraService) GetAttachment(ctx context.Context, attachmentID string) (*Attachment, error) {
	path := fmt.Sprintf("%s/attachment/%s", s.client.JiraBaseURL(), attachmentID)

	var attachment Attachment
	if err := s.client.Get(ctx, path, &attachment); err != nil {
		return nil, err
	}

	return &attachment, nil
}

// DownloadAttachment downloads an attachment and returns its content.
func (s *JiraService) DownloadAttachment(ctx context.Context, attachmentID string) ([]byte, string, error) {
	path := fmt.Sprintf("%s/attachment/content/%s", s.client.JiraBaseURL(), attachmentID)

	return s.client.GetRaw(ctx, path)
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

// ProjectIssueType represents an issue type available in a project.
type ProjectIssueType struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Subtask        bool   `json:"subtask"`
	HierarchyLevel int    `json:"hierarchyLevel,omitempty"`
}

// ProjectIssueTypesResponse represents the response from createmeta endpoint.
type ProjectIssueTypesResponse struct {
	IssueTypes []*ProjectIssueType `json:"issueTypes"`
}

// GetProjectIssueTypes gets the available issue types for a project.
func (s *JiraService) GetProjectIssueTypes(ctx context.Context, projectKey string) ([]*ProjectIssueType, error) {
	path := fmt.Sprintf("%s/issue/createmeta/%s/issuetypes", s.client.JiraBaseURL(), projectKey)

	var result ProjectIssueTypesResponse
	if err := s.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return result.IssueTypes, nil
}

// GetSubtaskType finds the subtask issue type for a project.
// Returns the first issue type where subtask=true.
func (s *JiraService) GetSubtaskType(ctx context.Context, projectKey string) (*ProjectIssueType, error) {
	types, err := s.GetProjectIssueTypes(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	for _, t := range types {
		if t.Subtask {
			return t, nil
		}
	}

	return nil, nil
}

// GetPriorities gets all available priorities in the Jira instance.
func (s *JiraService) GetPriorities(ctx context.Context) ([]*Priority, error) {
	path := fmt.Sprintf("%s/priority", s.client.JiraBaseURL())

	var result []*Priority
	if err := s.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return result, nil
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

// RemoteLink represents a remote/web link on an issue.
type RemoteLink struct {
	ID           int               `json:"id"`
	Self         string            `json:"self,omitempty"`
	GlobalID     string            `json:"globalId,omitempty"`
	Application  *RemoteLinkApp    `json:"application,omitempty"`
	Relationship string            `json:"relationship,omitempty"`
	Object       *RemoteLinkObject `json:"object"`
}

// RemoteLinkApp represents the application info for a remote link.
type RemoteLinkApp struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

// RemoteLinkObject represents the linked object details.
type RemoteLinkObject struct {
	URL     string            `json:"url"`
	Title   string            `json:"title"`
	Summary string            `json:"summary,omitempty"`
	Icon    *RemoteLinkIcon   `json:"icon,omitempty"`
	Status  *RemoteLinkStatus `json:"status,omitempty"`
}

// RemoteLinkIcon represents an icon for a remote link.
type RemoteLinkIcon struct {
	URL16x16 string `json:"url16x16,omitempty"`
	Title    string `json:"title,omitempty"`
}

// RemoteLinkStatus represents the status of a remote link.
type RemoteLinkStatus struct {
	Resolved bool            `json:"resolved,omitempty"`
	Icon     *RemoteLinkIcon `json:"icon,omitempty"`
}

// GetRemoteLinks gets all remote/web links for an issue.
func (s *JiraService) GetRemoteLinks(ctx context.Context, issueKey string) ([]*RemoteLink, error) {
	path := fmt.Sprintf("%s/issue/%s/remotelink", s.client.JiraBaseURL(), issueKey)

	var links []*RemoteLink
	if err := s.client.Get(ctx, path, &links); err != nil {
		return nil, err
	}

	return links, nil
}

// CreateRemoteLinkRequest represents a request to create a remote link.
type CreateRemoteLinkRequest struct {
	GlobalID     string            `json:"globalId,omitempty"`
	Application  *RemoteLinkApp    `json:"application,omitempty"`
	Relationship string            `json:"relationship,omitempty"`
	Object       *RemoteLinkObject `json:"object"`
}

// CreateRemoteLink creates a remote/web link on an issue.
func (s *JiraService) CreateRemoteLink(ctx context.Context, issueKey, url, title, summary string) (*RemoteLink, error) {
	path := fmt.Sprintf("%s/issue/%s/remotelink", s.client.JiraBaseURL(), issueKey)

	req := &CreateRemoteLinkRequest{
		Object: &RemoteLinkObject{
			URL:     url,
			Title:   title,
			Summary: summary,
		},
	}

	var result RemoteLink
	if err := s.client.Post(ctx, path, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteRemoteLink deletes a remote/web link from an issue.
func (s *JiraService) DeleteRemoteLink(ctx context.Context, issueKey string, linkID int) error {
	path := fmt.Sprintf("%s/issue/%s/remotelink/%d", s.client.JiraBaseURL(), issueKey, linkID)
	return s.client.Delete(ctx, path)
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

// GetFlaggedField finds the "Flagged" custom field.
// Returns the field or nil if not found.
func (s *JiraService) GetFlaggedField(ctx context.Context) (*Field, error) {
	fields, err := s.GetFields(ctx)
	if err != nil {
		return nil, err
	}

	for _, f := range fields {
		// The Flagged field has untranslatedName or name "Flagged"
		if f.Name == "Flagged" || strings.EqualFold(f.Name, "Flagged") {
			return f, nil
		}
	}

	return nil, nil
}

// FlagIssue flags an issue (adds the Impediment flag).
func (s *JiraService) FlagIssue(ctx context.Context, issueKey string) error {
	// First, find the Flagged field ID
	field, err := s.GetFlaggedField(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Flagged field: %w", err)
	}
	if field == nil {
		return fmt.Errorf("Flagged field not found. Make sure the Flagged field is available in your Jira instance")
	}

	path := fmt.Sprintf("%s/issue/%s", s.client.JiraBaseURL(), issueKey)

	body := map[string]interface{}{
		"fields": map[string]interface{}{
			field.ID: []map[string]string{
				{"value": "Impediment"},
			},
		},
	}

	return s.client.Put(ctx, path, body, nil)
}

// UnflagIssue removes the flag from an issue.
func (s *JiraService) UnflagIssue(ctx context.Context, issueKey string) error {
	// First, find the Flagged field ID
	field, err := s.GetFlaggedField(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Flagged field: %w", err)
	}
	if field == nil {
		return fmt.Errorf("Flagged field not found. Make sure the Flagged field is available in your Jira instance")
	}

	path := fmt.Sprintf("%s/issue/%s", s.client.JiraBaseURL(), issueKey)

	body := map[string]interface{}{
		"fields": map[string]interface{}{
			field.ID: []map[string]string{},
		},
	}

	return s.client.Put(ctx, path, body, nil)
}

// IsIssueFlagged checks if an issue is flagged.
func (s *JiraService) IsIssueFlagged(ctx context.Context, issueKey string) (bool, error) {
	// First, find the Flagged field ID
	field, err := s.GetFlaggedField(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get Flagged field: %w", err)
	}
	if field == nil {
		return false, nil // No flagged field means can't be flagged
	}

	// Get the issue with the flagged field
	path := fmt.Sprintf("%s/issue/%s?fields=%s", s.client.JiraBaseURL(), issueKey, field.ID)

	var result struct {
		Fields map[string]interface{} `json:"fields"`
	}

	if err := s.client.Get(ctx, path, &result); err != nil {
		return false, err
	}

	// Check if the field has a value
	if result.Fields == nil {
		return false, nil
	}

	flagValue := result.Fields[field.ID]
	if flagValue == nil {
		return false, nil
	}

	// The field value is an array of objects with "value" keys
	if arr, ok := flagValue.([]interface{}); ok {
		return len(arr) > 0, nil
	}

	return false, nil
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
	ID       int            `json:"id"`
	Name     string         `json:"name"`
	Type     string         `json:"type"` // scrum, kanban
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

// RankIssuesBefore ranks issues before a target issue.
// The issues will be placed directly before rankBeforeIssue in the backlog/board order.
func (s *JiraService) RankIssuesBefore(ctx context.Context, issueKeys []string, rankBeforeIssue string) error {
	path := fmt.Sprintf("%s/issue/rank", s.client.AgileBaseURL())

	body := map[string]interface{}{
		"issues":          issueKeys,
		"rankBeforeIssue": rankBeforeIssue,
	}

	return s.client.Put(ctx, path, body, nil)
}

// RankIssuesAfter ranks issues after a target issue.
// The issues will be placed directly after rankAfterIssue in the backlog/board order.
func (s *JiraService) RankIssuesAfter(ctx context.Context, issueKeys []string, rankAfterIssue string) error {
	path := fmt.Sprintf("%s/issue/rank", s.client.AgileBaseURL())

	body := map[string]interface{}{
		"issues":         issueKeys,
		"rankAfterIssue": rankAfterIssue,
	}

	return s.client.Put(ctx, path, body, nil)
}

// RankIssuesToTop ranks issues to the top of the backlog.
func (s *JiraService) RankIssuesToTop(ctx context.Context, issueKeys []string, boardID int) error {
	// Get the first issue on the board to rank before it
	path := fmt.Sprintf("%s/board/%d/issue", s.client.AgileBaseURL(), boardID)

	params := url.Values{}
	params.Set("maxResults", "1")

	var result struct {
		Issues []struct {
			Key string `json:"key"`
		} `json:"issues"`
	}

	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return err
	}

	if len(result.Issues) == 0 {
		// No issues on board, nothing to rank against
		return nil
	}

	// If the first issue is already one we're ranking, we're done
	for _, key := range issueKeys {
		if key == result.Issues[0].Key {
			return nil
		}
	}

	return s.RankIssuesBefore(ctx, issueKeys, result.Issues[0].Key)
}

// GetBoardIssues gets issues on a board.
func (s *JiraService) GetBoardIssues(ctx context.Context, boardID int, maxResults int) ([]*Issue, error) {
	path := fmt.Sprintf("%s/board/%d/issue", s.client.AgileBaseURL(), boardID)

	params := url.Values{}
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	} else {
		params.Set("maxResults", "50")
	}

	var result struct {
		Issues []*Issue `json:"issues"`
	}

	if err := s.client.Get(ctx, path+"?"+params.Encode(), &result); err != nil {
		return nil, err
	}

	return result.Issues, nil
}

// TextToADF converts plain text or markdown to Atlassian Document Format.
// Supports markdown syntax including: headings (#), bold (**), italic (*),
// inline code (`), code blocks (```), links, bullet lists (-/*), ordered lists,
// blockquotes (>), and horizontal rules (---).
func TextToADF(text string) *ADF {
	return MarkdownToADF(text)
}

// ADFToText converts Atlassian Document Format to Markdown text.
// Uses the jira-cli adf library for proper Markdown formatting.
func ADFToText(ourADF *ADF) string {
	if ourADF == nil {
		return ""
	}

	// Convert our ADF type to the library's ADF type
	libADF := convertToLibraryADF(ourADF)
	if libADF == nil || len(libADF.Content) == 0 {
		return ""
	}

	// Use the library's Markdown translator
	translator := adf.NewTranslator(libADF, adf.NewMarkdownTranslator())
	result := translator.Translate()

	return strings.TrimSpace(result)
}

// convertToLibraryADF converts our ADF type to the jira-cli library's ADF type.
func convertToLibraryADF(ourADF *ADF) *adf.ADF {
	if ourADF == nil {
		return nil
	}

	return &adf.ADF{
		Version: ourADF.Version,
		DocType: ourADF.Type,
		Content: convertNodes(ourADF.Content),
	}
}

// convertNodes converts our ADFContent slice to the library's Node slice.
func convertNodes(content []ADFContent) []*adf.Node {
	if len(content) == 0 {
		return nil
	}

	nodes := make([]*adf.Node, 0, len(content))
	for _, c := range content {
		node := convertNode(c)
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// convertNode converts a single ADFContent to the library's Node.
func convertNode(c ADFContent) *adf.Node {
	// Handle media nodes specially - convert to text with descriptive placeholder
	if c.Type == "media" {
		altText := "[Embedded image]"
		if c.Attrs != nil && c.Attrs.Alt != "" {
			altText = fmt.Sprintf("[Image: %s]", c.Attrs.Alt)
		}
		return &adf.Node{
			NodeType: adf.NodeType("text"),
			NodeValue: adf.NodeValue{
				Text: altText,
			},
		}
	}

	node := &adf.Node{
		NodeType: adf.NodeType(c.Type),
		Content:  convertNodes(c.Content),
		NodeValue: adf.NodeValue{
			Text:  c.Text,
			Marks: convertMarks(c.Marks),
		},
	}

	// Convert attributes
	if c.Attrs != nil {
		node.Attributes = convertAttrs(c.Attrs)
	}

	return node
}

// convertMarks converts our ADFMark slice to the library's MarkNode slice.
func convertMarks(marks []ADFMark) []adf.MarkNode {
	if len(marks) == 0 {
		return nil
	}

	result := make([]adf.MarkNode, 0, len(marks))
	for _, m := range marks {
		markNode := adf.MarkNode{
			MarkType: adf.NodeType(m.Type),
		}
		if m.Attrs != nil {
			markNode.Attributes = convertAttrs(m.Attrs)
		}
		result = append(result, markNode)
	}
	return result
}

// convertAttrs converts our ADFAttrs to a map for the library.
func convertAttrs(attrs *ADFAttrs) map[string]interface{} {
	if attrs == nil {
		return nil
	}

	result := make(map[string]interface{})

	if attrs.Level > 0 {
		result["level"] = attrs.Level
	}
	if attrs.URL != "" {
		result["url"] = attrs.URL
	}
	if attrs.Href != "" {
		result["href"] = attrs.Href
	}
	if attrs.Language != "" {
		result["language"] = attrs.Language
	}
	if attrs.ID != "" {
		result["id"] = attrs.ID
	}
	if attrs.Type != "" {
		result["type"] = attrs.Type
	}
	if attrs.Collection != "" {
		result["collection"] = attrs.Collection
	}
	if attrs.Alt != "" {
		result["alt"] = attrs.Alt
	}
	if attrs.Width > 0 {
		result["width"] = attrs.Width
	}
	if attrs.Height > 0 {
		result["height"] = attrs.Height
	}
	// Panel attributes
	if attrs.PanelType != "" {
		result["panelType"] = attrs.PanelType
	}
	// Expand attributes
	if attrs.Title != "" {
		result["title"] = attrs.Title
	}
	// Table attributes
	if attrs.Layout != "" {
		result["layout"] = attrs.Layout
	}
	// Table cell attributes
	if attrs.Colspan > 0 {
		result["colspan"] = attrs.Colspan
	}
	if attrs.Rowspan > 0 {
		result["rowspan"] = attrs.Rowspan
	}
	if len(attrs.Colwidth) > 0 {
		result["colwidth"] = attrs.Colwidth
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
