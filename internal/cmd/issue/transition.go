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

// TransitionOptions holds the options for the transition command.
type TransitionOptions struct {
	IO       *iostreams.IOStreams
	IssueKey string
	Status   string
	Comment  string
	List     bool
	JSON     bool
}

// NewCmdTransition creates the transition command.
func NewCmdTransition(ios *iostreams.IOStreams) *cobra.Command {
	opts := &TransitionOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:     "transition <issue-key> [status]",
		Aliases: []string{"move", "tr"},
		Short:   "Transition an issue to a new status",
		Long:    `Move a Jira issue to a different status in its workflow.`,
		Example: `  # List available transitions
  atl issue transition PROJ-1234 --list

  # Move issue to In Progress
  atl issue transition PROJ-1234 "In Progress"

  # Move issue to Done with a comment
  atl issue transition PROJ-1234 Done --comment "Completed the implementation"

  # Output result as JSON
  atl issue transition PROJ-1234 Done --json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]
			if len(args) > 1 {
				opts.Status = args[1]
			}
			return runTransition(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Comment, "comment", "c", "", "Add a comment with the transition")
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List available transitions")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// TransitionListOutput represents available transitions.
type TransitionListOutput struct {
	IssueKey    string             `json:"issue_key"`
	Transitions []*TransitionItem  `json:"transitions"`
}

// TransitionItem represents a single transition.
type TransitionItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ToStatus string `json:"to_status"`
}

// TransitionOutput represents the result of a transition.
type TransitionOutput struct {
	IssueKey   string `json:"issue_key"`
	FromStatus string `json:"from_status,omitempty"`
	ToStatus   string `json:"to_status"`
	URL        string `json:"url"`
}

func runTransition(opts *TransitionOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Get available transitions
	transitions, err := jira.GetTransitions(ctx, opts.IssueKey)
	if err != nil {
		return fmt.Errorf("failed to get transitions: %w", err)
	}

	if opts.List || opts.Status == "" {
		// List available transitions
		listOutput := &TransitionListOutput{
			IssueKey:    opts.IssueKey,
			Transitions: make([]*TransitionItem, 0, len(transitions)),
		}

		for _, t := range transitions {
			item := &TransitionItem{
				ID:   t.ID,
				Name: t.Name,
			}
			if t.To != nil {
				item.ToStatus = t.To.Name
			}
			listOutput.Transitions = append(listOutput.Transitions, item)
		}

		if opts.JSON {
			return output.JSON(opts.IO.Out, listOutput)
		}

		if len(listOutput.Transitions) == 0 {
			fmt.Fprintf(opts.IO.Out, "No transitions available for %s\n", opts.IssueKey)
			return nil
		}

		fmt.Fprintf(opts.IO.Out, "Available transitions for %s:\n\n", opts.IssueKey)
		for _, t := range listOutput.Transitions {
			fmt.Fprintf(opts.IO.Out, "  - %s (-> %s)\n", t.Name, t.ToStatus)
		}
		return nil
	}

	// Find matching transition
	var matchedTransition *api.Transition
	statusLower := strings.ToLower(opts.Status)

	for _, t := range transitions {
		if strings.ToLower(t.Name) == statusLower {
			matchedTransition = t
			break
		}
		// Also match on target status name
		if t.To != nil && strings.ToLower(t.To.Name) == statusLower {
			matchedTransition = t
			break
		}
	}

	if matchedTransition == nil {
		var available []string
		for _, t := range transitions {
			available = append(available, t.Name)
		}
		return fmt.Errorf("transition %q not found. Available transitions: %s", opts.Status, strings.Join(available, ", "))
	}

	// Get current status for output
	issue, err := jira.GetIssue(ctx, opts.IssueKey)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	fromStatus := ""
	if issue.Fields.Status != nil {
		fromStatus = issue.Fields.Status.Name
	}

	// Perform transition
	if err := jira.TransitionIssue(ctx, opts.IssueKey, matchedTransition.ID); err != nil {
		return fmt.Errorf("failed to transition issue: %w", err)
	}

	// Add comment if provided
	if opts.Comment != "" {
		if _, err := jira.AddComment(ctx, opts.IssueKey, opts.Comment); err != nil {
			fmt.Fprintf(opts.IO.ErrOut, "Warning: transition successful but failed to add comment: %v\n", err)
		}
	}

	toStatus := matchedTransition.Name
	if matchedTransition.To != nil {
		toStatus = matchedTransition.To.Name
	}

	transitionOutput := &TransitionOutput{
		IssueKey:   opts.IssueKey,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
		URL:        fmt.Sprintf("https://%s/browse/%s", client.Hostname(), opts.IssueKey),
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, transitionOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Transitioned %s: %s -> %s\n", opts.IssueKey, fromStatus, toStatus)
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", transitionOutput.URL)

	return nil
}
