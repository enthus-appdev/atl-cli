package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// CreateOptions holds the options for the create command.
type CreateOptions struct {
	IO       *iostreams.IOStreams
	Space    string
	Title    string
	ParentID string
	Body     string
	Draft    bool
	Web      bool
	JSON     bool
}

// NewCmdCreate creates the create command.
func NewCmdCreate(ios *iostreams.IOStreams) *cobra.Command {
	opts := &CreateOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Confluence page",
		Long: `Create a new page in a Confluence space.

Use --draft to create a draft page that is not yet published.
Draft pages can later be published using 'atl confluence page publish'.`,
		Example: `  # Create a page
  atl confluence page create --space DOCS --title "New Page"

  # Create a page with content
  atl confluence page create --space DOCS --title "New Page" --body "Page content here"

  # Create a draft page (not published)
  atl confluence page create --space DOCS --title "Draft Page" --draft

  # Create a child page
  atl confluence page create --space DOCS --title "Child Page" --parent 123456

  # Create and open in browser
  atl confluence page create --space DOCS --title "New Page" --web

  # Output as JSON
  atl confluence page create --space DOCS --title "New Page" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var missing []string
			if opts.Space == "" {
				missing = append(missing, "--space")
			}
			if opts.Title == "" {
				missing = append(missing, "--title")
			}
			if len(missing) > 0 {
				return fmt.Errorf("required flags not set: %v\n\nExample: atl confluence page create --space DOCS --title \"Page Title\"\n\nUse 'atl confluence space list' to see available spaces", missing)
			}
			return runCreate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Space, "space", "s", "", "Space key (required)")
	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Page title (required)")
	cmd.Flags().StringVarP(&opts.ParentID, "parent", "p", "", "Parent page ID")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Page body content")
	cmd.Flags().BoolVarP(&opts.Draft, "draft", "d", false, "Create as draft (not published)")
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open created page in browser")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// PageCreateOutput represents the output after creating a page.
type PageCreateOutput struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	SpaceID string `json:"space_id"`
	Status  string `json:"status"`
	URL     string `json:"url"`
}

func runCreate(opts *CreateOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	// Get space ID from key
	space, err := confluence.GetSpaceByKey(ctx, opts.Space)
	if err != nil {
		return fmt.Errorf("failed to get space: %w", err)
	}

	body := opts.Body
	if body == "" {
		body = "<p></p>" // Empty paragraph
	} else {
		// Wrap plain text in paragraph tags
		body = "<p>" + body + "</p>"
	}

	status := ""
	if opts.Draft {
		status = "draft"
	}

	page, err := confluence.CreatePage(ctx, space.ID, opts.Title, body, opts.ParentID, status)
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	url := fmt.Sprintf("https://%s/wiki/spaces/%s/pages/%s", client.Hostname(), opts.Space, page.ID)
	if page.Links != nil && page.Links.WebUI != "" {
		url = fmt.Sprintf("https://%s/wiki%s", client.Hostname(), page.Links.WebUI)
	}

	if opts.Web {
		auth.OpenBrowser(url)
	}

	createOutput := &PageCreateOutput{
		ID:      page.ID,
		Title:   page.Title,
		SpaceID: page.SpaceID,
		Status:  page.Status,
		URL:     url,
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, createOutput)
	}

	if page.Status == "draft" {
		fmt.Fprintf(opts.IO.Out, "Created draft page: %s\n", createOutput.Title)
	} else {
		fmt.Fprintf(opts.IO.Out, "Created page: %s\n", createOutput.Title)
	}
	fmt.Fprintf(opts.IO.Out, "ID: %s\n", createOutput.ID)
	fmt.Fprintf(opts.IO.Out, "Status: %s\n", createOutput.Status)
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", createOutput.URL)

	return nil
}
