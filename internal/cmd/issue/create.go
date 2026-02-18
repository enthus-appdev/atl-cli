package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// CreateOptions holds the options for the create command.
type CreateOptions struct {
	IO           *iostreams.IOStreams
	Project      string
	IssueType    string
	Summary      string
	Description  string
	Assignee     string
	Labels       []string
	Priority     string
	Parent       string
	CustomFields []string
	FieldFile    string
	Web          bool
	JSON         bool
}

// NewCmdCreate creates the create command.
func NewCmdCreate(ios *iostreams.IOStreams) *cobra.Command {
	opts := &CreateOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Jira issue",
		Long:  `Create a new Jira issue in a project.`,
		Example: `  # Create a bug
  atl issue create --project PROJ --type Bug --summary "Fix login issue"

  # Create a task with description
  atl issue create --project PROJ --type Task --summary "New feature" --description "Implement new feature"

  # Create and open in browser
  atl issue create --project PROJ --type Task --summary "New feature" --web

  # Create a subtask (auto-discovers subtask type)
  atl issue create --project PROJ --parent PROJ-123 --summary "Subtask"

  # Or specify the subtask type explicitly
  atl issue create --project PROJ --type "Sub-task" --parent PROJ-123 --summary "Subtask"

  # Create with custom fields by name (Story Points, etc.)
  atl issue create --project PROJ --type Story --summary "New story" --field "Story Points=5"

  # Or use field ID directly
  atl issue create --project PROJ --type Story --summary "New story" --field customfield_10016=5

  # Use a JSON file for complex field values (like ADF rich text)
  atl issue create --project PROJ --type Task --summary "Task" --field-file fields.json

  # Output as JSON
  atl issue create --project PROJ --type Bug --summary "Bug report" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var missing []string
			if opts.Project == "" {
				missing = append(missing, "--project")
			}
			// --type is optional if --parent is provided (auto-discovers subtask type)
			if opts.IssueType == "" && opts.Parent == "" {
				missing = append(missing, "--type")
			}
			if opts.Summary == "" {
				missing = append(missing, "--summary")
			}
			if len(missing) > 0 {
				return fmt.Errorf("required flags not set: %v\n\nExample: atl issue create --project PROJ --type Bug --summary \"Issue title\"", missing)
			}
			return runCreate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "Project key (required)")
	cmd.Flags().StringVarP(&opts.IssueType, "type", "t", "", "Issue type (e.g., Bug, Task, Story) (required)")
	cmd.Flags().StringVarP(&opts.Summary, "summary", "s", "", "Issue summary (required)")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Issue description")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "Assignee (use @me for yourself)")
	cmd.Flags().StringSliceVarP(&opts.Labels, "label", "l", nil, "Labels to add")
	cmd.Flags().StringVar(&opts.Priority, "priority", "", "Priority level")
	cmd.Flags().StringVar(&opts.Parent, "parent", "", "Parent issue key (for subtasks)")
	cmd.Flags().StringSliceVarP(&opts.CustomFields, "field", "f", nil, "Custom field in key=value format (can be repeated)")
	cmd.Flags().StringVar(&opts.FieldFile, "field-file", "", "JSON file with field values (for complex types like ADF)")
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open created issue in browser")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// CreateOutput represents the output after creating an issue.
type CreateOutput struct {
	Key     string `json:"key"`
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Type    string `json:"type"`
	Project string `json:"project"`
	URL     string `json:"url"`
}

func runCreate(opts *CreateOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Resolve @me assignee
	var assigneeID string
	if opts.Assignee != "" {
		if opts.Assignee == "@me" {
			user, err := jira.GetMyself(ctx)
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}
			assigneeID = user.AccountID
		} else {
			// Search for user
			users, err := jira.SearchUsers(ctx, opts.Assignee)
			if err != nil {
				return fmt.Errorf("failed to search for user: %w", err)
			}
			if len(users) == 0 {
				return fmt.Errorf("user not found: %s", opts.Assignee)
			}
			assigneeID = users[0].AccountID
		}
	}

	// Auto-discover subtask type if --parent is provided but --type is not
	issueTypeName := opts.IssueType
	if opts.Parent != "" && opts.IssueType == "" {
		subtaskType, err := jira.GetSubtaskType(ctx, opts.Project)
		if err != nil {
			return fmt.Errorf("failed to discover subtask type: %w", err)
		}
		if subtaskType == nil {
			return fmt.Errorf("no subtask type found for project %s\n\nUse 'atl issue types --project %s' to list available types", opts.Project, opts.Project)
		}
		issueTypeName = subtaskType.Name
	}

	req := &api.CreateIssueRequest{
		Fields: api.CreateIssueFields{
			Project:   &api.ProjectID{Key: opts.Project},
			Summary:   opts.Summary,
			IssueType: &api.IssueTypeID{Name: issueTypeName},
			Labels:    opts.Labels,
		},
	}

	if opts.Description != "" {
		req.Fields.Description = api.TextToADF(opts.Description)
	}

	if assigneeID != "" {
		req.Fields.Assignee = &api.AccountID{AccountID: assigneeID}
	}

	if opts.Priority != "" {
		req.Fields.Priority = &api.PriorityID{Name: opts.Priority}
	}

	if opts.Parent != "" {
		req.Fields.Parent = &api.ParentID{Key: opts.Parent}
	}

	// Parse custom fields from file first (if provided)
	if opts.FieldFile != "" {
		data, err := os.ReadFile(opts.FieldFile)
		if err != nil {
			return fmt.Errorf("failed to read field file: %w", err)
		}

		var fileFields map[string]interface{}
		if err := json.Unmarshal(data, &fileFields); err != nil {
			return fmt.Errorf("failed to parse field file as JSON: %w", err)
		}

		req.Fields.CustomFields = make(map[string]interface{})
		for key, value := range fileFields {
			// Resolve field name to ID if needed
			if !strings.HasPrefix(key, "customfield_") && !isSystemField(key) {
				resolvedField, err := jira.GetFieldByName(ctx, key)
				if err != nil {
					return fmt.Errorf("failed to look up field '%s': %w", key, err)
				}
				if resolvedField == nil {
					return fmt.Errorf("field not found: %s\n\nUse 'atl issue fields --search \"%s\"' to find available fields", key, key)
				}
				key = resolvedField.ID
			}
			req.Fields.CustomFields[key] = value
		}
	}

	// Parse custom fields from command line (override file values)
	if len(opts.CustomFields) > 0 {
		if req.Fields.CustomFields == nil {
			req.Fields.CustomFields = make(map[string]interface{})
		}
		for _, field := range opts.CustomFields {
			key, fieldValue, err := ParseCustomField(ctx, jira, field)
			if err != nil {
				return err
			}
			req.Fields.CustomFields[key] = fieldValue
		}
	}

	result, err := jira.CreateIssue(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	createOutput := &CreateOutput{
		Key:     result.Key,
		ID:      result.ID,
		Summary: opts.Summary,
		Type:    opts.IssueType,
		Project: opts.Project,
		URL:     fmt.Sprintf("https://%s/browse/%s", client.Hostname(), result.Key),
	}

	if opts.Web {
		auth.OpenBrowser(createOutput.URL)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, createOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Created issue: %s\n", createOutput.Key)
	fmt.Fprintf(opts.IO.Out, "Summary: %s\n", createOutput.Summary)
	fmt.Fprintf(opts.IO.Out, "Type: %s\n", createOutput.Type)
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", createOutput.URL)

	return nil
}
