package issue

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// CommentOptions holds the options for the comment command.
type CommentOptions struct {
	IO             *iostreams.IOStreams
	IssueKey       string
	Body           string
	CommentID      string
	ReplyTo        string
	Edit           bool
	Delete         bool
	List           bool
	VisibilityType string
	VisibilityName string
	JSON           bool
}

// NewCmdComment creates the comment command.
func NewCmdComment(ios *iostreams.IOStreams) *cobra.Command {
	opts := &CommentOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "comment <issue-key>",
		Short: "Add, edit, delete, or view comments on an issue",
		Long: `Manage comments on a Jira issue.

Supports adding new comments, editing existing ones, replying to comments,
and restricting visibility to specific roles or groups.`,
		Example: `  # Add a comment
  atl issue comment PROJ-1234 --body "This is my comment"

  # Add an internal comment (visible only to a role)
  atl issue comment PROJ-1234 --body "Internal note" --visibility-type role --visibility-name "Developers"

  # Add a comment visible only to a group
  atl issue comment PROJ-1234 --body "Team note" --visibility-type group --visibility-name "jira-developers"

  # Reply to a specific comment (quotes the original)
  atl issue comment PROJ-1234 --body "I agree!" --reply-to 12345

  # Edit an existing comment
  atl issue comment PROJ-1234 --edit --comment-id 12345 --body "Updated comment text"

  # Delete a comment
  atl issue comment PROJ-1234 --delete --comment-id 12345

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
	cmd.Flags().StringVar(&opts.CommentID, "comment-id", "", "Comment ID (for edit/delete)")
	cmd.Flags().StringVar(&opts.ReplyTo, "reply-to", "", "Comment ID to reply to (quotes original)")
	cmd.Flags().BoolVarP(&opts.Edit, "edit", "e", false, "Edit an existing comment")
	cmd.Flags().BoolVarP(&opts.Delete, "delete", "D", false, "Delete a comment")
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List existing comments")
	cmd.Flags().StringVar(&opts.VisibilityType, "visibility-type", "", "Visibility type: 'role' or 'group'")
	cmd.Flags().StringVar(&opts.VisibilityName, "visibility-name", "", "Role or group name for visibility restriction")
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

// AddCommentOutput represents the result of adding a comment.
type AddCommentOutput struct {
	IssueKey  string `json:"issue_key"`
	CommentID string `json:"comment_id"`
	Action    string `json:"action"`
	URL       string `json:"url"`
}

func runComment(opts *CommentOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Handle list
	if opts.List {
		return listComments(ctx, jira, client.Hostname(), opts)
	}

	// Handle delete
	if opts.Delete {
		if opts.CommentID == "" {
			return fmt.Errorf("--comment-id is required for delete")
		}
		return deleteComment(ctx, jira, client.Hostname(), opts)
	}

	// Handle edit
	if opts.Edit {
		if opts.CommentID == "" {
			return fmt.Errorf("--comment-id is required for edit")
		}
		if opts.Body == "" {
			return fmt.Errorf("--body is required for edit")
		}
		return editComment(ctx, jira, client.Hostname(), opts)
	}

	// Handle reply
	if opts.ReplyTo != "" {
		if opts.Body == "" {
			return fmt.Errorf("--body is required for reply")
		}
		return replyToComment(ctx, jira, client.Hostname(), opts)
	}

	// Handle add
	if opts.Body == "" {
		return fmt.Errorf("one of --body, --list, --edit, or --delete must be specified")
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
		fmt.Fprintf(opts.IO.Out, "**%s** (%s) [ID: %s]\n\n", c.Author, c.Created, c.ID)
		fmt.Fprintln(opts.IO.Out, c.Body)
		fmt.Fprintln(opts.IO.Out)
	}

	return nil
}

func addComment(ctx context.Context, jira *api.JiraService, hostname string, opts *CommentOptions) error {
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

func editComment(ctx context.Context, jira *api.JiraService, hostname string, opts *CommentOptions) error {
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

func deleteComment(ctx context.Context, jira *api.JiraService, hostname string, opts *CommentOptions) error {
	err := jira.DeleteComment(ctx, opts.IssueKey, opts.CommentID)
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

func replyToComment(ctx context.Context, jira *api.JiraService, hostname string, opts *CommentOptions) error {
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

	// Create quoted reply
	quotedLines := strings.Split(originalText, "\n")
	var quoted strings.Builder
	quoted.WriteString(fmt.Sprintf("*Replying to %s:*\n", originalAuthor))
	quoted.WriteString("{quote}\n")
	for _, line := range quotedLines {
		quoted.WriteString(line)
		quoted.WriteString("\n")
	}
	quoted.WriteString("{quote}\n\n")
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
