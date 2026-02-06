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

// ChangelogOptions holds the options for the changelog command.
type ChangelogOptions struct {
	IO       *iostreams.IOStreams
	IssueKey string
	Field    string
	Limit    int
	JSON     bool
}

// ChangelogEntryOutput represents a single changelog entry for output.
type ChangelogEntryOutput struct {
	Created string                 `json:"created"`
	Author  string                 `json:"author"`
	Items   []*ChangelogItemOutput `json:"items"`
}

// ChangelogItemOutput represents a single field change for output.
type ChangelogItemOutput struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// NewCmdChangelog creates the changelog command.
func NewCmdChangelog(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ChangelogOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:     "changelog <issue-key>",
		Aliases: []string{"history"},
		Short:   "View the changelog of a Jira issue",
		Long:    `Display the history of field changes for a Jira issue.`,
		Example: `  # View full changelog
  atl issue changelog NX-1234

  # Filter by field name
  atl issue changelog NX-1234 --field status

  # Limit number of entries
  atl issue changelog NX-1234 --limit 5

  # Output as JSON
  atl issue changelog NX-1234 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]
			return runChangelog(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	cmd.Flags().StringVarP(&opts.Field, "field", "f", "", "Filter by field name")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 0, "Maximum number of entries to show")

	return cmd
}

func runChangelog(opts *ChangelogOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Fetch all changelog pages
	var allEntries []*api.ChangelogEntry
	startAt := 0
	for {
		resp, err := jira.GetChangelog(ctx, opts.IssueKey, startAt)
		if err != nil {
			return fmt.Errorf("failed to get changelog: %w", err)
		}

		allEntries = append(allEntries, resp.Values...)

		if resp.IsLast || len(resp.Values) == 0 {
			break
		}
		startAt += len(resp.Values)
	}

	// Convert to output format
	entries := make([]*ChangelogEntryOutput, 0, len(allEntries))
	for _, e := range allEntries {
		entries = append(entries, formatChangelogEntryOutput(e))
	}

	// Apply field filter
	if opts.Field != "" {
		entries = filterChangelogByField(entries, opts.Field)
	}

	// Apply limit
	if opts.Limit > 0 && len(entries) > opts.Limit {
		entries = entries[len(entries)-opts.Limit:]
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, entries)
	}

	printChangelog(opts.IO, opts.IssueKey, entries)
	return nil
}

func formatChangelogEntryOutput(entry *api.ChangelogEntry) *ChangelogEntryOutput {
	out := &ChangelogEntryOutput{
		Created: entry.Created,
	}

	if entry.Author != nil {
		out.Author = entry.Author.DisplayName
	}

	out.Items = make([]*ChangelogItemOutput, 0, len(entry.Items))
	for _, item := range entry.Items {
		out.Items = append(out.Items, &ChangelogItemOutput{
			Field: item.Field,
			From:  item.FromString,
			To:    item.ToString,
		})
	}

	return out
}

func filterChangelogByField(entries []*ChangelogEntryOutput, field string) []*ChangelogEntryOutput {
	fieldLower := strings.ToLower(field)
	var filtered []*ChangelogEntryOutput

	for _, entry := range entries {
		var matchingItems []*ChangelogItemOutput
		for _, item := range entry.Items {
			if strings.ToLower(item.Field) == fieldLower {
				matchingItems = append(matchingItems, item)
			}
		}
		if len(matchingItems) > 0 {
			filtered = append(filtered, &ChangelogEntryOutput{
				Created: entry.Created,
				Author:  entry.Author,
				Items:   matchingItems,
			})
		}
	}

	return filtered
}

func printChangelog(ios *iostreams.IOStreams, issueKey string, entries []*ChangelogEntryOutput) {
	if len(entries) == 0 {
		fmt.Fprintf(ios.Out, "No changelog entries found for %s\n", issueKey)
		return
	}

	for i, entry := range entries {
		if i > 0 {
			fmt.Fprintln(ios.Out)
		}
		fmt.Fprintf(ios.Out, "%s  %s\n", formatTime(entry.Created), entry.Author)
		for _, item := range entry.Items {
			fmt.Fprintf(ios.Out, "  %s: %q → %q\n", item.Field, item.From, item.To)
		}
	}
}
