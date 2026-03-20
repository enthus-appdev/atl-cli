package comment

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// EditOptions holds the options for the edit command.
type EditOptions struct {
	IO        *iostreams.IOStreams
	CommentID string
	Body      string
	BodyFile  string
	JSON      bool
}

// NewCmdEdit creates the edit command.
func NewCmdEdit(ios *iostreams.IOStreams) *cobra.Command {
	opts := &EditOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a comment on a page",
		Long: `Edit an existing footer comment on a Confluence page.

Requires the comment ID which can be found using 'atl confluence comment list'.`,
		Example: `  # Edit a comment
  atl confluence comment edit --id 9876543 --body "<p>Updated text</p>"

  # Edit from file
  atl confluence comment edit --id 9876543 --body-file updated.html

  # Output as JSON
  atl confluence comment edit --id 9876543 --body "<p>Text</p>" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.CommentID == "" {
				return fmt.Errorf("--id is required\n\nUse 'atl confluence comment list <page-id>' to see comment IDs")
			}
			if err := resolveBody(&opts.Body, opts.BodyFile); err != nil {
				return err
			}
			if opts.Body == "" {
				return fmt.Errorf("--body or --body-file is required")
			}

			return runEdit(opts)
		},
	}

	cmd.Flags().StringVar(&opts.CommentID, "id", "", "Comment ID to edit (required)")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New comment body in HTML (required)")
	cmd.Flags().StringVar(&opts.BodyFile, "body-file", "", "Read comment body from file")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

func runEdit(opts *EditOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	// Get existing comment to determine current version
	existing, err := confluence.GetFooterComment(ctx, opts.CommentID)
	if err != nil {
		return fmt.Errorf("failed to get comment: %w", err)
	}

	version := 0
	if existing.Version != nil {
		version = existing.Version.Number
	}

	comment, err := confluence.UpdateFooterComment(ctx, opts.CommentID, opts.Body, version)
	if err != nil {
		return fmt.Errorf("failed to edit comment: %w", err)
	}

	editOutput := &AddCommentOutput{
		PageID:    existing.PageID,
		CommentID: comment.ID,
		Action:    "edited",
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, editOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Edited comment %s\n", editOutput.CommentID)

	return nil
}
