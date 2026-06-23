package sprint

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

type closeOptions struct {
	IO       *iostreams.IOStreams
	SprintID int
	Force    bool
	JSON     bool
}

func newCmdClose(ios *iostreams.IOStreams) *cobra.Command {
	opts := &closeOptions{IO: ios}
	cmd := &cobra.Command{
		Use:   "close <sprint-id>",
		Short: "Close (complete) a sprint",
		Args:  cobra.ExactArgs(1),
		Example: `  atl jira sprint close 123
  atl jira sprint close 123 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid sprint ID %q", args[0])
			}
			opts.SprintID = id
			return runClose(opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	return cmd
}

func runClose(opts *closeOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Confirm unless forced, JSON, or non-interactive.
	if !opts.Force && !opts.JSON && opts.IO.IsStdinTTY {
		name := strconv.Itoa(opts.SprintID)
		if s, err := jira.GetSprint(ctx, opts.SprintID); err == nil {
			name = fmt.Sprintf("%s (%d)", s.Name, s.ID)
		}
		fmt.Fprintf(opts.IO.Out, "Close sprint %s? Incomplete issues move out. [y/N]: ", name)
		var confirm string
		fmt.Fscanln(opts.IO.In, &confirm)
		if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
			fmt.Fprintln(opts.IO.Out, "Aborted")
			return nil
		}
	}

	sprint, err := jira.UpdateSprint(ctx, opts.SprintID, map[string]any{"state": "closed"})
	if err != nil {
		return fmt.Errorf("failed to close sprint: %w", err)
	}
	if !opts.JSON {
		fmt.Fprintf(opts.IO.Out, "Closed sprint %d\n", sprint.ID)
	}
	return printSprint(opts.IO, sprint, opts.JSON)
}
