package sprint

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

type editOptions struct {
	IO        *iostreams.IOStreams
	SprintID  int
	Name      string
	Goal      string
	StartDate string
	EndDate   string
	JSON      bool
}

func newCmdEdit(ios *iostreams.IOStreams) *cobra.Command {
	opts := &editOptions{IO: ios}
	cmd := &cobra.Command{
		Use:   "edit <sprint-id>",
		Short: "Edit a sprint's name, goal, or dates",
		Args:  cobra.ExactArgs(1),
		Example: `  atl jira sprint edit 123 --goal "Updated goal"
  atl jira sprint edit 123 --name "Sprint 30 (extended)" --end-date 2026-07-14`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid sprint ID %q", args[0])
			}
			opts.SprintID = id
			return runEdit(opts)
		},
	}
	cmd.Flags().StringVar(&opts.Name, "name", "", "New sprint name")
	cmd.Flags().StringVar(&opts.Goal, "goal", "", "New sprint goal")
	cmd.Flags().StringVar(&opts.StartDate, "start-date", "", "New start date YYYY-MM-DD")
	cmd.Flags().StringVar(&opts.EndDate, "end-date", "", "New end date YYYY-MM-DD")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	return cmd
}

func runEdit(opts *editOptions) error {
	body, err := buildEditBody(opts.Name, opts.Goal, opts.StartDate, opts.EndDate)
	if err != nil {
		return err
	}
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	jira := api.NewJiraService(client)

	sprint, err := jira.UpdateSprint(ctx, opts.SprintID, body)
	if err != nil {
		return fmt.Errorf("failed to update sprint: %w", err)
	}
	if !opts.JSON {
		fmt.Fprintf(opts.IO.Out, "Updated sprint %d\n", sprint.ID)
	}
	return printSprint(opts.IO, sprint, opts.JSON)
}
