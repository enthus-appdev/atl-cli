package comment

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// AddOptions holds the options for the add command.
type AddOptions struct {
	IO       *iostreams.IOStreams
	PageID   string
	Body     string
	BodyFile string
	ReplyTo  string
	JSON     bool
}

// NewCmdAdd creates the add command.
func NewCmdAdd(ios *iostreams.IOStreams) *cobra.Command {
	opts := &AddOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "add <page-id>",
		Short: "Add a comment to a page",
		Long: `Add a new footer comment to a Confluence page.

The body must be HTML (Confluence storage format).
Use --reply-to to create a threaded reply to an existing comment.`,
		Example: `  # Add a comment
  atl confluence comment add 1234567 --body "<p>This looks good!</p>"

  # Add a comment from a file
  atl confluence comment add 1234567 --body-file comment.html

  # Reply to an existing comment
  atl confluence comment add 1234567 --body "<p>I agree</p>" --reply-to 9876543

  # Output as JSON
  atl confluence comment add 1234567 --body "<p>Comment</p>" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageID = args[0]

			if err := resolveBody(&opts.Body, opts.BodyFile); err != nil {
				return err
			}
			if opts.Body == "" {
				return fmt.Errorf("--body or --body-file is required")
			}

			return runAdd(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Comment body in HTML (required)")
	cmd.Flags().StringVar(&opts.BodyFile, "body-file", "", "Read comment body from file")
	cmd.Flags().StringVar(&opts.ReplyTo, "reply-to", "", "Comment ID to reply to (creates threaded reply)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// AddCommentOutput represents the result of adding a comment.
type AddCommentOutput struct {
	PageID    string `json:"page_id"`
	CommentID string `json:"comment_id"`
	Action    string `json:"action"`
}

func runAdd(opts *AddOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	comment, err := confluence.CreateFooterComment(ctx, opts.PageID, opts.Body, opts.ReplyTo)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	addOutput := &AddCommentOutput{
		PageID:    opts.PageID,
		CommentID: comment.ID,
		Action:    "added",
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, addOutput)
	}

	action := "Added comment to"
	if opts.ReplyTo != "" {
		action = fmt.Sprintf("Replied to comment %s on", opts.ReplyTo)
	}
	fmt.Fprintf(opts.IO.Out, "%s page %s\n", action, opts.PageID)
	fmt.Fprintf(opts.IO.Out, "Comment ID: %s\n", addOutput.CommentID)

	return nil
}
