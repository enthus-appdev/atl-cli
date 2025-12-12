package issue

import (
	"bytes"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// TestFormatTime tests the time formatting function.
func TestFormatTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "Jira format",
			input: "2024-01-15T10:30:00.000+0000",
			want:  "2024-01-15 10:30:00",
		},
		{
			name:  "RFC3339 format",
			input: "2024-01-15T10:30:00Z",
			want:  "2024-01-15 10:30:00",
		},
		{
			name:  "invalid format returns original",
			input: "not-a-date",
			want:  "not-a-date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.input)
			if got != tt.want {
				t.Errorf("formatTime(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestFormatIssueOutput tests the issue output formatter.
func TestFormatIssueOutput(t *testing.T) {
	issue := &api.Issue{
		ID:  "10001",
		Key: "TEST-123",
		Fields: api.IssueFields{
			Summary: "Test Summary",
			Status: &api.Status{
				ID:   "1",
				Name: "To Do",
				StatusCategory: &api.StatusCategory{
					ID:  1,
					Key: "new",
				},
			},
			Priority: &api.Priority{
				ID:   "3",
				Name: "Medium",
			},
			IssueType: &api.IssueType{
				ID:   "10001",
				Name: "Task",
			},
			Assignee: &api.User{
				AccountID:    "user-123",
				DisplayName:  "John Doe",
				EmailAddress: "john@example.com",
			},
			Reporter: &api.User{
				AccountID:   "user-456",
				DisplayName: "Jane Doe",
			},
			Project: &api.Project{
				ID:   "10000",
				Key:  "TEST",
				Name: "Test Project",
			},
			Labels:  []string{"bug", "urgent"},
			Created: "2024-01-15T10:00:00.000+0000",
			Updated: "2024-01-16T14:30:00.000+0000",
		},
	}

	hostname := "example.atlassian.net"
	output := formatIssueOutput(issue, hostname)

	// Verify basic fields
	if output.Key != "TEST-123" {
		t.Errorf("Key = %q, want %q", output.Key, "TEST-123")
	}
	if output.ID != "10001" {
		t.Errorf("ID = %q, want %q", output.ID, "10001")
	}
	if output.Summary != "Test Summary" {
		t.Errorf("Summary = %q, want %q", output.Summary, "Test Summary")
	}
	if output.Status != "To Do" {
		t.Errorf("Status = %q, want %q", output.Status, "To Do")
	}
	if output.StatusCategory != "new" {
		t.Errorf("StatusCategory = %q, want %q", output.StatusCategory, "new")
	}
	if output.Priority != "Medium" {
		t.Errorf("Priority = %q, want %q", output.Priority, "Medium")
	}
	if output.Type != "Task" {
		t.Errorf("Type = %q, want %q", output.Type, "Task")
	}

	// Verify user fields
	if output.Assignee == nil {
		t.Error("Assignee should not be nil")
	} else {
		if output.Assignee.DisplayName != "John Doe" {
			t.Errorf("Assignee.DisplayName = %q, want %q", output.Assignee.DisplayName, "John Doe")
		}
		if output.Assignee.Email != "john@example.com" {
			t.Errorf("Assignee.Email = %q, want %q", output.Assignee.Email, "john@example.com")
		}
	}

	if output.Reporter == nil {
		t.Error("Reporter should not be nil")
	} else if output.Reporter.DisplayName != "Jane Doe" {
		t.Errorf("Reporter.DisplayName = %q, want %q", output.Reporter.DisplayName, "Jane Doe")
	}

	// Verify project
	if output.Project == nil {
		t.Error("Project should not be nil")
	} else {
		if output.Project.Key != "TEST" {
			t.Errorf("Project.Key = %q, want %q", output.Project.Key, "TEST")
		}
	}

	// Verify labels
	if len(output.Labels) != 2 {
		t.Errorf("Labels count = %d, want 2", len(output.Labels))
	}

	// Verify URL
	expectedURL := "https://example.atlassian.net/browse/TEST-123"
	if output.URL != expectedURL {
		t.Errorf("URL = %q, want %q", output.URL, expectedURL)
	}
}

// TestFormatIssueOutputMinimal tests formatter with minimal issue data.
func TestFormatIssueOutputMinimal(t *testing.T) {
	issue := &api.Issue{
		ID:  "10001",
		Key: "TEST-1",
		Fields: api.IssueFields{
			Summary: "Minimal Issue",
		},
	}

	output := formatIssueOutput(issue, "example.atlassian.net")

	if output.Key != "TEST-1" {
		t.Errorf("Key = %q, want %q", output.Key, "TEST-1")
	}
	if output.Summary != "Minimal Issue" {
		t.Errorf("Summary = %q, want %q", output.Summary, "Minimal Issue")
	}
	if output.Assignee != nil {
		t.Error("Assignee should be nil for minimal issue")
	}
	if output.Reporter != nil {
		t.Error("Reporter should be nil for minimal issue")
	}
	if output.Status != "" {
		t.Errorf("Status should be empty for minimal issue, got %q", output.Status)
	}
}

// TestPrintIssueDetails tests the text output formatter.
func TestPrintIssueDetails(t *testing.T) {
	outBuf := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		Out: outBuf,
	}

	issueOutput := &IssueOutput{
		Key:         "TEST-123",
		Summary:     "Test Issue",
		Type:        "Task",
		Status:      "To Do",
		Priority:    "High",
		Project:     &ProjectOutput{Key: "TEST", Name: "Test Project"},
		Assignee:    &UserOutput{DisplayName: "John Doe"},
		Reporter:    &UserOutput{DisplayName: "Jane Doe"},
		Labels:      []string{"bug"},
		Created:     "2024-01-15 10:00:00",
		Updated:     "2024-01-16 14:30:00",
		URL:         "https://example.atlassian.net/browse/TEST-123",
		Description: "This is the description.",
	}

	printIssueDetails(ios, issueOutput)

	output := outBuf.String()

	// Check for expected content
	expectedStrings := []string{
		"# TEST-123: Test Issue",
		"Type: Task",
		"Status: To Do",
		"Priority: High",
		"Project: Test Project (TEST)",
		"Assignee: John Doe",
		"Reporter: Jane Doe",
		"Labels: bug",
		"URL: https://example.atlassian.net/browse/TEST-123",
		"## Description",
		"This is the description.",
	}

	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("Output missing %q\nGot: %s", expected, output)
		}
	}
}

// TestPrintIssueDetailsUnassigned tests output when issue is unassigned.
func TestPrintIssueDetailsUnassigned(t *testing.T) {
	outBuf := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		Out: outBuf,
	}

	issueOutput := &IssueOutput{
		Key:      "TEST-123",
		Summary:  "Unassigned Issue",
		Type:     "Task",
		Status:   "Open",
		Assignee: nil,
		URL:      "https://example.atlassian.net/browse/TEST-123",
	}

	printIssueDetails(ios, issueOutput)

	output := outBuf.String()
	if !contains(output, "Assignee: Unassigned") {
		t.Errorf("Output should show 'Assignee: Unassigned'\nGot: %s", output)
	}
}

// TestNewCmdView tests the command creation.
func TestNewCmdView(t *testing.T) {
	ios := iostreams.Test()
	cmd := NewCmdView(ios)

	if cmd == nil {
		t.Fatal("NewCmdView() returned nil")
	}
	if cmd.Use != "view <issue-key>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "view <issue-key>")
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Check flags exist
	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("--json flag should exist")
	}
	webFlag := cmd.Flags().Lookup("web")
	if webFlag == nil {
		t.Error("--web flag should exist")
	}
}

// TestViewOptions tests the ViewOptions struct.
func TestViewOptions(t *testing.T) {
	ios := iostreams.Test()
	opts := &ViewOptions{
		IO:       ios,
		IssueKey: "TEST-123",
		JSON:     true,
		Web:      false,
	}

	if opts.IO == nil {
		t.Error("IO should not be nil")
	}
	if opts.IssueKey != "TEST-123" {
		t.Errorf("IssueKey = %q, want %q", opts.IssueKey, "TEST-123")
	}
	if !opts.JSON {
		t.Error("JSON should be true")
	}
	if opts.Web {
		t.Error("Web should be false")
	}
}

// helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
