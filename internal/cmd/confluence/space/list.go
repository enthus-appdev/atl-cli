package space

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
	IO     *iostreams.IOStreams
	Limit  int
	Cursor string
	All    bool
	JSON   bool
}

// NewCmdList creates the list command.
func NewCmdList(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		IO:    ios,
		Limit: 25,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Confluence spaces",
		Long:    `List all Confluence spaces you have access to.`,
		Example: `  # List spaces
  atl confluence space list

  # List more spaces
  atl confluence space list --limit 100

  # Fetch all spaces
  atl confluence space list --all

  # Get next page using cursor
  atl confluence space list --cursor <cursor>

  # Output as JSON
  atl confluence space list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts)
		},
	}

	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 25, "Maximum number of spaces per page")
	cmd.Flags().StringVar(&opts.Cursor, "cursor", "", "Pagination cursor for next page")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Fetch all spaces (ignores --limit and --cursor)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// SpaceListOutput represents the output for space list.
type SpaceListOutput struct {
	Spaces     []*SpaceOutput `json:"spaces"`
	Total      int            `json:"total"`
	HasMore    bool           `json:"has_more"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// SpaceOutput represents a single space in the list.
type SpaceOutput struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

func runList(opts *ListOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	var spaces []*api.Space
	var nextCursor string
	var hasMore bool

	if opts.All {
		// Fetch all spaces
		if !opts.JSON {
			fmt.Fprint(opts.IO.Out, "Fetching all spaces...")
		}
		spaces, err = confluence.GetSpacesAll(ctx)
		if err != nil {
			return fmt.Errorf("failed to get spaces: %w", err)
		}
		if !opts.JSON {
			fmt.Fprintln(opts.IO.Out, " done")
		}
	} else {
		// Single page fetch
		result, err := confluence.GetSpaces(ctx, opts.Limit, opts.Cursor)
		if err != nil {
			return fmt.Errorf("failed to get spaces: %w", err)
		}
		spaces = result.Results

		if result.Links != nil && result.Links.Next != "" {
			hasMore = true
			// Extract cursor for user
			nextCursor = extractCursorFromURL(result.Links.Next)
		}
	}

	listOutput := &SpaceListOutput{
		Spaces:     make([]*SpaceOutput, 0, len(spaces)),
		Total:      len(spaces),
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}

	for _, space := range spaces {
		listOutput.Spaces = append(listOutput.Spaces, &SpaceOutput{
			ID:     space.ID,
			Key:    space.Key,
			Name:   space.Name,
			Type:   space.Type,
			Status: space.Status,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, listOutput)
	}

	if len(listOutput.Spaces) == 0 {
		fmt.Fprintln(opts.IO.Out, "No spaces found.")
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Found %d spaces\n\n", listOutput.Total)

	headers := []string{"KEY", "NAME", "TYPE", "STATUS"}
	rows := make([][]string, 0, len(listOutput.Spaces))

	for _, space := range listOutput.Spaces {
		rows = append(rows, []string{
			space.Key,
			space.Name,
			space.Type,
			space.Status,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)

	// Show pagination hint
	if hasMore && nextCursor != "" {
		fmt.Fprintf(opts.IO.Out, "\nMore spaces available. Use --cursor %s to see next page, or --all to fetch everything\n", nextCursor)
	}

	return nil
}

// extractCursorFromURL extracts the cursor parameter from a pagination URL.
func extractCursorFromURL(nextURL string) string {
	// Simple extraction - find cursor= in the URL
	const prefix = "cursor="
	start := 0
	for i := 0; i < len(nextURL)-len(prefix); i++ {
		if nextURL[i:i+len(prefix)] == prefix {
			start = i + len(prefix)
			break
		}
	}
	if start == 0 {
		return ""
	}
	// Find end of cursor value
	end := start
	for end < len(nextURL) && nextURL[end] != '&' {
		end++
	}
	return nextURL[start:end]
}
