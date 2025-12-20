package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// SearchOptions holds the options for the search command.
type SearchOptions struct {
	IO       *iostreams.IOStreams
	Query    string
	Space    string
	CQL      string
	Limit    int
	JSON     bool
}

// NewCmdSearch creates the search command.
func NewCmdSearch(ios *iostreams.IOStreams) *cobra.Command {
	opts := &SearchOptions{
		IO:    ios,
		Limit: 25,
	}

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search for Confluence pages",
		Long: `Search for Confluence pages by title or using CQL (Confluence Query Language).

By default, searches page titles. Use --cql for advanced searches.`,
		Example: `  # Search for pages with "API" in the title
  atl confluence page search --query "API"

  # Search in a specific space
  atl confluence page search --query "documentation" --space CTO

  # Search using CQL
  atl confluence page search --cql "type = page AND text ~ 'kubernetes'"

  # Output as JSON
  atl confluence page search --query "test" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Query == "" && opts.CQL == "" {
				return fmt.Errorf("either --query or --cql flag is required")
			}
			return runSearch(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Query, "query", "q", "", "Search term for page titles")
	cmd.Flags().StringVarP(&opts.Space, "space", "s", "", "Limit search to a specific space (key)")
	cmd.Flags().StringVar(&opts.CQL, "cql", "", "CQL query for advanced searches")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 25, "Maximum number of results")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// SearchResultOutput represents a search result in the output.
type SearchResultOutput struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SpaceKey string `json:"space_key,omitempty"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Excerpt  string `json:"excerpt,omitempty"`
}

// SearchOutput represents the output for search results.
type SearchOutput struct {
	Query   string                `json:"query"`
	Results []*SearchResultOutput `json:"results"`
	Total   int                   `json:"total"`
}

func runSearch(opts *SearchOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	var result *api.ConfluenceSearchResponse

	if opts.CQL != "" {
		// Use CQL search
		result, err = confluence.SearchWithCQL(ctx, opts.CQL, opts.Limit, "")
	} else {
		// Search by title
		result, err = confluence.SearchByTitle(ctx, opts.Query, opts.Space, opts.Limit)
	}

	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	searchOutput := &SearchOutput{
		Query:   opts.Query,
		Results: make([]*SearchResultOutput, 0, len(result.Results)),
		Total:   len(result.Results),
	}

	if opts.CQL != "" {
		searchOutput.Query = opts.CQL
	}

	for _, r := range result.Results {
		searchOutput.Results = append(searchOutput.Results, &SearchResultOutput{
			ID:       r.ID,
			Title:    r.Title,
			SpaceKey: r.SpaceKey,
			Type:     r.Type,
			Status:   r.Status,
			Excerpt:  r.Excerpt,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, searchOutput)
	}

	if len(searchOutput.Results) == 0 {
		fmt.Fprintf(opts.IO.Out, "No pages found matching '%s'\n", searchOutput.Query)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Found %d pages:\n\n", searchOutput.Total)

	headers := []string{"ID", "TITLE", "SPACE", "STATUS"}
	rows := make([][]string, 0, len(searchOutput.Results))

	for _, r := range searchOutput.Results {
		title := r.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		rows = append(rows, []string{
			r.ID,
			title,
			r.SpaceKey,
			r.Status,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)
	return nil
}
