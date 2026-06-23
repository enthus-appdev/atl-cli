package sprint

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

type startOptions struct {
	IO        *iostreams.IOStreams
	SprintID  int
	Duration  string
	StartDate string
	EndDate   string
	JSON      bool
}

func newCmdStart(ios *iostreams.IOStreams) *cobra.Command {
	opts := &startOptions{IO: ios}
	cmd := &cobra.Command{
		Use:   "start <sprint-id>",
		Short: "Start a sprint (set state to active)",
		Args:  cobra.ExactArgs(1),
		Example: `  atl jira sprint start 123 --duration 14d
  atl jira sprint start 123 --start-date 2026-06-23 --end-date 2026-07-07`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid sprint ID %q", args[0])
			}
			opts.SprintID = id
			return runStart(opts)
		},
	}
	cmd.Flags().StringVar(&opts.Duration, "duration", "14d", "Sprint length (e.g. 14d, 2w)")
	cmd.Flags().StringVar(&opts.StartDate, "start-date", "", "Start date YYYY-MM-DD (overrides now)")
	cmd.Flags().StringVar(&opts.EndDate, "end-date", "", "End date YYYY-MM-DD (overrides duration)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	return cmd
}

func runStart(opts *startOptions) error {
	dur, err := parseDuration(opts.Duration)
	if err != nil {
		return err
	}
	start, end, err := resolveActiveDates(opts.StartDate, opts.EndDate, dur, time.Now().UTC())
	if err != nil {
		return err
	}
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	jira := api.NewJiraService(client)

	sprint, err := jira.UpdateSprint(ctx, opts.SprintID, buildActivateBody(start, end))
	if err != nil {
		return fmt.Errorf("failed to start sprint: %w", err)
	}
	if !opts.JSON {
		fmt.Fprintf(opts.IO.Out, "Started sprint %d\n", sprint.ID)
	}
	return printSprint(opts.IO, sprint, opts.JSON)
}
