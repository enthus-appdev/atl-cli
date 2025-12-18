package issue

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// FieldsOptions holds the options for the fields command.
type FieldsOptions struct {
	IO         *iostreams.IOStreams
	CustomOnly bool
	Search     string
	JSON       bool
}

// NewCmdFields creates the fields command.
func NewCmdFields(ios *iostreams.IOStreams) *cobra.Command {
	opts := &FieldsOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "fields",
		Short: "List available Jira fields",
		Long: `List all available fields in Jira, including custom fields.

Use this command to discover field IDs for custom fields like "Story Points"
which are needed when using the --field flag with create or edit commands.`,
		Example: `  # List all fields
  atl issue fields

  # List only custom fields
  atl issue fields --custom

  # Search for a specific field
  atl issue fields --search "story points"

  # Output as JSON
  atl issue fields --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFields(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.CustomOnly, "custom", "c", false, "Show only custom fields")
	cmd.Flags().StringVarP(&opts.Search, "search", "s", "", "Search for fields by name")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// FieldOutput represents a field in the output.
type FieldOutput struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Custom bool   `json:"custom"`
}

// FieldsOutput represents the output for fields list.
type FieldsOutput struct {
	Fields []*FieldOutput `json:"fields"`
	Total  int            `json:"total"`
}

func runFields(opts *FieldsOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	fields, err := jira.GetFields(ctx)
	if err != nil {
		return fmt.Errorf("failed to get fields: %w", err)
	}

	fieldsOutput := &FieldsOutput{
		Fields: make([]*FieldOutput, 0),
	}

	searchLower := strings.ToLower(opts.Search)

	for _, f := range fields {
		// Filter by custom only
		if opts.CustomOnly && !f.Custom {
			continue
		}

		// Filter by search term
		if opts.Search != "" && !strings.Contains(strings.ToLower(f.Name), searchLower) {
			continue
		}

		fieldType := ""
		if f.Schema != nil {
			fieldType = f.Schema.Type
			if f.Schema.Custom != "" {
				// Extract the custom field type from the full schema
				parts := strings.Split(f.Schema.Custom, ":")
				if len(parts) > 1 {
					fieldType = parts[len(parts)-1]
				}
			}
		}

		fieldsOutput.Fields = append(fieldsOutput.Fields, &FieldOutput{
			ID:     f.ID,
			Name:   f.Name,
			Type:   fieldType,
			Custom: f.Custom,
		})
	}

	fieldsOutput.Total = len(fieldsOutput.Fields)

	if opts.JSON {
		return output.JSON(opts.IO.Out, fieldsOutput)
	}

	if fieldsOutput.Total == 0 {
		fmt.Fprintln(opts.IO.Out, "No fields found")
		return nil
	}

	what := "fields"
	if opts.CustomOnly {
		what = "custom fields"
	}
	fmt.Fprintf(opts.IO.Out, "Found %d %s:\n\n", fieldsOutput.Total, what)

	headers := []string{"ID", "NAME", "TYPE", "CUSTOM"}
	rows := make([][]string, 0, len(fieldsOutput.Fields))

	for _, f := range fieldsOutput.Fields {
		name := f.Name
		if len(name) > 40 {
			name = name[:37] + "..."
		}
		custom := ""
		if f.Custom {
			custom = "✓"
		}
		rows = append(rows, []string{
			f.ID,
			name,
			f.Type,
			custom,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)

	if opts.CustomOnly || opts.Search != "" {
		fmt.Fprintf(opts.IO.Out, "\nUse field ID with: atl issue edit ISSUE-123 --field %s=VALUE\n", fieldsOutput.Fields[0].ID)
	}

	return nil
}
