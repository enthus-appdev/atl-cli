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

// PublishOptions holds the options for the publish command.
type PublishOptions struct {
	IO      *iostreams.IOStreams
	PageIDs []string
	Web     bool
	JSON    bool
}

// NewCmdPublish creates the publish command.
func NewCmdPublish(ios *iostreams.IOStreams) *cobra.Command {
	opts := &PublishOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "publish <page-id> [page-id...]",
		Short: "Publish draft Confluence pages",
		Long: `Publish one or more draft pages, making them visible to other users.

This command changes the page status from "draft" to "current".
Use 'atl confluence page list --status draft' to find draft pages.`,
		Example: `  # Publish a single draft page
  atl confluence page publish 123456

  # Publish multiple draft pages
  atl confluence page publish 123456 789012

  # Publish and open in browser
  atl confluence page publish 123456 --web

  # Output as JSON
  atl confluence page publish 123456 --json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageIDs = args
			return runPublish(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open published page(s) in browser")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// PublishOutput represents the output of the publish command.
type PublishOutput struct {
	Pages     []*PublishedPage `json:"pages"`
	Published int              `json:"published"`
	Failed    int              `json:"failed"`
	Success   bool             `json:"success"`
}

// PublishedPage represents a successfully published page.
type PublishedPage struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

func runPublish(opts *PublishOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	var publishedPages []*PublishedPage
	var failedIDs []string

	for _, pageID := range opts.PageIDs {
		page, err := confluence.PublishPage(ctx, pageID)
		if err != nil {
			failedIDs = append(failedIDs, pageID)
			if !opts.JSON {
				fmt.Fprintf(opts.IO.Out, "Failed to publish %s: %v\n", pageID, err)
			}
			continue
		}

		url := fmt.Sprintf("https://%s/wiki/pages/%s", client.Hostname(), page.ID)
		if page.Links != nil && page.Links.WebUI != "" {
			url = fmt.Sprintf("https://%s/wiki%s", client.Hostname(), page.Links.WebUI)
		}

		publishedPages = append(publishedPages, &PublishedPage{
			ID:    page.ID,
			Title: page.Title,
			URL:   url,
		})

		if opts.Web {
			auth.OpenBrowser(url)
		}
	}

	publishOutput := &PublishOutput{
		Pages:     publishedPages,
		Published: len(publishedPages),
		Failed:    len(failedIDs),
		Success:   len(failedIDs) == 0,
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, publishOutput)
	}

	for _, page := range publishedPages {
		fmt.Fprintf(opts.IO.Out, "Published: %s (%s)\n", page.Title, page.ID)
		fmt.Fprintf(opts.IO.Out, "URL: %s\n", page.URL)
	}

	if len(failedIDs) > 0 {
		return fmt.Errorf("failed to publish %d page(s)", len(failedIDs))
	}

	return nil
}
