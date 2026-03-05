package comment

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// EditOptions holds the options for the edit command.
type EditOptions struct {
	IO             *iostreams.IOStreams
	IssueKey       string
	CommentID      string
	Body           string
	BodyFile       string
	VisibilityType string
	VisibilityName string
	JSON           bool
}

// NewCmdEdit creates the edit command.
func NewCmdEdit(ios *iostreams.IOStreams) *cobra.Command {
	opts := &EditOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "edit <issue-key>",
		Short: "Edit a comment on an issue",
		Long: `Edit an existing comment on a Jira issue.

Requires the comment ID which can be found using 'atl issue comment list'.`,
		Example: `  # Edit a comment
  atl issue comment edit PROJ-1234 --id 12345 --body "Updated comment text"

  # Update visibility while editing
  atl issue comment edit PROJ-1234 --id 12345 --body "Text" --visibility-type role --visibility-name "Developers"

  # Read comment body from a file
  atl issue comment edit PROJ-1234 --id 12345 --body-file comment.md

  # Output as JSON
  atl issue comment edit PROJ-1234 --id 12345 --body "Text" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]

			if opts.CommentID == "" {
				return fmt.Errorf("--id is required\n\nUse 'atl issue comment list %s' to see comment IDs", args[0])
			}
			if opts.Body != "" && opts.BodyFile != "" {
				return fmt.Errorf("--body and --body-file are mutually exclusive")
			}
			if opts.Body == "" && opts.BodyFile == "" {
				return fmt.Errorf("--body or --body-file is required")
			}
			if opts.BodyFile != "" {
				data, err := os.ReadFile(opts.BodyFile)
				if err != nil {
					return fmt.Errorf("failed to read body file: %w", err)
				}
				opts.Body = string(data)
			}

			return runEdit(opts)
		},
	}

	cmd.Flags().StringVar(&opts.CommentID, "id", "", "Comment ID to edit (required)")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New comment text (mutually exclusive with --body-file)")
	cmd.Flags().StringVar(&opts.BodyFile, "body-file", "", "Read comment body from file (mutually exclusive with --body)")
	cmd.Flags().StringVar(&opts.VisibilityType, "visibility-type", "", "Visibility type: 'role' or 'group'")
	cmd.Flags().StringVar(&opts.VisibilityName, "visibility-name", "", "Role or group name for visibility restriction")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

func runEdit(opts *EditOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)
	hostname := client.Hostname()

	commentOpts := &api.CommentOptions{
		Body:           opts.Body,
		VisibilityType: opts.VisibilityType,
		VisibilityName: opts.VisibilityName,
	}

	comment, err := jira.UpdateComment(ctx, opts.IssueKey, opts.CommentID, commentOpts)
	if err != nil {
		return fmt.Errorf("failed to edit comment: %w", err)
	}

	editOutput := &AddCommentOutput{
		IssueKey:  opts.IssueKey,
		CommentID: comment.ID,
		Action:    "edited",
		URL:       fmt.Sprintf("https://%s/browse/%s?focusedCommentId=%s", hostname, opts.IssueKey, comment.ID),
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, editOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Edited comment on %s\n", opts.IssueKey)
	fmt.Fprintf(opts.IO.Out, "Comment ID: %s\n", editOutput.CommentID)
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", editOutput.URL)

	return nil
}
