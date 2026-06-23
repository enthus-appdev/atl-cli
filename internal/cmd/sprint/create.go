package sprint

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

type createOptions struct {
	IO        *iostreams.IOStreams
	BoardID   int
	Name      string
	Goal      string
	Start     bool
	Duration  string
	StartDate string
	EndDate   string
	JSON      bool
}

func newCmdCreate(ios *iostreams.IOStreams) *cobra.Command {
	opts := &createOptions{IO: ios}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new sprint",
		Example: `  # Create a future sprint with a goal
  atl jira sprint create --board 42 --name "Sprint 30" --goal "Ship MI cutover"

  # Create and start it immediately for 14 days
  atl jira sprint create --board 42 --name "Sprint 30" --goal "..." --start --duration 14d`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.BoardID == 0 {
				return fmt.Errorf("--board is required\n\nUse 'atl jira board list --project PROJ' to find a board ID")
			}
			if opts.Name == "" {
				return fmt.Errorf("--name is required")
			}
			return runCreate(opts)
		},
	}
	cmd.Flags().IntVar(&opts.BoardID, "board", 0, "Board ID the sprint belongs to (required)")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Sprint name (required)")
	cmd.Flags().StringVar(&opts.Goal, "goal", "", "Sprint goal")
	cmd.Flags().BoolVar(&opts.Start, "start", false, "Start the sprint immediately (sets state active)")
	cmd.Flags().StringVar(&opts.Duration, "duration", "14d", "Sprint length when starting (e.g. 14d, 2w)")
	cmd.Flags().StringVar(&opts.StartDate, "start-date", "", "Start date YYYY-MM-DD (overrides now)")
	cmd.Flags().StringVar(&opts.EndDate, "end-date", "", "End date YYYY-MM-DD (overrides duration)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	return cmd
}

func runCreate(opts *createOptions) error {
	// Validate and resolve --start dates before any server-side mutation so that
	// a bad --duration or --end-date does not leave an orphaned unstarted sprint.
	var (
		activateStart time.Time
		activateEnd   time.Time
	)
	if opts.Start {
		dur, err := parseDuration(opts.Duration)
		if err != nil {
			return err
		}
		s, e, err := resolveActiveDates(opts.StartDate, opts.EndDate, dur, time.Now())
		if err != nil {
			return err
		}
		activateStart, activateEnd = s, e
	}

	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	jira := api.NewJiraService(client)

	body, err := buildCreateBody(opts.BoardID, opts.Name, opts.Goal, opts.StartDate, opts.EndDate)
	if err != nil {
		return err
	}
	sprint, err := jira.CreateSprint(ctx, body)
	if err != nil {
		return fmt.Errorf("failed to create sprint: %w", err)
	}

	if opts.Start {
		createdID := sprint.ID // UpdateSprint returns nil on error; keep the ID for the message
		started, err := jira.UpdateSprint(ctx, createdID, buildActivateBody(activateStart, activateEnd))
		if err != nil {
			return fmt.Errorf("sprint created (ID %d) but failed to start: %w", createdID, err)
		}
		sprint = started
	}

	if !opts.JSON {
		fmt.Fprintf(opts.IO.Out, "Created sprint %d\n", sprint.ID)
	}
	return printSprint(opts.IO, sprint, opts.JSON)
}
