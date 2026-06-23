package sprint

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

type backlogOptions struct {
	IO        *iostreams.IOStreams
	IssueKeys []string
	JSON      bool
}

func newCmdBacklog(ios *iostreams.IOStreams) *cobra.Command {
	opts := &backlogOptions{IO: ios}
	cmd := &cobra.Command{
		Use:     "backlog <issue-keys...>",
		Short:   "Move issues to the backlog (remove from sprint)",
		Args:    cobra.MinimumNArgs(1),
		Example: `  atl jira sprint backlog NX-1 NX-2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKeys = args
			return runBacklog(opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	return cmd
}

func runBacklog(opts *backlogOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	jira := api.NewJiraService(client)

	if err := jira.RemoveIssuesFromSprint(ctx, opts.IssueKeys); err != nil {
		return fmt.Errorf("failed to move issues to backlog: %w", err)
	}
	if opts.JSON {
		return output.JSON(opts.IO.Out, map[string]any{"issues": opts.IssueKeys, "action": "moved_to_backlog"})
	}
	fmt.Fprintf(opts.IO.Out, "Moved %d issue(s) to backlog\n", len(opts.IssueKeys))
	return nil
}
