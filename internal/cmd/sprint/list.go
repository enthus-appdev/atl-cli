package sprint

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

type listOptions struct {
	IO      *iostreams.IOStreams
	BoardID int
	State   string
	JSON    bool
}

type sprintListOutput struct {
	BoardID int             `json:"board_id"`
	Sprints []*sprintDetail `json:"sprints"`
	Total   int             `json:"total"`
}

func newCmdList(ios *iostreams.IOStreams) *cobra.Command {
	opts := &listOptions{IO: ios}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sprints on a board",
		Example: `  atl jira sprint list --board 42
  atl jira sprint list --board 42 --state closed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.BoardID == 0 {
				return fmt.Errorf("--board is required\n\nUse 'atl jira board list --project PROJ' to find a board ID")
			}
			return runList(opts)
		},
	}
	cmd.Flags().IntVar(&opts.BoardID, "board", 0, "Board ID (required)")
	cmd.Flags().StringVar(&opts.State, "state", "active,future", "Comma-separated states: active, future, closed")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	return cmd
}

func runList(opts *listOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	jira := api.NewJiraService(client)

	sprints, err := jira.GetSprints(ctx, opts.BoardID, opts.State)
	if err != nil {
		return fmt.Errorf("failed to get sprints: %w", err)
	}

	out := &sprintListOutput{BoardID: opts.BoardID, Sprints: make([]*sprintDetail, 0, len(sprints)), Total: len(sprints)}
	for _, s := range sprints {
		out.Sprints = append(out.Sprints, newSprintDetail(s))
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, out)
	}
	if out.Total == 0 {
		fmt.Fprintf(opts.IO.Out, "No sprints found for board %d (state: %s)\n", opts.BoardID, opts.State)
		return nil
	}
	fmt.Fprintf(opts.IO.Out, "Sprints for board %d:\n\n", opts.BoardID)
	headers := []string{"ID", "NAME", "STATE", "START", "END", "GOAL"}
	rows := make([][]string, 0, len(out.Sprints))
	for _, s := range out.Sprints {
		rows = append(rows, []string{
			strconv.Itoa(s.ID), s.Name, s.State,
			dateOnly(s.StartDate), dateOnly(s.EndDate), s.Goal,
		})
	}
	output.SimpleTable(opts.IO.Out, headers, rows)
	return nil
}
