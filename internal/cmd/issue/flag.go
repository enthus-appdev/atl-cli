package issue

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// FlagOptions holds the options for the flag command.
type FlagOptions struct {
	IO       *iostreams.IOStreams
	IssueKey string
	Unflag   bool
	Status   bool
	JSON     bool
}

// NewCmdFlag creates the flag command.
func NewCmdFlag(ios *iostreams.IOStreams) *cobra.Command {
	opts := &FlagOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "flag <issue-key>",
		Short: "Flag or unflag a Jira issue",
		Long: `Flag or unflag a Jira issue.

Flagged issues are marked as having an impediment and are highlighted
in sprint boards and backlogs. Use flags to indicate blocked work.`,
		Example: `  # Flag an issue
  atl issue flag PROJ-123

  # Unflag an issue
  atl issue flag PROJ-123 --unflag

  # Check flag status
  atl issue flag PROJ-123 --status

  # Output as JSON
  atl issue flag PROJ-123 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]
			return runFlag(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Unflag, "unflag", "u", false, "Remove the flag from the issue")
	cmd.Flags().BoolVarP(&opts.Status, "status", "s", false, "Check if the issue is flagged (don't change)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// FlagOutput represents the output of the flag command.
type FlagOutput struct {
	IssueKey string `json:"issue_key"`
	Flagged  bool   `json:"flagged"`
	Action   string `json:"action"`
}

func runFlag(opts *FlagOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Check status only
	if opts.Status {
		flagged, err := jira.IsIssueFlagged(ctx, opts.IssueKey)
		if err != nil {
			return fmt.Errorf("failed to check flag status: %w", err)
		}

		flagOutput := &FlagOutput{
			IssueKey: opts.IssueKey,
			Flagged:  flagged,
			Action:   "status",
		}

		if opts.JSON {
			return output.JSON(opts.IO.Out, flagOutput)
		}

		if flagged {
			fmt.Fprintf(opts.IO.Out, "%s is flagged\n", opts.IssueKey)
		} else {
			fmt.Fprintf(opts.IO.Out, "%s is not flagged\n", opts.IssueKey)
		}
		return nil
	}

	var flagOutput *FlagOutput

	if opts.Unflag {
		// Unflag the issue
		err = jira.UnflagIssue(ctx, opts.IssueKey)
		if err != nil {
			return fmt.Errorf("failed to unflag issue: %w", err)
		}

		flagOutput = &FlagOutput{
			IssueKey: opts.IssueKey,
			Flagged:  false,
			Action:   "unflagged",
		}

		if opts.JSON {
			return output.JSON(opts.IO.Out, flagOutput)
		}

		fmt.Fprintf(opts.IO.Out, "Removed flag from %s\n", opts.IssueKey)
	} else {
		// Flag the issue
		err = jira.FlagIssue(ctx, opts.IssueKey)
		if err != nil {
			return fmt.Errorf("failed to flag issue: %w", err)
		}

		flagOutput = &FlagOutput{
			IssueKey: opts.IssueKey,
			Flagged:  true,
			Action:   "flagged",
		}

		if opts.JSON {
			return output.JSON(opts.IO.Out, flagOutput)
		}

		fmt.Fprintf(opts.IO.Out, "Flagged %s\n", opts.IssueKey)
	}

	return nil
}
