package comment

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
	IO       *iostreams.IOStreams
	IssueKey string
	JSON     bool
}

// NewCmdList creates the list command.
func NewCmdList(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:     "list <issue-key>",
		Aliases: []string{"ls"},
		Short:   "List comments on an issue",
		Long:    `View all comments on a Jira issue.`,
		Example: `  # List comments on an issue
  atl issue comment list PROJ-1234

  # Output as JSON
  atl issue comment list PROJ-1234 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]
			return runList(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// CommentListOutput represents the list of comments.
type CommentListOutput struct {
	IssueKey string           `json:"issue_key"`
	Comments []*CommentOutput `json:"comments"`
	Total    int              `json:"total"`
}

// CommentOutput represents a single comment.
type CommentOutput struct {
	ID         string `json:"id"`
	Author     string `json:"author"`
	Body       string `json:"body"`
	Created    string `json:"created"`
	Updated    string `json:"updated,omitempty"`
	Visibility string `json:"visibility,omitempty"`
}

func runList(opts *ListOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	comments, err := jira.GetComments(ctx, opts.IssueKey)
	if err != nil {
		return fmt.Errorf("failed to get comments: %w", err)
	}

	listOutput := &CommentListOutput{
		IssueKey: opts.IssueKey,
		Comments: make([]*CommentOutput, 0, len(comments)),
		Total:    len(comments),
	}

	for _, c := range comments {
		comment := &CommentOutput{
			ID:      c.ID,
			Created: formatTime(c.Created),
			Updated: formatTime(c.Updated),
		}
		if c.Author != nil {
			comment.Author = c.Author.DisplayName
		}
		if c.Body != nil {
			comment.Body = api.ADFToText(c.Body)
		}
		listOutput.Comments = append(listOutput.Comments, comment)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, listOutput)
	}

	if len(listOutput.Comments) == 0 {
		fmt.Fprintf(opts.IO.Out, "No comments on %s\n", opts.IssueKey)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "# Comments on %s (%d total)\n\n", opts.IssueKey, listOutput.Total)

	for i, c := range listOutput.Comments {
		if i > 0 {
			fmt.Fprintln(opts.IO.Out, "---")
		}
		fmt.Fprintf(opts.IO.Out, "**%s** (%s) [ID: %s]\n\n", c.Author, c.Created, c.ID)
		fmt.Fprintln(opts.IO.Out, c.Body)
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
