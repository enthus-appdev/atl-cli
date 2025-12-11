package worklog

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// DeleteOptions holds the options for the delete command.
type DeleteOptions struct {
	IO        *iostreams.IOStreams
	WorklogID string
	Confirm   bool
}

// NewCmdDelete creates the delete command.
func NewCmdDelete(ios *iostreams.IOStreams) *cobra.Command {
	opts := &DeleteOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "delete <worklog-id>",
		Short: "Delete a worklog",
		Long:  `Delete a Tempo worklog entry.`,
		Example: `  # Delete a worklog (will prompt for confirmation)
  atl worklog delete 12345

  # Delete without confirmation
  atl worklog delete 12345 --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.WorklogID = args[0]
			return runDelete(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(opts *DeleteOptions) error {
	// TODO: Implement worklog delete

	fmt.Fprintf(opts.IO.Out, "Deleting worklog: %s\n", opts.WorklogID)
	fmt.Fprintln(opts.IO.Out, "Not yet implemented. Please run 'atl auth login' first.")

	return nil
}
