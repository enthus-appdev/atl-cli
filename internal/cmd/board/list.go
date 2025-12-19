package board

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// ListOptions holds the options for the list command.
type ListOptions struct {
	IO      *iostreams.IOStreams
	Project string
	JSON    bool
}

// NewCmdList creates the list command.
func NewCmdList(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Jira boards",
		Long:  `List all Jira boards, optionally filtered by project.`,
		Example: `  # List all boards
  atl board list

  # List boards for a specific project
  atl board list --project PROJ

  # Output as JSON
  atl board list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "Filter boards by project key")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// BoardOutput represents a board in output.
type BoardOutput struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	ProjectKey string `json:"project_key,omitempty"`
}

// BoardListOutput represents the list output.
type BoardListOutput struct {
	Boards []*BoardOutput `json:"boards"`
	Total  int            `json:"total"`
}

func runList(opts *ListOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	boards, err := jira.GetBoards(ctx, opts.Project)
	if err != nil {
		return fmt.Errorf("failed to get boards: %w", err)
	}

	listOutput := &BoardListOutput{
		Boards: make([]*BoardOutput, 0, len(boards)),
		Total:  len(boards),
	}

	for _, b := range boards {
		board := &BoardOutput{
			ID:   b.ID,
			Name: b.Name,
			Type: b.Type,
		}
		if b.Location != nil {
			board.ProjectKey = b.Location.ProjectKey
		}
		listOutput.Boards = append(listOutput.Boards, board)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, listOutput)
	}

	if len(listOutput.Boards) == 0 {
		fmt.Fprintln(opts.IO.Out, "No boards found")
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Boards (%d):\n\n", listOutput.Total)

	headers := []string{"ID", "NAME", "TYPE", "PROJECT"}
	rows := make([][]string, 0, len(listOutput.Boards))

	for _, b := range listOutput.Boards {
		rows = append(rows, []string{
			fmt.Sprintf("%d", b.ID),
			b.Name,
			b.Type,
			b.ProjectKey,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)

	return nil
}
