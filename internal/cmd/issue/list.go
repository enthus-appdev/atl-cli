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

// ListOptions holds the options for the list command.
type ListOptions struct {
	IO        *iostreams.IOStreams
	JQL       string
	Project   string
	Assignee  string
	Status    string
	Type      string
	Limit     int
	All       bool
	JSON      bool
	NextToken string // For cursor-based pagination
}

// NewCmdList creates the list command.
func NewCmdList(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		IO:    ios,
		Limit: 50,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Jira issues",
		Long: `List and search for Jira issues using JQL or filters.

By default, lists issues assigned to you. Use --project, --assignee, or --jql
to specify different search criteria.`,
		Example: `  # List your issues (default)
  atl issue list

  # List issues in a project
  atl issue list --project PROJ

  # List issues with custom JQL
  atl issue list --jql "project = PROJ AND status = 'In Progress'"

  # List open issues assigned to you
  atl issue list --assignee @me --status Open

  # Get next page using token from previous result
  atl issue list --project PROJ --next-token "TOKEN_FROM_PREVIOUS_RESULT"

  # Fetch all matching issues (may be slow for large result sets)
  atl issue list --project PROJ --all

  # Output as JSON for LLM processing
  atl issue list --project PROJ --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.JQL, "jql", "q", "", "JQL query to filter issues")
	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "Filter by project key")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "Filter by assignee (use @me for yourself)")
	cmd.Flags().StringVarP(&opts.Status, "status", "s", "", "Filter by status")
	cmd.Flags().StringVarP(&opts.Type, "type", "t", "", "Filter by issue type (e.g., Bug, Story, Task)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 50, "Maximum number of issues per page")
	cmd.Flags().StringVar(&opts.NextToken, "next-token", "", "Pagination token for fetching next page")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Fetch all matching issues (ignores --limit)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// IssueListOutput represents the output for issue list.
type IssueListOutput struct {
	Issues        []*IssueListItem `json:"issues"`
	Total         int              `json:"total"`
	Count         int              `json:"count"`
	HasMore       bool             `json:"has_more"`
	NextPageToken string           `json:"next_page_token,omitempty"`
	JQL           string           `json:"jql"`
}

// IssueListItem represents a single issue in the list.
type IssueListItem struct {
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Priority string `json:"priority,omitempty"`
	Type     string `json:"type"`
	Assignee string `json:"assignee,omitempty"`
	Created  string `json:"created"`
	Updated  string `json:"updated"`
}

func runList(opts *ListOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Build JQL query
	jql := buildJQL(opts)

	var allIssues []*api.Issue
	var total int
	var nextPageToken string
	var isLast bool

	if opts.All {
		// Fetch all pages using cursor-based pagination
		pageSize := 100 // Use larger page size for --all
		var token string
		for {
			searchOpts := api.SearchOptions{
				JQL:           jql,
				MaxResults:    pageSize,
				NextPageToken: token,
			}
			result, err := jira.Search(ctx, searchOpts)
			if err != nil {
				return fmt.Errorf("failed to search issues: %w", err)
			}
			if result.Total > 0 {
				total = result.Total
			}
			allIssues = append(allIssues, result.Issues...)

			if result.IsLast || result.NextPageToken == "" || len(result.Issues) == 0 {
				break
			}
			token = result.NextPageToken

			// Progress indicator for large fetches
			if !opts.JSON {
				fmt.Fprintf(opts.IO.Out, "\rFetching issues... %d", len(allIssues))
			}
		}
		if !opts.JSON && len(allIssues) > 100 {
			fmt.Fprintln(opts.IO.Out, "") // Clear progress line
		}
		isLast = true
	} else {
		// Single page fetch
		searchOpts := api.SearchOptions{
			JQL:           jql,
			MaxResults:    opts.Limit,
			NextPageToken: opts.NextToken,
		}
		result, err := jira.Search(ctx, searchOpts)
		if err != nil {
			return fmt.Errorf("failed to search issues: %w", err)
		}
		if result.Total > 0 {
			total = result.Total
		}
		allIssues = result.Issues
		nextPageToken = result.NextPageToken
		isLast = result.IsLast
	}

	hasMore := !isLast && nextPageToken != ""

	listOutput := &IssueListOutput{
		Issues:        make([]*IssueListItem, 0, len(allIssues)),
		Total:         total,
		Count:         len(allIssues),
		HasMore:       hasMore,
		NextPageToken: nextPageToken,
		JQL:           jql,
	}

	for _, issue := range allIssues {
		item := &IssueListItem{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Created: formatTime(issue.Fields.Created),
			Updated: formatTime(issue.Fields.Updated),
		}

		if issue.Fields.Status != nil {
			item.Status = issue.Fields.Status.Name
		}
		if issue.Fields.Priority != nil {
			item.Priority = issue.Fields.Priority.Name
		}
		if issue.Fields.IssueType != nil {
			item.Type = issue.Fields.IssueType.Name
		}
		if issue.Fields.Assignee != nil {
			item.Assignee = issue.Fields.Assignee.DisplayName
		}

		listOutput.Issues = append(listOutput.Issues, item)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, listOutput)
	}

	// Plain text output (LLM-friendly tabular format)
	if len(listOutput.Issues) == 0 {
		fmt.Fprintln(opts.IO.Out, "No issues found.")
		return nil
	}

	// Header with pagination info
	if opts.All {
		fmt.Fprintf(opts.IO.Out, "Found %d issues\n\n", len(allIssues))
	} else if total > 0 {
		fmt.Fprintf(opts.IO.Out, "Showing %d of %d issues\n\n", len(allIssues), total)
	} else {
		fmt.Fprintf(opts.IO.Out, "Showing %d issues\n\n", len(allIssues))
	}

	// Table header
	headers := []string{"KEY", "TYPE", "STATUS", "PRIORITY", "ASSIGNEE", "SUMMARY"}
	rows := make([][]string, 0, len(listOutput.Issues))

	for _, issue := range listOutput.Issues {
		assignee := issue.Assignee
		if assignee == "" {
			assignee = "-"
		}
		priority := issue.Priority
		if priority == "" {
			priority = "-"
		}
		// Truncate summary for table display
		summary := issue.Summary
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		rows = append(rows, []string{
			issue.Key,
			issue.Type,
			issue.Status,
			priority,
			assignee,
			summary,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)

	// Show pagination hint
	if hasMore {
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "More results available. Use --all to fetch everything, or use --json to get the next_page_token for pagination.")
	}

	return nil
}

func buildJQL(opts *ListOptions) string {
	if opts.JQL != "" {
		return opts.JQL
	}

	var clauses []string

	if opts.Project != "" {
		clauses = append(clauses, fmt.Sprintf("project = %q", opts.Project))
	}

	if opts.Assignee != "" {
		if opts.Assignee == "@me" {
			clauses = append(clauses, "assignee = currentUser()")
		} else {
			clauses = append(clauses, fmt.Sprintf("assignee = %q", opts.Assignee))
		}
	}

	if opts.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = %q", opts.Status))
	}

	if opts.Type != "" {
		clauses = append(clauses, fmt.Sprintf("issuetype = %q", opts.Type))
	}

	// The new /search/jql API requires bounded queries.
	// Default to current user's issues if no filter is specified.
	if len(clauses) == 0 {
		clauses = append(clauses, "assignee = currentUser()")
	}

	return strings.Join(clauses, " AND ") + " ORDER BY updated DESC"
}
