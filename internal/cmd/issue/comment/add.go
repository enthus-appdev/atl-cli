package comment

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// AddOptions holds the options for the add command.
type AddOptions struct {
	IO             *iostreams.IOStreams
	IssueKey       string
	Body           string
	BodyFile       string
	ReplyTo        string
	VisibilityType string
	VisibilityName string
	JSON           bool
}

// NewCmdAdd creates the add command.
func NewCmdAdd(ios *iostreams.IOStreams) *cobra.Command {
	opts := &AddOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "add <issue-key>",
		Short: "Add a comment to an issue",
		Long: `Add a new comment to a Jira issue.

Supports visibility restrictions to limit who can see the comment,
and replying to existing comments with automatic quoting.`,
		Example: `  # Add a comment
  atl issue comment add PROJ-1234 --body "This is my comment"

  # Add an internal comment (visible only to a role)
  atl issue comment add PROJ-1234 --body "Internal note" --visibility-type role --visibility-name "Developers"

  # Add a comment visible only to a group
  atl issue comment add PROJ-1234 --body "Team note" --visibility-type group --visibility-name "jira-developers"

  # Reply to a specific comment (quotes the original)
  atl issue comment add PROJ-1234 --body "I agree!" --reply-to 12345

  # Read comment body from a file
  atl issue comment add PROJ-1234 --body-file comment.md

  # Output as JSON
  atl issue comment add PROJ-1234 --body "Comment" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]

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

			return runAdd(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Comment text (mutually exclusive with --body-file)")
	cmd.Flags().StringVar(&opts.BodyFile, "body-file", "", "Read comment body from file (mutually exclusive with --body)")
	cmd.Flags().StringVar(&opts.ReplyTo, "reply-to", "", "Comment ID to reply to (quotes original)")
	cmd.Flags().StringVar(&opts.VisibilityType, "visibility-type", "", "Visibility type: 'role' or 'group'")
	cmd.Flags().StringVar(&opts.VisibilityName, "visibility-name", "", "Role or group name for visibility restriction")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// AddCommentOutput represents the result of adding a comment.
type AddCommentOutput struct {
	IssueKey  string `json:"issue_key"`
	CommentID string `json:"comment_id"`
	Action    string `json:"action"`
	URL       string `json:"url"`
}

func runAdd(opts *AddOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)
	hostname := client.Hostname()

	// Handle reply
	if opts.ReplyTo != "" {
		return replyToComment(ctx, jira, hostname, opts)
	}

	commentOpts := &api.CommentOptions{
		Body:           opts.Body,
		VisibilityType: opts.VisibilityType,
		VisibilityName: opts.VisibilityName,
	}

	comment, err := jira.AddCommentWithOptions(ctx, opts.IssueKey, commentOpts)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	addOutput := &AddCommentOutput{
		IssueKey:  opts.IssueKey,
		CommentID: comment.ID,
		Action:    "added",
		URL:       fmt.Sprintf("https://%s/browse/%s?focusedCommentId=%s", hostname, opts.IssueKey, comment.ID),
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, addOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Added comment to %s\n", opts.IssueKey)
	fmt.Fprintf(opts.IO.Out, "Comment ID: %s\n", addOutput.CommentID)
	if opts.VisibilityType != "" {
		fmt.Fprintf(opts.IO.Out, "Visibility: %s '%s'\n", opts.VisibilityType, opts.VisibilityName)
	}
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", addOutput.URL)

	return nil
}

func replyToComment(ctx context.Context, jira *api.JiraService, hostname string, opts *AddOptions) error {
	// Get the original comment to quote it
	originalComment, err := jira.GetComment(ctx, opts.IssueKey, opts.ReplyTo)
	if err != nil {
		return fmt.Errorf("failed to get original comment: %w", err)
	}

	// Build the reply with a quote of the original
	originalText := ""
	if originalComment.Body != nil {
		originalText = api.ADFToText(originalComment.Body)
	}
	originalAuthor := "Unknown"
	if originalComment.Author != nil {
		originalAuthor = originalComment.Author.DisplayName
	}

	// Create quoted reply using Markdown blockquote syntax
	quotedLines := strings.Split(originalText, "\n")
	var quoted strings.Builder
	fmt.Fprintf(&quoted, "_Replying to %s:_\n\n", originalAuthor)
	for _, line := range quotedLines {
		quoted.WriteString("> ")
		quoted.WriteString(line)
		quoted.WriteString("\n")
	}
	quoted.WriteString("\n")
	quoted.WriteString(opts.Body)

	commentOpts := &api.CommentOptions{
		Body:           quoted.String(),
		VisibilityType: opts.VisibilityType,
		VisibilityName: opts.VisibilityName,
	}

	comment, err := jira.AddCommentWithOptions(ctx, opts.IssueKey, commentOpts)
	if err != nil {
		return fmt.Errorf("failed to add reply: %w", err)
	}

	replyOutput := &AddCommentOutput{
		IssueKey:  opts.IssueKey,
		CommentID: comment.ID,
		Action:    "replied",
		URL:       fmt.Sprintf("https://%s/browse/%s?focusedCommentId=%s", hostname, opts.IssueKey, comment.ID),
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, replyOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Replied to comment %s on %s\n", opts.ReplyTo, opts.IssueKey)
	fmt.Fprintf(opts.IO.Out, "New comment ID: %s\n", replyOutput.CommentID)
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", replyOutput.URL)

	return nil
}
