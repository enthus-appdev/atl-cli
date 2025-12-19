package worklog

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// ListOptions holds the options for the list command.
type ListOptions struct {
	IO       *iostreams.IOStreams
	From     string
	To       string
	IssueKey string
	Limit    int
	JSON     bool
}

// NewCmdList creates the list command.
func NewCmdList(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		IO:    ios,
		Limit: 50,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List worklogs",
		Long:    `List Tempo worklogs for a date range or issue.`,
		Example: `  # List worklogs for today
  atl worklog list

  # List worklogs for a date range
  atl worklog list --from 2024-01-01 --to 2024-01-31

  # List worklogs for a specific issue
  atl worklog list --issue PROJ-1234

  # List this week's worklogs
  atl worklog list --from monday`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts)
		},
	}

	cmd.Flags().StringVar(&opts.From, "from", "", "Start date (YYYY-MM-DD or 'today', 'monday', etc.)")
	cmd.Flags().StringVar(&opts.To, "to", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringVarP(&opts.IssueKey, "issue", "i", "", "Filter by issue key")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 50, "Maximum number of worklogs to return")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

func runList(opts *ListOptions) error {
	// TODO: Implement worklog list

	fmt.Fprintln(opts.IO.Out, "Listing worklogs...")
	fmt.Fprintln(opts.IO.Out, "Not yet implemented. Please run 'atl auth login' first.")

	return nil
}
