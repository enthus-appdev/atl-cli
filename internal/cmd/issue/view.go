package issue

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// ViewOptions holds the options for the view command.
type ViewOptions struct {
	IO       *iostreams.IOStreams
	IssueKey string
	JSON     bool
	Web      bool
}

// NewCmdView creates the view command.
func NewCmdView(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ViewOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "view <issue-key>",
		Short: "View a Jira issue",
		Long:  `Display details of a Jira issue.`,
		Example: `  # View an issue
  atl issue view PROJ-1234

  # View an issue as JSON
  atl issue view PROJ-1234 --json

  # Open issue in browser
  atl issue view PROJ-1234 --web`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]
			return runView(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open in web browser")

	return cmd
}

// IssueOutput represents the output format for an issue (LLM-friendly).
type IssueOutput struct {
	Key         string        `json:"key"`
	ID          string        `json:"id"`
	Summary     string        `json:"summary"`
	Description string        `json:"description,omitempty"`
	Status      string        `json:"status"`
	StatusCategory string     `json:"status_category,omitempty"`
	Priority    string        `json:"priority,omitempty"`
	Type        string        `json:"type"`
	Assignee    *UserOutput   `json:"assignee,omitempty"`
	Reporter    *UserOutput   `json:"reporter,omitempty"`
	Project     *ProjectOutput `json:"project"`
	Labels      []string      `json:"labels,omitempty"`
	Created     string        `json:"created"`
	Updated     string        `json:"updated"`
	URL         string        `json:"url"`
}

// UserOutput represents user information.
type UserOutput struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email,omitempty"`
}

// ProjectOutput represents project information.
type ProjectOutput struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func runView(opts *ViewOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	if opts.Web {
		url := fmt.Sprintf("https://%s/browse/%s", client.Hostname(), opts.IssueKey)
		return auth.OpenBrowser(url)
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	issue, err := jira.GetIssue(ctx, opts.IssueKey)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	issueOutput := formatIssueOutput(issue, client.Hostname())

	if opts.JSON {
		return output.JSON(opts.IO.Out, issueOutput)
	}

	// Plain text output (LLM-friendly format)
	printIssueDetails(opts.IO, issueOutput)

	return nil
}

func formatIssueOutput(issue *api.Issue, hostname string) *IssueOutput {
	out := &IssueOutput{
		Key:     issue.Key,
		ID:      issue.ID,
		Summary: issue.Fields.Summary,
		URL:     fmt.Sprintf("https://%s/browse/%s", hostname, issue.Key),
	}

	if issue.Fields.Description != nil {
		out.Description = api.ADFToText(issue.Fields.Description)
	}

	if issue.Fields.Status != nil {
		out.Status = issue.Fields.Status.Name
		if issue.Fields.Status.StatusCategory != nil {
			out.StatusCategory = issue.Fields.Status.StatusCategory.Key
		}
	}

	if issue.Fields.Priority != nil {
		out.Priority = issue.Fields.Priority.Name
	}

	if issue.Fields.IssueType != nil {
		out.Type = issue.Fields.IssueType.Name
	}

	if issue.Fields.Assignee != nil {
		out.Assignee = &UserOutput{
			AccountID:   issue.Fields.Assignee.AccountID,
			DisplayName: issue.Fields.Assignee.DisplayName,
			Email:       issue.Fields.Assignee.EmailAddress,
		}
	}

	if issue.Fields.Reporter != nil {
		out.Reporter = &UserOutput{
			AccountID:   issue.Fields.Reporter.AccountID,
			DisplayName: issue.Fields.Reporter.DisplayName,
			Email:       issue.Fields.Reporter.EmailAddress,
		}
	}

	if issue.Fields.Project != nil {
		out.Project = &ProjectOutput{
			Key:  issue.Fields.Project.Key,
			Name: issue.Fields.Project.Name,
		}
	}

	out.Labels = issue.Fields.Labels
	out.Created = formatTime(issue.Fields.Created)
	out.Updated = formatTime(issue.Fields.Updated)

	return out
}

func printIssueDetails(ios *iostreams.IOStreams, issue *IssueOutput) {
	fmt.Fprintf(ios.Out, "# %s: %s\n\n", issue.Key, issue.Summary)

	fmt.Fprintf(ios.Out, "Type: %s\n", issue.Type)
	fmt.Fprintf(ios.Out, "Status: %s\n", issue.Status)
	if issue.Priority != "" {
		fmt.Fprintf(ios.Out, "Priority: %s\n", issue.Priority)
	}

	if issue.Project != nil {
		fmt.Fprintf(ios.Out, "Project: %s (%s)\n", issue.Project.Name, issue.Project.Key)
	}

	if issue.Assignee != nil {
		fmt.Fprintf(ios.Out, "Assignee: %s\n", issue.Assignee.DisplayName)
	} else {
		fmt.Fprintln(ios.Out, "Assignee: Unassigned")
	}

	if issue.Reporter != nil {
		fmt.Fprintf(ios.Out, "Reporter: %s\n", issue.Reporter.DisplayName)
	}

	if len(issue.Labels) > 0 {
		fmt.Fprintf(ios.Out, "Labels: %s\n", strings.Join(issue.Labels, ", "))
	}

	fmt.Fprintf(ios.Out, "Created: %s\n", issue.Created)
	fmt.Fprintf(ios.Out, "Updated: %s\n", issue.Updated)
	fmt.Fprintf(ios.Out, "URL: %s\n", issue.URL)

	if issue.Description != "" {
		fmt.Fprintln(ios.Out, "")
		fmt.Fprintln(ios.Out, "## Description")
		fmt.Fprintln(ios.Out, "")
		fmt.Fprintln(ios.Out, issue.Description)
	}
}

func formatTime(timeStr string) string {
	if timeStr == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02T15:04:05.000-0700", timeStr)
	if err != nil {
		// Try alternative format
		t, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return timeStr
		}
	}
	return t.Format("2006-01-02 15:04:05")
}
