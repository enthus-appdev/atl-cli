package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// ChildrenOptions holds the options for the children command.
type ChildrenOptions struct {
	IO          *iostreams.IOStreams
	PageID      string
	Descendants bool
	All         bool
	JSON        bool
	Type        string
}

// NewCmdChildren creates the children command.
func NewCmdChildren(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ChildrenOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "children <page-id>",
		Short: "List child pages of a Confluence page",
		Long: `List child pages of a Confluence page.

By default, lists only immediate children. Use --descendants to include
all nested pages (grandchildren, etc.).`,
		Example: `  # List immediate children of a page
  atl confluence page children 123456

  # List all descendants (nested pages)
  atl confluence page children 123456 --descendants

  # List only folders
  atl confluence page children 123456 --type folder

  # List only pages (no folders)
  atl confluence page children 123456 --type page

  # Output as JSON
  atl confluence page children 123456 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageID = args[0]
			return runChildren(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Descendants, "descendants", "d", false, "Include all descendants (not just immediate children)")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Fetch all pages (follow pagination)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	cmd.Flags().StringVarP(&opts.Type, "type", "t", "", "Filter by type: 'page' or 'folder'")

	return cmd
}

// ChildOutput represents a child page in the output.
type ChildOutput struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	ParentID string `json:"parent_id,omitempty"`
	Depth    int    `json:"depth,omitempty"`
}

// ChildrenOutput represents the output for children list.
type ChildrenOutput struct {
	PageID   string         `json:"page_id"`
	Children []*ChildOutput `json:"children"`
	Total    int            `json:"total"`
}

func runChildren(opts *ChildrenOptions) error {
	// Validate type filter
	if opts.Type != "" && opts.Type != "page" && opts.Type != "folder" {
		return fmt.Errorf("--type must be 'page' or 'folder', got '%s'", opts.Type)
	}

	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	var children []*api.PageChild

	if opts.Descendants {
		if opts.All {
			if !opts.JSON {
				fmt.Fprint(opts.IO.Out, "Fetching all descendants...")
			}
			children, err = confluence.GetPageDescendantsAll(ctx, opts.PageID)
			if !opts.JSON {
				fmt.Fprintln(opts.IO.Out, " done")
			}
		} else {
			result, err := confluence.GetPageDescendants(ctx, opts.PageID, 100, "")
			if err != nil {
				return fmt.Errorf("failed to get descendants: %w", err)
			}
			children = result.Results
		}
	} else {
		result, err := confluence.GetPageChildren(ctx, opts.PageID, 100, "")
		if err != nil {
			return fmt.Errorf("failed to get children: %w", err)
		}
		children = result.Results
	}

	if err != nil {
		return fmt.Errorf("failed to get children: %w", err)
	}

	// Filter by type if specified
	if opts.Type != "" {
		filtered := make([]*api.PageChild, 0)
		for _, child := range children {
			if child.Type == opts.Type {
				filtered = append(filtered, child)
			}
		}
		children = filtered
	}

	childrenOutput := &ChildrenOutput{
		PageID:   opts.PageID,
		Children: make([]*ChildOutput, 0, len(children)),
		Total:    len(children),
	}

	for _, child := range children {
		childrenOutput.Children = append(childrenOutput.Children, &ChildOutput{
			ID:       child.ID,
			Title:    child.Title,
			Status:   child.Status,
			Type:     child.Type,
			ParentID: child.ParentID,
			Depth:    child.Depth,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, childrenOutput)
	}

	if len(childrenOutput.Children) == 0 {
		fmt.Fprintf(opts.IO.Out, "No child pages found for page %s\n", opts.PageID)
		return nil
	}

	what := "children"
	if opts.Descendants {
		what = "descendants"
	}
	fmt.Fprintf(opts.IO.Out, "Found %d %s of page %s\n\n", childrenOutput.Total, what, opts.PageID)

	if opts.Descendants {
		headers := []string{"ID", "TITLE", "TYPE", "DEPTH", "STATUS"}
		rows := make([][]string, 0, len(childrenOutput.Children))

		for _, child := range childrenOutput.Children {
			title := child.Title
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			rows = append(rows, []string{
				child.ID,
				title,
				child.Type,
				fmt.Sprintf("%d", child.Depth),
				child.Status,
			})
		}

		output.SimpleTable(opts.IO.Out, headers, rows)
	} else {
		headers := []string{"ID", "TITLE", "TYPE", "STATUS"}
		rows := make([][]string, 0, len(childrenOutput.Children))

		for _, child := range childrenOutput.Children {
			title := child.Title
			if len(title) > 55 {
				title = title[:52] + "..."
			}
			rows = append(rows, []string{
				child.ID,
				title,
				child.Type,
				child.Status,
			})
		}

		output.SimpleTable(opts.IO.Out, headers, rows)
	}

	return nil
}
