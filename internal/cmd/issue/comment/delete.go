package comment

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// DeleteOptions holds the options for the delete command.
type DeleteOptions struct {
	IO        *iostreams.IOStreams
	IssueKey  string
	CommentID string
	Force     bool
	JSON      bool
}

// NewCmdDelete creates the delete command.
func NewCmdDelete(ios *iostreams.IOStreams) *cobra.Command {
	opts := &DeleteOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:     "delete <issue-key>",
		Aliases: []string{"rm"},
		Short:   "Delete a comment from an issue",
		Long: `Delete an existing comment from a Jira issue.

Requires the comment ID which can be found using 'atl issue comment list'.`,
		Example: `  # Delete a comment (prompts for confirmation)
  atl issue comment delete PROJ-1234 --id 12345

  # Delete without confirmation
  atl issue comment delete PROJ-1234 --id 12345 --force

  # Output as JSON
  atl issue comment delete PROJ-1234 --id 12345 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]

			if opts.CommentID == "" {
				return fmt.Errorf("--id is required\n\nUse 'atl issue comment list %s' to see comment IDs", args[0])
			}

			return runDelete(opts)
		},
	}

	cmd.Flags().StringVar(&opts.CommentID, "id", "", "Comment ID to delete (required)")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

func runDelete(opts *DeleteOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)
	hostname := client.Hostname()

	// Confirm deletion unless --force
	if !opts.Force && !opts.JSON {
		fmt.Fprintf(opts.IO.Out, "Delete comment %s from %s? [y/N]: ", opts.CommentID, opts.IssueKey)
		var confirm string
		fmt.Fscanln(opts.IO.In, &confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Fprintln(opts.IO.Out, "Canceled")
			return nil
		}
	}

	err = jira.DeleteComment(ctx, opts.IssueKey, opts.CommentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	deleteOutput := &AddCommentOutput{
		IssueKey:  opts.IssueKey,
		CommentID: opts.CommentID,
		Action:    "deleted",
		URL:       fmt.Sprintf("https://%s/browse/%s", hostname, opts.IssueKey),
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, deleteOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Deleted comment %s from %s\n", opts.CommentID, opts.IssueKey)

	return nil
}
