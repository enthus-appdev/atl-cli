package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// EditOptions holds the options for the edit command.
type EditOptions struct {
	IO     *iostreams.IOStreams
	PageID string
	Title  string
	Body   string
	Append bool
	JSON   bool
}

// NewCmdEdit creates the edit command.
func NewCmdEdit(ios *iostreams.IOStreams) *cobra.Command {
	opts := &EditOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "edit <page-id>",
		Short: "Edit a Confluence page",
		Long: `Edit the content of an existing Confluence page.

By default, --body replaces the entire page content.
Use --append to add content to the end of the existing page instead.`,
		Example: `  # Edit page title
  atl confluence page edit 123456 --title "Updated Title"

  # Replace page content
  atl confluence page edit 123456 --body "<p>New content here</p>"

  # Append to existing content
  atl confluence page edit 123456 --body "<p>Additional content</p>" --append

  # Edit both title and content
  atl confluence page edit 123456 --title "New Title" --body "<p>New content</p>"

  # Output as JSON
  atl confluence page edit 123456 --title "New Title" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageID = args[0]
			return runEdit(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "New page title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New page body content")
	cmd.Flags().BoolVarP(&opts.Append, "append", "a", false, "Append to existing content instead of replacing")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// PageEditOutput represents the output after editing a page.
type PageEditOutput struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Version int    `json:"version"`
	URL     string `json:"url"`
}

func runEdit(opts *EditOptions) error {
	if opts.Title == "" && opts.Body == "" {
		return fmt.Errorf("either --title or --body must be specified")
	}

	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	// Get current page to get version and current values
	currentPage, err := confluence.GetPage(ctx, opts.PageID)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	title := opts.Title
	if title == "" {
		title = currentPage.Title
	}

	// Get existing body content
	existingBody := ""
	if currentPage.Body != nil && currentPage.Body.Storage != nil {
		existingBody = currentPage.Body.Storage.Value
	}

	var body string
	if opts.Body != "" {
		if opts.Append {
			// Append new content to existing body
			body = existingBody + opts.Body
		} else {
			// Replace with new content
			body = opts.Body
		}
	} else {
		body = existingBody
	}

	currentVersion := 1
	if currentPage.Version != nil {
		currentVersion = currentPage.Version.Number
	}

	page, err := confluence.UpdatePage(ctx, opts.PageID, title, body, currentVersion, "Updated via atl CLI")
	if err != nil {
		return fmt.Errorf("failed to update page: %w", err)
	}

	url := fmt.Sprintf("https://%s/wiki/pages/viewpage.action?pageId=%s", client.Hostname(), page.ID)
	if page.Links != nil && page.Links.WebUI != "" {
		url = fmt.Sprintf("https://%s/wiki%s", client.Hostname(), page.Links.WebUI)
	}

	newVersion := currentVersion + 1
	if page.Version != nil {
		newVersion = page.Version.Number
	}

	editOutput := &PageEditOutput{
		ID:      page.ID,
		Title:   page.Title,
		Version: newVersion,
		URL:     url,
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, editOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Updated page: %s\n", editOutput.Title)
	fmt.Fprintf(opts.IO.Out, "ID: %s\n", editOutput.ID)
	fmt.Fprintf(opts.IO.Out, "Version: %d\n", editOutput.Version)
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", editOutput.URL)

	return nil
}
