package issue

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// TypesOptions holds the options for the types command.
type TypesOptions struct {
	IO      *iostreams.IOStreams
	Project string
	JSON    bool
}

// NewCmdTypes creates the types command.
func NewCmdTypes(ios *iostreams.IOStreams) *cobra.Command {
	opts := &TypesOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "types",
		Short: "List available issue types for a project",
		Long: `List all available issue types for a Jira project.

Shows which types are regular issues vs subtasks. Use this to find
the correct issue type name when creating subtasks.`,
		Example: `  # List issue types for a project
  atl issue types --project PROJ

  # Output as JSON
  atl issue types --project PROJ --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Project == "" {
				return fmt.Errorf("--project is required\n\nExample: atl issue types --project PROJ")
			}
			return runTypes(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "Project key (required)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// TypeOutput represents an issue type in output.
type TypeOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Subtask     bool   `json:"subtask"`
}

// TypesOutput represents the list output.
type TypesOutput struct {
	Project string        `json:"project"`
	Types   []*TypeOutput `json:"types"`
	Total   int           `json:"total"`
}

func runTypes(opts *TypesOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	types, err := jira.GetProjectIssueTypes(ctx, opts.Project)
	if err != nil {
		return fmt.Errorf("failed to get issue types: %w", err)
	}

	typesOutput := &TypesOutput{
		Project: opts.Project,
		Types:   make([]*TypeOutput, 0, len(types)),
		Total:   len(types),
	}

	for _, t := range types {
		typesOutput.Types = append(typesOutput.Types, &TypeOutput{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Subtask:     t.Subtask,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, typesOutput)
	}

	if len(typesOutput.Types) == 0 {
		fmt.Fprintf(opts.IO.Out, "No issue types found for project %s\n", opts.Project)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Issue types for %s:\n\n", opts.Project)

	headers := []string{"ID", "NAME", "SUBTASK", "DESCRIPTION"}
	rows := make([][]string, 0, len(typesOutput.Types))

	for _, t := range typesOutput.Types {
		subtask := ""
		if t.Subtask {
			subtask = "Yes"
		}
		desc := t.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		rows = append(rows, []string{
			t.ID,
			t.Name,
			subtask,
			desc,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)

	// Show hint about subtasks
	for _, t := range typesOutput.Types {
		if t.Subtask {
			fmt.Fprintf(opts.IO.Out, "\nTo create a subtask:\n")
			fmt.Fprintf(opts.IO.Out, "  atl issue create --project %s --type \"%s\" --parent PROJ-123 --summary \"Subtask\"\n", opts.Project, t.Name)
			break
		}
	}

	return nil
}
