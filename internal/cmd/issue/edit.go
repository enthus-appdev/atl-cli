package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// EditOptions holds the options for the edit command.
type EditOptions struct {
	IO           *iostreams.IOStreams
	IssueKey     string
	Summary      string
	Description  string
	Append       bool
	Assignee     string
	AddLabels    []string
	RemoveLabels []string
	Priority     string
	CustomFields []string
	FieldFile    string
	JSON         bool
}

// NewCmdEdit creates the edit command.
func NewCmdEdit(ios *iostreams.IOStreams) *cobra.Command {
	opts := &EditOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "edit <issue-key>",
		Short: "Edit a Jira issue",
		Long:  `Edit fields of an existing Jira issue.`,
		Example: `  # Edit issue summary
  atl issue edit PROJ-1234 --summary "Updated summary"

  # Set new description (replaces existing)
  atl issue edit PROJ-1234 --description "New description content"

  # Append to existing description (preserves embedded media)
  atl issue edit PROJ-1234 --description "Additional notes" --append

  # Add labels
  atl issue edit PROJ-1234 --add-label bug --add-label urgent

  # Remove labels
  atl issue edit PROJ-1234 --remove-label wontfix

  # Change assignee
  atl issue edit PROJ-1234 --assignee john.doe

  # Change priority
  atl issue edit PROJ-1234 --priority High

  # Set custom fields by name (Story Points, etc.)
  atl issue edit PROJ-1234 --field "Story Points=8"

  # Or use field ID directly
  atl issue edit PROJ-1234 --field customfield_10016=8

  # Use a JSON file for complex field values (like ADF rich text)
  atl issue edit PROJ-1234 --field-file fields.json

  # Output result as JSON
  atl issue edit PROJ-1234 --summary "New summary" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]
			return runEdit(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Summary, "summary", "s", "", "New summary")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "New description")
	cmd.Flags().BoolVar(&opts.Append, "append", false, "Append to existing description instead of replacing")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "New assignee (use @me for yourself, empty to unassign)")
	cmd.Flags().StringSliceVar(&opts.AddLabels, "add-label", nil, "Labels to add")
	cmd.Flags().StringSliceVar(&opts.RemoveLabels, "remove-label", nil, "Labels to remove")
	cmd.Flags().StringVar(&opts.Priority, "priority", "", "New priority")
	cmd.Flags().StringSliceVarP(&opts.CustomFields, "field", "f", nil, "Custom field in key=value format (can be repeated)")
	cmd.Flags().StringVar(&opts.FieldFile, "field-file", "", "JSON file with field values (for complex types like ADF)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// EditOutput represents the output after editing an issue.
type EditOutput struct {
	Key           string   `json:"key"`
	FieldsUpdated []string `json:"fields_updated"`
	LabelsAdded   []string `json:"labels_added,omitempty"`
	LabelsRemoved []string `json:"labels_removed,omitempty"`
	URL           string   `json:"url"`
}

func runEdit(opts *EditOptions) error {
	// Check that at least one field is being edited
	if opts.Summary == "" && opts.Description == "" && opts.Assignee == "" &&
		len(opts.AddLabels) == 0 && len(opts.RemoveLabels) == 0 && opts.Priority == "" &&
		len(opts.CustomFields) == 0 && opts.FieldFile == "" {
		return fmt.Errorf("at least one field must be specified to edit")
	}

	// Validate --append requires --description
	if opts.Append && opts.Description == "" {
		return fmt.Errorf("--append requires --description flag")
	}

	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	editOutput := &EditOutput{
		Key:           opts.IssueKey,
		FieldsUpdated: []string{},
		URL:           fmt.Sprintf("https://%s/browse/%s", client.Hostname(), opts.IssueKey),
	}

	// Build update request
	req := &api.UpdateIssueRequest{
		Fields: make(map[string]interface{}),
		Update: make(map[string][]api.UpdateOp),
	}

	if opts.Summary != "" {
		req.Fields["summary"] = opts.Summary
		editOutput.FieldsUpdated = append(editOutput.FieldsUpdated, "summary")
	}

	if opts.Description != "" {
		newADF := api.TextToADF(opts.Description)

		if opts.Append {
			// Fetch existing issue to get current description
			issue, err := jira.GetIssue(ctx, opts.IssueKey)
			if err != nil {
				return fmt.Errorf("failed to fetch existing issue: %w", err)
			}

			// Merge existing and new description content
			if issue.Fields.Description != nil && len(issue.Fields.Description.Content) > 0 {
				// Append new content to existing content
				mergedContent := append(issue.Fields.Description.Content, newADF.Content...)
				newADF.Content = mergedContent
			}
		}

		req.Fields["description"] = newADF
		editOutput.FieldsUpdated = append(editOutput.FieldsUpdated, "description")
	}

	if opts.Priority != "" {
		req.Fields["priority"] = map[string]string{"name": opts.Priority}
		editOutput.FieldsUpdated = append(editOutput.FieldsUpdated, "priority")
	}

	// Handle labels
	if len(opts.AddLabels) > 0 {
		var ops []api.UpdateOp
		for _, label := range opts.AddLabels {
			ops = append(ops, api.UpdateOp{Add: label})
		}
		req.Update["labels"] = ops
		editOutput.LabelsAdded = opts.AddLabels
		editOutput.FieldsUpdated = append(editOutput.FieldsUpdated, "labels")
	}

	if len(opts.RemoveLabels) > 0 {
		ops := req.Update["labels"]
		for _, label := range opts.RemoveLabels {
			ops = append(ops, api.UpdateOp{Remove: label})
		}
		req.Update["labels"] = ops
		editOutput.LabelsRemoved = opts.RemoveLabels
		if len(opts.AddLabels) == 0 {
			editOutput.FieldsUpdated = append(editOutput.FieldsUpdated, "labels")
		}
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
			req.Fields[key] = value
			editOutput.FieldsUpdated = append(editOutput.FieldsUpdated, key)
		}
	}

	// Parse and add custom fields from command line (override file values)
	for _, field := range opts.CustomFields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid field format: %s (expected key=value)", field)
		}
		key, value := parts[0], parts[1]

		var resolvedField *api.Field
		var err error

		if strings.HasPrefix(key, "customfield_") {
			// Look up field by ID to get type information
			resolvedField, err = jira.GetFieldByID(ctx, key)
			if err != nil {
				return fmt.Errorf("failed to look up field '%s': %w", key, err)
			}
			// Note: resolvedField may be nil if field doesn't exist, we'll still try to set it
		} else if !isSystemField(key) {
			// Resolve field by name
			resolvedField, err = jira.GetFieldByName(ctx, key)
			if err != nil {
				return fmt.Errorf("failed to look up field '%s': %w", key, err)
			}
			if resolvedField == nil {
				return fmt.Errorf("field not found: %s\n\nUse 'atl issue fields --search \"%s\"' to find available fields", key, key)
			}
			key = resolvedField.ID
		}

		// Determine field value based on field type
		var fieldValue interface{}

		// Check if this is a textarea field that requires ADF format
		if resolvedField != nil && resolvedField.Schema != nil &&
			strings.Contains(resolvedField.Schema.Custom, "textarea") {
			// Convert Markdown to ADF for textarea fields
			fieldValue = api.TextToADF(value)
		} else if numVal, err := strconv.ParseFloat(value, 64); err == nil {
			// Try to parse value as number
			fieldValue = numVal
		} else {
			fieldValue = value
		}
		req.Fields[key] = fieldValue
		editOutput.FieldsUpdated = append(editOutput.FieldsUpdated, key)
	}

	// Update the issue fields first
	if len(req.Fields) > 0 || len(req.Update) > 0 {
		if err := jira.UpdateIssue(ctx, opts.IssueKey, req); err != nil {
			return fmt.Errorf("failed to update issue: %w", err)
		}
	}

	// Handle assignee separately (uses different endpoint)
	if opts.Assignee != "" {
		var accountID string
		switch opts.Assignee {
		case "@me":
			user, err := jira.GetMyself(ctx)
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}
			accountID = user.AccountID
		case "-", "none":
			accountID = "" // Unassign
		default:
			users, err := jira.SearchUsers(ctx, opts.Assignee)
			if err != nil {
				return fmt.Errorf("failed to search for user: %w", err)
			}
			if len(users) == 0 {
				return fmt.Errorf("user not found: %s", opts.Assignee)
			}
			accountID = users[0].AccountID
		}

		if err := jira.AssignIssue(ctx, opts.IssueKey, accountID); err != nil {
			return fmt.Errorf("failed to assign issue: %w", err)
		}
		editOutput.FieldsUpdated = append(editOutput.FieldsUpdated, "assignee")
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, editOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Updated issue: %s\n", editOutput.Key)
	fmt.Fprintf(opts.IO.Out, "Fields updated: %v\n", editOutput.FieldsUpdated)
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", editOutput.URL)

	return nil
}
