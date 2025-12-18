package issue

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// WebLinkOptions holds the options for the weblink command.
type WebLinkOptions struct {
	IO       *iostreams.IOStreams
	IssueKey string
	URL      string
	Title    string
	Summary  string
	List     bool
	Delete   int
	JSON     bool
}

// NewCmdWebLink creates the weblink command.
func NewCmdWebLink(ios *iostreams.IOStreams) *cobra.Command {
	opts := &WebLinkOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "weblink <issue-key>",
		Short: "Add, list, or remove web links on a Jira issue",
		Long: `Manage web links (remote links) on a Jira issue.

Web links connect issues to external URLs like documentation,
pull requests, or related resources.`,
		Example: `  # Add a web link
  atl issue weblink PROJ-123 --url "https://example.com/doc" --title "Documentation"

  # Add a web link with description
  atl issue weblink PROJ-123 --url "https://github.com/org/repo/pull/123" --title "PR #123" --summary "Fix for the bug"

  # List all web links on an issue
  atl issue weblink PROJ-123 --list

  # Delete a web link by ID
  atl issue weblink PROJ-123 --delete 12345

  # Output as JSON
  atl issue weblink PROJ-123 --list --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]

			// Validate flags
			if opts.List {
				return runWebLinkList(opts)
			}
			if opts.Delete > 0 {
				return runWebLinkDelete(opts)
			}
			if opts.URL == "" {
				return fmt.Errorf("--url is required to add a web link\n\nUse --list to view existing links or --delete to remove one")
			}
			if opts.Title == "" {
				return fmt.Errorf("--title is required to add a web link")
			}
			return runWebLinkAdd(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.URL, "url", "u", "", "URL to link to")
	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Link title (displayed text)")
	cmd.Flags().StringVarP(&opts.Summary, "summary", "s", "", "Link summary/description")
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List all web links on the issue")
	cmd.Flags().IntVarP(&opts.Delete, "delete", "d", 0, "Delete web link by ID")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// WebLinkOutput represents a web link in output.
type WebLinkOutput struct {
	ID      int    `json:"id"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Summary string `json:"summary,omitempty"`
}

// WebLinkListOutput represents the list output.
type WebLinkListOutput struct {
	IssueKey string           `json:"issue_key"`
	Links    []*WebLinkOutput `json:"links"`
	Total    int              `json:"total"`
}

// WebLinkAddOutput represents the add output.
type WebLinkAddOutput struct {
	IssueKey string `json:"issue_key"`
	LinkID   int    `json:"link_id"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Action   string `json:"action"`
}

func runWebLinkList(opts *WebLinkOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	links, err := jira.GetRemoteLinks(ctx, opts.IssueKey)
	if err != nil {
		return fmt.Errorf("failed to get web links: %w", err)
	}

	listOutput := &WebLinkListOutput{
		IssueKey: opts.IssueKey,
		Links:    make([]*WebLinkOutput, 0, len(links)),
		Total:    len(links),
	}

	for _, link := range links {
		if link.Object == nil {
			continue
		}
		listOutput.Links = append(listOutput.Links, &WebLinkOutput{
			ID:      link.ID,
			URL:     link.Object.URL,
			Title:   link.Object.Title,
			Summary: link.Object.Summary,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, listOutput)
	}

	if len(listOutput.Links) == 0 {
		fmt.Fprintf(opts.IO.Out, "No web links on %s\n", opts.IssueKey)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Web links on %s (%d total):\n\n", opts.IssueKey, listOutput.Total)

	for _, link := range listOutput.Links {
		fmt.Fprintf(opts.IO.Out, "[%d] %s\n", link.ID, link.Title)
		fmt.Fprintf(opts.IO.Out, "    %s\n", link.URL)
		if link.Summary != "" {
			fmt.Fprintf(opts.IO.Out, "    %s\n", link.Summary)
		}
		fmt.Fprintln(opts.IO.Out)
	}

	fmt.Fprintf(opts.IO.Out, "To delete a link: atl issue weblink %s --delete <id>\n", opts.IssueKey)

	return nil
}

func runWebLinkAdd(opts *WebLinkOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	link, err := jira.CreateRemoteLink(ctx, opts.IssueKey, opts.URL, opts.Title, opts.Summary)
	if err != nil {
		return fmt.Errorf("failed to add web link: %w", err)
	}

	addOutput := &WebLinkAddOutput{
		IssueKey: opts.IssueKey,
		LinkID:   link.ID,
		URL:      opts.URL,
		Title:    opts.Title,
		Action:   "added",
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, addOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Added web link to %s\n", opts.IssueKey)
	fmt.Fprintf(opts.IO.Out, "  Title: %s\n", opts.Title)
	fmt.Fprintf(opts.IO.Out, "  URL: %s\n", opts.URL)
	fmt.Fprintf(opts.IO.Out, "  Link ID: %d\n", link.ID)

	return nil
}

func runWebLinkDelete(opts *WebLinkOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	err = jira.DeleteRemoteLink(ctx, opts.IssueKey, opts.Delete)
	if err != nil {
		return fmt.Errorf("failed to delete web link: %w", err)
	}

	deleteOutput := &WebLinkAddOutput{
		IssueKey: opts.IssueKey,
		LinkID:   opts.Delete,
		Action:   "deleted",
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, deleteOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Deleted web link %d from %s\n", opts.Delete, opts.IssueKey)

	return nil
}

// Helper to parse link ID from string
func parseLinkID(s string) (int, error) {
	return strconv.Atoi(s)
}
