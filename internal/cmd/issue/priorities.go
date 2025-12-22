package issue

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// PrioritiesOptions holds the options for the priorities command.
type PrioritiesOptions struct {
	IO   *iostreams.IOStreams
	JSON bool
}

// NewCmdPriorities creates the priorities command.
func NewCmdPriorities(ios *iostreams.IOStreams) *cobra.Command {
	opts := &PrioritiesOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "priorities",
		Short: "List available priorities",
		Long: `List all available priorities in the Jira instance.

Use this to find the correct priority name when creating or editing issues.`,
		Example: `  # List all priorities
  atl issue priorities

  # Output as JSON
  atl issue priorities --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPriorities(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// PriorityOutput represents a priority in output.
type PriorityOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PrioritiesOutput represents the list output.
type PrioritiesOutput struct {
	Priorities []*PriorityOutput `json:"priorities"`
	Total      int               `json:"total"`
}

func runPriorities(opts *PrioritiesOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	priorities, err := jira.GetPriorities(ctx)
	if err != nil {
		return fmt.Errorf("failed to get priorities: %w", err)
	}

	prioritiesOutput := &PrioritiesOutput{
		Priorities: make([]*PriorityOutput, 0, len(priorities)),
		Total:      len(priorities),
	}

	for _, p := range priorities {
		prioritiesOutput.Priorities = append(prioritiesOutput.Priorities, &PriorityOutput{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, prioritiesOutput)
	}

	if len(prioritiesOutput.Priorities) == 0 {
		fmt.Fprintf(opts.IO.Out, "No priorities found\n")
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Available priorities:\n\n")

	headers := []string{"ID", "NAME", "DESCRIPTION"}
	rows := make([][]string, 0, len(prioritiesOutput.Priorities))

	for _, p := range prioritiesOutput.Priorities {
		desc := p.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		rows = append(rows, []string{
			p.ID,
			p.Name,
			desc,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)

	// Show usage hint
	fmt.Fprintf(opts.IO.Out, "\nUsage:\n")
	fmt.Fprintf(opts.IO.Out, "  atl issue create --project PROJ --type Bug --summary \"Title\" --priority \"%s\"\n", prioritiesOutput.Priorities[0].Name)
	fmt.Fprintf(opts.IO.Out, "  atl issue edit PROJ-1234 --priority \"%s\"\n", prioritiesOutput.Priorities[0].Name)

	return nil
}
