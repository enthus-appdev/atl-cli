package worklog

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// EditOptions holds the options for the edit command.
type EditOptions struct {
	IO          *iostreams.IOStreams
	WorklogID   string
	Time        string
	Description string
	Date        string
}

// NewCmdEdit creates the edit command.
func NewCmdEdit(ios *iostreams.IOStreams) *cobra.Command {
	opts := &EditOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "edit <worklog-id>",
		Short: "Edit a worklog",
		Long:  `Edit an existing Tempo worklog entry.`,
		Example: `  # Update time
  atl worklog edit 12345 --time 3h

  # Update description
  atl worklog edit 12345 --description "Updated work description"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.WorklogID = args[0]
			return runEdit(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Time, "time", "t", "", "New time spent")
	cmd.Flags().StringVar(&opts.Description, "description", "", "New description")
	cmd.Flags().StringVarP(&opts.Date, "date", "d", "", "New date")

	return cmd
}

func runEdit(opts *EditOptions) error {
	// TODO: Implement worklog edit

	fmt.Fprintf(opts.IO.Out, "Editing worklog: %s\n", opts.WorklogID)
	fmt.Fprintln(opts.IO.Out, "Not yet implemented. Please run 'atl auth login' first.")

	return nil
}
