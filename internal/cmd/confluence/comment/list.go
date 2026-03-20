package comment

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// ListOptions holds the options for the list command.
type ListOptions struct {
	IO     *iostreams.IOStreams
	PageID string
	JSON   bool
}

// NewCmdList creates the list command.
func NewCmdList(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:     "list <page-id>",
		Aliases: []string{"ls"},
		Short:   "List comments on a page",
		Long:    `View all footer comments on a Confluence page.`,
		Example: `  # List comments on a page
  atl confluence comment list 1234567

  # Output as JSON
  atl confluence comment list 1234567 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageID = args[0]
			return runList(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// CommentListOutput represents the list of comments.
type CommentListOutput struct {
	PageID   string           `json:"page_id"`
	Comments []*CommentOutput `json:"comments"`
	Total    int              `json:"total"`
}

// CommentOutput represents a single comment.
type CommentOutput struct {
	ID        string `json:"id"`
	AuthorID  string `json:"author_id"`
	Body      string `json:"body"`
	ParentID  string `json:"parent_id,omitempty"`
	CreatedAt string `json:"created_at"`
	Version   int    `json:"version"`
}

func runList(opts *ListOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	comments, err := confluence.GetPageFooterCommentsAll(ctx, opts.PageID)
	if err != nil {
		return fmt.Errorf("failed to get comments: %w", err)
	}

	listOutput := &CommentListOutput{
		PageID:   opts.PageID,
		Comments: make([]*CommentOutput, 0, len(comments)),
		Total:    len(comments),
	}

	for _, c := range comments {
		comment := &CommentOutput{
			ID:        c.ID,
			AuthorID:  c.AuthorID,
			ParentID:  c.ParentID,
			CreatedAt: formatTime(c.CreatedAt),
		}
		if c.Version != nil {
			comment.Version = c.Version.Number
		}
		if c.Body != nil && c.Body.Storage != nil {
			comment.Body = c.Body.Storage.Value
		}
		listOutput.Comments = append(listOutput.Comments, comment)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, listOutput)
	}

	if len(listOutput.Comments) == 0 {
		fmt.Fprintf(opts.IO.Out, "No comments on page %s\n", opts.PageID)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "# Comments on page %s (%d total)\n\n", opts.PageID, listOutput.Total)

	for i, c := range listOutput.Comments {
		if i > 0 {
			fmt.Fprintln(opts.IO.Out, "---")
		}
		prefix := ""
		if c.ParentID != "" {
			prefix = fmt.Sprintf(" (reply to %s)", c.ParentID)
		}
		fmt.Fprintf(opts.IO.Out, "**%s** (%s) [ID: %s]%s\n\n", c.AuthorID, c.CreatedAt, c.ID, prefix)
		fmt.Fprintln(opts.IO.Out, stripHTML(c.Body))
		fmt.Fprintln(opts.IO.Out)
	}

	return nil
}

func formatTime(t string) string {
	if len(t) >= 19 {
		return t[:10] + " " + t[11:19]
	}
	return t
}

// stripHTML removes basic HTML tags for plain-text display.
func stripHTML(s string) string {
	// Simple tag stripping for display purposes
	result := s
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return strings.TrimSpace(result)
}
