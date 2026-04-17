package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// SMService handles Jira Service Management API operations.
type SMService struct {
	client *Client
}

// NewSMService creates a new Service Management service.
func NewSMService(client *Client) *SMService {
	return &SMService{client: client}
}

// SMBaseURL returns the base URL for JSM API requests.
func (c *Client) SMBaseURL() string {
	return fmt.Sprintf("%s/ex/jira/%s/rest/servicedeskapi", AtlassianAPIURL, c.cloudID)
}

// ServiceDesk represents a Jira Service Management service desk.
type ServiceDesk struct {
	ID          string `json:"id"`
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
	ProjectKey  string `json:"projectKey"`
}

// ServiceDeskPage represents a paginated list of service desks.
type ServiceDeskPage struct {
	Size       int            `json:"size"`
	Start      int            `json:"start"`
	Limit      int            `json:"limit"`
	IsLastPage bool           `json:"isLastPage"`
	Values     []*ServiceDesk `json:"values"`
}

// GetServiceDesks lists all service desks.
func (s *SMService) GetServiceDesks(ctx context.Context) ([]*ServiceDesk, error) {
	path := fmt.Sprintf("%s/servicedesk", s.client.SMBaseURL())

	var result ServiceDeskPage
	if err := s.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return result.Values, nil
}

// RequestType represents a request type in a service desk.
type RequestType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ServiceDesk *struct {
		ID string `json:"id"`
	} `json:"serviceDeskId,omitempty"`
}

// RequestTypePage represents a paginated list of request types.
type RequestTypePage struct {
	Size       int            `json:"size"`
	Start      int            `json:"start"`
	Limit      int            `json:"limit"`
	IsLastPage bool           `json:"isLastPage"`
	Values     []*RequestType `json:"values"`
}

// GetRequestTypes lists request types for a service desk.
func (s *SMService) GetRequestTypes(ctx context.Context, serviceDeskID int) ([]*RequestType, error) {
	path := fmt.Sprintf("%s/servicedesk/%d/requesttype", s.client.SMBaseURL(), serviceDeskID)

	var result RequestTypePage
	if err := s.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return result.Values, nil
}

// RequestTypeField represents a field in a request type.
type RequestTypeField struct {
	FieldID      string              `json:"fieldId"`
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	Required     bool                `json:"required"`
	DefaultValue []json.RawMessage   `json:"defaultValues"`
	ValidValues  []*RequestTypeValue `json:"validValues"`
	PresetValues []string            `json:"presetValues"`
	JiraSchema   *RequestTypeSchema  `json:"jiraSchema,omitempty"`
	Visible      bool                `json:"visible"`
}

// RequestTypeValue represents a valid value for a request type field.
type RequestTypeValue struct {
	Value    string              `json:"value"`
	Label    string              `json:"label"`
	Children []*RequestTypeValue `json:"children"`
}

// RequestTypeSchema represents the Jira schema of a request type field.
type RequestTypeSchema struct {
	Type          string                 `json:"type"`
	Items         string                 `json:"items,omitempty"`
	System        string                 `json:"system,omitempty"`
	Custom        string                 `json:"custom,omitempty"`
	CustomID      int                    `json:"customId,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
}

// RequestTypeFieldsResponse represents the response from the request type fields endpoint.
type RequestTypeFieldsResponse struct {
	RequestTypeFields         []*RequestTypeField `json:"requestTypeFields"`
	CanRaiseOnBehalfOf        bool                `json:"canRaiseOnBehalfOf"`
	CanAddRequestParticipants bool                `json:"canAddRequestParticipants"`
}

// GetRequestTypeFields gets the fields for a request type.
func (s *SMService) GetRequestTypeFields(ctx context.Context, serviceDeskID, requestTypeID int) (*RequestTypeFieldsResponse, error) {
	path := fmt.Sprintf("%s/servicedesk/%d/requesttype/%d/field",
		s.client.SMBaseURL(), serviceDeskID, requestTypeID)

	var result RequestTypeFieldsResponse
	if err := s.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
