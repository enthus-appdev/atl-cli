package issue

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// AssignOptions holds the options for the assign command.
type AssignOptions struct {
	IO       *iostreams.IOStreams
	IssueKey string
	Assignee string
	JSON     bool
}

// NewCmdAssign creates the assign command.
func NewCmdAssign(ios *iostreams.IOStreams) *cobra.Command {
	opts := &AssignOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "assign <issue-key>",
		Short: "Assign an issue to a user",
		Long:  `Assign a Jira issue to a user or unassign it.`,
		Example: `  # Assign to yourself
  atl issue assign PROJ-1234 --assignee @me

  # Assign to another user
  atl issue assign PROJ-1234 --assignee john.doe

  # Unassign
  atl issue assign PROJ-1234 --assignee -

  # Output as JSON
  atl issue assign PROJ-1234 --assignee @me --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]
			if opts.Assignee == "" {
				return fmt.Errorf("--assignee flag is required\n\nUse @me to assign to yourself, or - to unassign")
			}
			return runAssign(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "User to assign (use @me for yourself, - to unassign)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// AssignOutput represents the result of assigning an issue.
type AssignOutput struct {
	IssueKey string `json:"issue_key"`
	Assignee string `json:"assignee"`
	URL      string `json:"url"`
}

func runAssign(opts *AssignOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	var accountID string
	var assigneeName string

	switch opts.Assignee {
	case "-", "none", "":
		accountID = "" // Unassign
		assigneeName = "Unassigned"
	case "@me":
		user, err := jira.GetMyself(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}
		accountID = user.AccountID
		assigneeName = user.DisplayName
	default:
		users, err := jira.SearchUsers(ctx, opts.Assignee)
		if err != nil {
			return fmt.Errorf("failed to search for user: %w", err)
		}
		if len(users) == 0 {
			return fmt.Errorf("user not found: %s", opts.Assignee)
		}
		accountID = users[0].AccountID
		assigneeName = users[0].DisplayName
	}

	if err := jira.AssignIssue(ctx, opts.IssueKey, accountID); err != nil {
		return fmt.Errorf("failed to assign issue: %w", err)
	}

	assignOutput := &AssignOutput{
		IssueKey: opts.IssueKey,
		Assignee: assigneeName,
		URL:      fmt.Sprintf("https://%s/browse/%s", client.Hostname(), opts.IssueKey),
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, assignOutput)
	}

	if assigneeName == "Unassigned" {
		fmt.Fprintf(opts.IO.Out, "Unassigned %s\n", opts.IssueKey)
	} else {
		fmt.Fprintf(opts.IO.Out, "Assigned %s to %s\n", opts.IssueKey, assigneeName)
	}
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", assignOutput.URL)

	return nil
}
