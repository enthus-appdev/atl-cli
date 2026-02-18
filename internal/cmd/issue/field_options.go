package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// FieldOptionsOptions holds the options for the field-options command.
type FieldOptionsOptions struct {
	IO        *iostreams.IOStreams
	Project   string
	IssueType string
	Field     string
	JSON      bool
}

// NewCmdFieldOptions creates the field-options command.
func NewCmdFieldOptions(ios *iostreams.IOStreams) *cobra.Command {
	opts := &FieldOptionsOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "field-options",
		Short: "Show allowed values for issue fields",
		Long:  `Display field metadata and allowed values for a project and issue type. Useful for discovering valid values for select, radio, and other constrained fields.`,
		Example: `  # Show all fields with allowed values for bugs
  atl issue field-options --project NX --type Bug

  # Show options for a specific field
  atl issue field-options --project NX --type Bug --field "Fehlverhalten"

  # Output as JSON
  atl issue field-options --project NX --type Bug --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Project == "" {
				return fmt.Errorf("--project flag is required\n\nUse 'atl issue types --project PROJ' to list available projects")
			}
			if opts.IssueType == "" {
				return fmt.Errorf("--type flag is required\n\nUse 'atl issue types --project %s' to list available issue types", opts.Project)
			}
			return runFieldOptions(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "Project key (required)")
	cmd.Flags().StringVarP(&opts.IssueType, "type", "t", "", "Issue type name (required)")
	cmd.Flags().StringVarP(&opts.Field, "field", "f", "", "Filter to a specific field by name")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// FieldOptionOutput represents a field with its allowed values.
type FieldOptionOutput struct {
	FieldID       string   `json:"field_id"`
	Name          string   `json:"name"`
	Required      bool     `json:"required"`
	Type          string   `json:"type,omitempty"`
	CustomType    string   `json:"custom_type,omitempty"`
	AllowedValues []string `json:"allowed_values"`
}

func runFieldOptions(opts *FieldOptionsOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Resolve issue type name to ID
	issueTypes, err := jira.GetProjectIssueTypes(ctx, opts.Project)
	if err != nil {
		return fmt.Errorf("failed to get issue types: %w", err)
	}

	var issueTypeID string
	typeLower := strings.ToLower(opts.IssueType)
	for _, it := range issueTypes {
		if strings.ToLower(it.Name) == typeLower {
			issueTypeID = it.ID
			break
		}
	}
	if issueTypeID == "" {
		var available []string
		for _, it := range issueTypes {
			available = append(available, it.Name)
		}
		return fmt.Errorf("issue type %q not found in project %s\n\nAvailable types: %s", opts.IssueType, opts.Project, strings.Join(available, ", "))
	}

	// Get field metadata
	fieldMetas, err := jira.GetFieldOptions(ctx, opts.Project, issueTypeID)
	if err != nil {
		return fmt.Errorf("failed to get field options: %w", err)
	}

	// Filter and format
	var results []*FieldOptionOutput
	fieldLower := strings.ToLower(opts.Field)

	for _, fm := range fieldMetas {
		// Skip fields without allowed values (unless filtering by name)
		if len(fm.AllowedValues) == 0 && opts.Field == "" {
			continue
		}

		// Filter by field name if specified
		if opts.Field != "" && !strings.Contains(strings.ToLower(fm.Name), fieldLower) {
			continue
		}

		result := &FieldOptionOutput{
			FieldID:  fm.FieldID,
			Name:     fm.Name,
			Required: fm.Required,
		}

		if fm.Schema != nil {
			result.Type = fm.Schema.Type
			result.CustomType = fm.Schema.Custom
		}

		// Extract allowed values
		for _, rawVal := range fm.AllowedValues {
			val := extractAllowedValue(rawVal)
			if val != "" {
				result.AllowedValues = append(result.AllowedValues, val)
			}
		}

		results = append(results, result)
	}

	// Sort by name
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	if opts.JSON {
		return output.JSON(opts.IO.Out, results)
	}

	if len(results) == 0 {
		if opts.Field != "" {
			fmt.Fprintf(opts.IO.Out, "No fields matching %q found with allowed values\n", opts.Field)
		} else {
			fmt.Fprintf(opts.IO.Out, "No fields with allowed values found for %s %s\n", opts.Project, opts.IssueType)
		}
		return nil
	}

	for i, r := range results {
		if i > 0 {
			fmt.Fprintln(opts.IO.Out)
		}
		required := ""
		if r.Required {
			required = " (required)"
		}
		fmt.Fprintf(opts.IO.Out, "%s [%s]%s\n", r.Name, r.FieldID, required)
		if r.CustomType != "" {
			fmt.Fprintf(opts.IO.Out, "  Type: %s\n", r.CustomType)
		} else if r.Type != "" {
			fmt.Fprintf(opts.IO.Out, "  Type: %s\n", r.Type)
		}
		if len(r.AllowedValues) > 0 {
			fmt.Fprintf(opts.IO.Out, "  Values: %s\n", strings.Join(r.AllowedValues, ", "))
		}
	}

	return nil
}

// extractAllowedValue extracts a display value from a raw allowed value JSON.
func extractAllowedValue(raw json.RawMessage) string {
	// Try {value: "..."} pattern (select/radio)
	var selectVal struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &selectVal); err == nil && selectVal.Value != "" {
		return selectVal.Value
	}

	// Try {name: "..."} pattern (priority, status, etc.)
	var nameVal struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &nameVal); err == nil && nameVal.Name != "" {
		return nameVal.Name
	}

	// Try plain string
	var strVal string
	if err := json.Unmarshal(raw, &strVal); err == nil {
		return strVal
	}

	return ""
}
