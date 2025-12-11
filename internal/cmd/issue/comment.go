package issue

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// CommentOptions holds the options for the comment command.
type CommentOptions struct {
	IO       *iostreams.IOStreams
	IssueKey string
	Body     string
	List     bool
	JSON     bool
}

// NewCmdComment creates the comment command.
func NewCmdComment(ios *iostreams.IOStreams) *cobra.Command {
	opts := &CommentOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "comment <issue-key>",
		Short: "Add or view comments on an issue",
		Long:  `Add a comment to a Jira issue or view existing comments.`,
		Example: `  # Add a comment
  atl issue comment PROJ-1234 --body "This is my comment"

  # View comments on an issue
  atl issue comment PROJ-1234 --list

  # Output as JSON
  atl issue comment PROJ-1234 --list --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]
			return runComment(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Comment text")
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List existing comments")
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
	ID      string `json:"id"`
	Author  string `json:"author"`
	Body    string `json:"body"`
	Created string `json:"created"`
	Updated string `json:"updated,omitempty"`
}

// AddCommentOutput represents the result of adding a comment.
type AddCommentOutput struct {
	IssueKey  string `json:"issue_key"`
	CommentID string `json:"comment_id"`
	URL       string `json:"url"`
}

func runComment(opts *CommentOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	if opts.List {
		return listComments(ctx, jira, client.Hostname(), opts)
	}

	if opts.Body == "" {
		return fmt.Errorf("either --body or --list must be specified")
	}

	return addComment(ctx, jira, client.Hostname(), opts)
}

func listComments(ctx context.Context, jira *api.JiraService, hostname string, opts *CommentOptions) error {
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
		fmt.Fprintf(opts.IO.Out, "**%s** (%s)\n\n", c.Author, c.Created)
		fmt.Fprintln(opts.IO.Out, c.Body)
		fmt.Fprintln(opts.IO.Out)
	}

	return nil
}

func addComment(ctx context.Context, jira *api.JiraService, hostname string, opts *CommentOptions) error {
	comment, err := jira.AddComment(ctx, opts.IssueKey, opts.Body)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	addOutput := &AddCommentOutput{
		IssueKey:  opts.IssueKey,
		CommentID: comment.ID,
		URL:       fmt.Sprintf("https://%s/browse/%s?focusedCommentId=%s", hostname, opts.IssueKey, comment.ID),
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, addOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Added comment to %s\n", opts.IssueKey)
	fmt.Fprintf(opts.IO.Out, "Comment ID: %s\n", addOutput.CommentID)
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", addOutput.URL)

	return nil
}
