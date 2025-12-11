package page

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// ViewOptions holds the options for the view command.
type ViewOptions struct {
	IO     *iostreams.IOStreams
	PageID string
	Space  string
	Title  string
	JSON   bool
	Web    bool
}

// NewCmdView creates the view command.
func NewCmdView(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ViewOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "view [page-id]",
		Short: "View a Confluence page",
		Long:  `Display the content of a Confluence page.`,
		Example: `  # View a page by ID
  atl confluence page view 123456

  # View a page by space and title
  atl confluence page view --space DOCS --title "Getting Started"

  # Open page in browser
  atl confluence page view 123456 --web

  # Output as JSON
  atl confluence page view 123456 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.PageID = args[0]
			}
			return runView(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Space, "space", "s", "", "Space key")
	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Page title")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open in browser")

	return cmd
}

// PageViewOutput represents the output for page view.
type PageViewOutput struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	SpaceID string `json:"space_id"`
	Status  string `json:"status"`
	Version int    `json:"version"`
	Body    string `json:"body"`
	URL     string `json:"url"`
}

func runView(opts *ViewOptions) error {
	if opts.PageID == "" && (opts.Space == "" || opts.Title == "") {
		return fmt.Errorf("please provide a page ID or both --space and --title")
	}

	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	var page *api.Page

	if opts.PageID != "" {
		page, err = confluence.GetPage(ctx, opts.PageID)
		if err != nil {
			return fmt.Errorf("failed to get page: %w", err)
		}
	} else {
		// Search by space and title
		result, err := confluence.SearchPages(ctx, opts.Space, opts.Title, 1)
		if err != nil {
			return fmt.Errorf("failed to search pages: %w", err)
		}
		if len(result.Results) == 0 {
			return fmt.Errorf("page not found: %s / %s", opts.Space, opts.Title)
		}
		page = result.Results[0]
		// Get full page content
		page, err = confluence.GetPage(ctx, page.ID)
		if err != nil {
			return fmt.Errorf("failed to get page: %w", err)
		}
	}

	url := fmt.Sprintf("https://%s/wiki/spaces/%s/pages/%s", client.Hostname(), opts.Space, page.ID)
	if page.Links != nil && page.Links.WebUI != "" {
		url = fmt.Sprintf("https://%s/wiki%s", client.Hostname(), page.Links.WebUI)
	}

	if opts.Web {
		return auth.OpenBrowser(url)
	}

	viewOutput := &PageViewOutput{
		ID:      page.ID,
		Title:   page.Title,
		SpaceID: page.SpaceID,
		Status:  page.Status,
		URL:     url,
	}

	if page.Version != nil {
		viewOutput.Version = page.Version.Number
	}

	if page.Body != nil && page.Body.Storage != nil {
		viewOutput.Body = storageToPlainText(page.Body.Storage.Value)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, viewOutput)
	}

	// Plain text output (LLM-friendly)
	fmt.Fprintf(opts.IO.Out, "# %s\n\n", viewOutput.Title)
	fmt.Fprintf(opts.IO.Out, "ID: %s\n", viewOutput.ID)
	fmt.Fprintf(opts.IO.Out, "Status: %s\n", viewOutput.Status)
	fmt.Fprintf(opts.IO.Out, "Version: %d\n", viewOutput.Version)
	fmt.Fprintf(opts.IO.Out, "URL: %s\n", viewOutput.URL)

	if viewOutput.Body != "" {
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, "## Content")
		fmt.Fprintln(opts.IO.Out, "")
		fmt.Fprintln(opts.IO.Out, viewOutput.Body)
	}

	return nil
}

// storageToPlainText converts Confluence storage format to plain text.
func storageToPlainText(storage string) string {
	// Simple conversion - strip HTML tags for plain text
	// This is a basic implementation; a full HTML parser would be better

	// Remove ac: tags content (macros)
	acRegex := regexp.MustCompile(`<ac:[^>]*>.*?</ac:[^>]*>`)
	text := acRegex.ReplaceAllString(storage, "")

	// Convert common tags to text
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n\n")
	text = strings.ReplaceAll(text, "</li>", "\n")
	text = strings.ReplaceAll(text, "<li>", "• ")

	// Strip remaining HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	text = tagRegex.ReplaceAllString(text, "")

	// Clean up whitespace
	text = strings.TrimSpace(text)
	spaceRegex := regexp.MustCompile(`\n{3,}`)
	text = spaceRegex.ReplaceAllString(text, "\n\n")

	return text
}
