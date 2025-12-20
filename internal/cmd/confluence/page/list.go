package page

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
	Space  string
	Status string
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
		Short:   "List pages in a space",
		Long: `List Confluence pages in a specified space.

The --space flag is required. Use 'atl confluence space list' to see available spaces.

Filter by status to see draft or archived pages.`,
		Example: `  # List pages in a space
  atl confluence page list --space DOCS

  # List draft pages
  atl confluence page list --space DOCS --status draft

  # List archived pages
  atl confluence page list --space DOCS --status archived

  # List more pages
  atl confluence page list --space DOCS --limit 100

  # Fetch all pages in a space
  atl confluence page list --space DOCS --all

  # Get next page using cursor
  atl confluence page list --space DOCS --cursor <cursor>

  # Output as JSON
  atl confluence page list --space DOCS --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Space == "" {
				return fmt.Errorf("--space flag is required\n\nUse 'atl confluence space list' to see available spaces")
			}
			return runList(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Space, "space", "s", "", "Space key (required)")
	cmd.Flags().StringVar(&opts.Status, "status", "", "Filter by status: current, draft, archived (default: current)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 25, "Maximum number of pages per page")
	cmd.Flags().StringVar(&opts.Cursor, "cursor", "", "Pagination cursor for next page")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Fetch all pages (ignores --limit and --cursor)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// PageListOutput represents the output for page list.
type PageListOutput struct {
	SpaceKey   string        `json:"space_key"`
	Pages      []*PageOutput `json:"pages"`
	Total      int           `json:"total"`
	HasMore    bool          `json:"has_more"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

// PageOutput represents a single page in the list.
type PageOutput struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
}

func runList(opts *ListOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	// First get the space to get its ID
	space, err := confluence.GetSpaceByKey(ctx, opts.Space)
	if err != nil {
		return fmt.Errorf("failed to get space: %w", err)
	}

	var pages []*api.Page
	var nextCursor string
	var hasMore bool

	if opts.All {
		// Fetch all pages
		if !opts.JSON {
			fmt.Fprint(opts.IO.Out, "Fetching all pages...")
		}
		pages, err = confluence.GetPagesAll(ctx, space.ID, opts.Status)
		if err != nil {
			return fmt.Errorf("failed to get pages: %w", err)
		}
		if !opts.JSON {
			fmt.Fprintln(opts.IO.Out, " done")
		}
	} else {
		// Single page fetch
		result, err := confluence.GetPages(ctx, space.ID, opts.Limit, opts.Cursor, opts.Status)
		if err != nil {
			return fmt.Errorf("failed to get pages: %w", err)
		}
		pages = result.Results

		if result.Links != nil && result.Links.Next != "" {
			hasMore = true
			nextCursor = extractCursorFromURL(result.Links.Next)
		}
	}

	listOutput := &PageListOutput{
		SpaceKey:   opts.Space,
		Pages:      make([]*PageOutput, 0, len(pages)),
		Total:      len(pages),
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}

	for _, page := range pages {
		listOutput.Pages = append(listOutput.Pages, &PageOutput{
			ID:        page.ID,
			Title:     page.Title,
			Status:    page.Status,
			CreatedAt: page.CreatedAt,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, listOutput)
	}

	if len(listOutput.Pages) == 0 {
		fmt.Fprintf(opts.IO.Out, "No pages found in space %s\n", opts.Space)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Found %d pages in space %s\n\n", listOutput.Total, opts.Space)

	headers := []string{"ID", "TITLE", "STATUS"}
	rows := make([][]string, 0, len(listOutput.Pages))

	for _, page := range listOutput.Pages {
		title := page.Title
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		rows = append(rows, []string{
			page.ID,
			title,
			page.Status,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)

	// Show pagination hint
	if hasMore && nextCursor != "" {
		fmt.Fprintf(opts.IO.Out, "\nMore pages available. Use --cursor %s to see next page, or --all to fetch everything\n", nextCursor)
	}

	return nil
}

// extractCursorFromURL extracts the cursor parameter from a pagination URL.
func extractCursorFromURL(nextURL string) string {
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
	end := start
	for end < len(nextURL) && nextURL[end] != '&' {
		end++
	}
	return nextURL[start:end]
}
