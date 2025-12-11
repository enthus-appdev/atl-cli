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
	IO       *iostreams.IOStreams
	JQL      string
	Project  string
	Assignee string
	Status   string
	Type     string
	Limit    int
	Page     int
	All      bool
	JSON     bool
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
		Long:    `List and search for Jira issues using JQL or filters.`,
		Example: `  # List issues assigned to you
  atl issue list --assignee @me

  # List issues in a project
  atl issue list --project PROJ

  # List issues with custom JQL
  atl issue list --jql "project = PROJ AND status = 'In Progress'"

  # List open issues assigned to you
  atl issue list --assignee @me --status Open

  # Pagination: get page 2 of results
  atl issue list --project PROJ --page 2

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
	cmd.Flags().IntVar(&opts.Page, "page", 1, "Page number (1-based)")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Fetch all matching issues (ignores --limit and --page)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// IssueListOutput represents the output for issue list.
type IssueListOutput struct {
	Issues     []*IssueListItem `json:"issues"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	TotalPages int              `json:"total_pages"`
	PerPage    int              `json:"per_page"`
	HasMore    bool             `json:"has_more"`
	JQL        string           `json:"jql"`
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

	if opts.All {
		// Fetch all pages
		startAt := 0
		pageSize := 100 // Use larger page size for --all
		for {
			searchOpts := api.SearchOptions{
				JQL:        jql,
				StartAt:    startAt,
				MaxResults: pageSize,
			}
			result, err := jira.Search(ctx, searchOpts)
			if err != nil {
				return fmt.Errorf("failed to search issues: %w", err)
			}
			total = result.Total
			allIssues = append(allIssues, result.Issues...)

			if len(result.Issues) < pageSize || len(allIssues) >= total {
				break
			}
			startAt += len(result.Issues)

			// Progress indicator for large fetches
			if !opts.JSON {
				fmt.Fprintf(opts.IO.Out, "\rFetching issues... %d/%d", len(allIssues), total)
			}
		}
		if !opts.JSON && total > 100 {
			fmt.Fprintln(opts.IO.Out, "") // Clear progress line
		}
	} else {
		// Single page fetch
		startAt := (opts.Page - 1) * opts.Limit
		searchOpts := api.SearchOptions{
			JQL:        jql,
			StartAt:    startAt,
			MaxResults: opts.Limit,
		}
		result, err := jira.Search(ctx, searchOpts)
		if err != nil {
			return fmt.Errorf("failed to search issues: %w", err)
		}
		total = result.Total
		allIssues = result.Issues
	}

	// Calculate pagination info
	perPage := opts.Limit
	if opts.All {
		perPage = len(allIssues)
	}
	totalPages := (total + opts.Limit - 1) / opts.Limit
	if totalPages == 0 {
		totalPages = 1
	}
	hasMore := opts.Page < totalPages && !opts.All

	listOutput := &IssueListOutput{
		Issues:     make([]*IssueListItem, 0, len(allIssues)),
		Total:      total,
		Page:       opts.Page,
		TotalPages: totalPages,
		PerPage:    perPage,
		HasMore:    hasMore,
		JQL:        jql,
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
		fmt.Fprintf(opts.IO.Out, "Found %d issues\n\n", total)
	} else {
		fmt.Fprintf(opts.IO.Out, "Page %d of %d (%d issues total)\n\n", opts.Page, totalPages, total)
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
	if hasMore && !opts.All {
		fmt.Fprintf(opts.IO.Out, "\nUse --page %d to see more, or --all to fetch everything\n", opts.Page+1)
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

	if len(clauses) == 0 {
		return "ORDER BY updated DESC"
	}

	return strings.Join(clauses, " AND ") + " ORDER BY updated DESC"
}
