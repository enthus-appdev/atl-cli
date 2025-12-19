package worklog

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// AddOptions holds the options for the add command.
type AddOptions struct {
	IO          *iostreams.IOStreams
	IssueKey    string
	Time        string
	Date        string
	Description string
	StartTime   string
}

// NewCmdAdd creates the add command.
func NewCmdAdd(ios *iostreams.IOStreams) *cobra.Command {
	opts := &AddOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "add <issue-key>",
		Short: "Log time to an issue",
		Long:  `Add a worklog entry to a Jira issue via Tempo.`,
		Example: `  # Log 2 hours to an issue
  atl worklog add PROJ-1234 --time 2h

  # Log time for a specific date
  atl worklog add PROJ-1234 --time 1h30m --date 2024-01-15

  # Log time with description
  atl worklog add PROJ-1234 --time 2h --description "Implemented feature X"

  # Log time with start time
  atl worklog add PROJ-1234 --time 2h --start 09:00`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]
			if opts.Time == "" {
				return fmt.Errorf("--time flag is required\n\nExample: atl worklog add PROJ-123 --time 2h")
			}
			return runAdd(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Time, "time", "t", "", "Time spent (e.g., 2h, 1h30m, 90m) (required)")
	cmd.Flags().StringVarP(&opts.Date, "date", "d", "", "Date of work (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Work description")
	cmd.Flags().StringVar(&opts.StartTime, "start", "", "Start time (HH:MM)")

	return cmd
}

func runAdd(opts *AddOptions) error {
	// TODO: Implement worklog add

	fmt.Fprintf(opts.IO.Out, "Logging %s to %s\n", opts.Time, opts.IssueKey)
	fmt.Fprintln(opts.IO.Out, "Not yet implemented. Please run 'atl auth login' first.")

	return nil
}
