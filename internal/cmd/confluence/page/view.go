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
	Raw    bool
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
  atl confluence page view 123456 --json

  # Output raw storage format (XHTML with macros)
  atl confluence page view 123456 --raw`,
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
	cmd.Flags().BoolVarP(&opts.Raw, "raw", "r", false, "Output raw storage format (XHTML with macros)")

	return cmd
}

// PageViewOutput represents the output for page view.
type PageViewOutput struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	SpaceID    string `json:"space_id"`
	Status     string `json:"status"`
	Version    int    `json:"version"`
	Body       string `json:"body"`
	BodyFormat string `json:"body_format,omitempty"`
	URL        string `json:"url"`
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
		// Search by title
		result, err := confluence.SearchPages(ctx, opts.Title, 10)
		if err != nil {
			return fmt.Errorf("failed to search pages: %w", err)
		}
		if len(result.Results) == 0 {
			return fmt.Errorf("page not found: %s", opts.Title)
		}
		// Find first matching page (optionally in the specified space)
		for _, p := range result.Results {
			if opts.Space == "" || p.SpaceID == opts.Space {
				page = p
				break
			}
		}
		if page == nil {
			return fmt.Errorf("page not found: %s in space %s", opts.Title, opts.Space)
		}
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

	// Extract body content - try storage first, then atlas_doc_format
	if page.Body != nil {
		if page.Body.Storage != nil && page.Body.Storage.Value != "" {
			viewOutput.BodyFormat = "storage"
			if opts.Raw {
				viewOutput.Body = page.Body.Storage.Value
			} else {
				viewOutput.Body = storageToPlainText(page.Body.Storage.Value)
			}
		} else if page.Body.AtlasDocFormat != nil && page.Body.AtlasDocFormat.Value != "" {
			viewOutput.BodyFormat = "atlas_doc_format"
			if opts.Raw {
				viewOutput.Body = page.Body.AtlasDocFormat.Value
			} else {
				viewOutput.Body = adfToPlainText(page.Body.AtlasDocFormat.Value)
			}
		}
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
// Extracts text content from macros instead of removing them.
func storageToPlainText(storage string) string {
	text := storage

	// Extract text from CDATA sections in macros (code blocks, etc.)
	// <ac:plain-text-body><![CDATA[content]]></ac:plain-text-body>
	cdataRegex := regexp.MustCompile(`<!\[CDATA\[(.*?)\]\]>`)
	text = cdataRegex.ReplaceAllString(text, "$1\n")

	// Extract text from rich-text-body in macros
	// <ac:rich-text-body>content</ac:rich-text-body>
	richTextRegex := regexp.MustCompile(`<ac:rich-text-body>(.*?)</ac:rich-text-body>`)
	text = richTextRegex.ReplaceAllString(text, "$1\n")

	// Extract macro names for context (e.g., [Macro: jira] or [Macro: toc])
	macroNameRegex := regexp.MustCompile(`<ac:structured-macro[^>]*ac:name="([^"]*)"[^>]*>`)
	text = macroNameRegex.ReplaceAllString(text, "\n[Macro: $1]\n")

	// Remove remaining ac: tags but keep their content
	acTagRegex := regexp.MustCompile(`</?ac:[^>]*>`)
	text = acTagRegex.ReplaceAllString(text, "")

	// Remove ri: (resource identifier) tags
	riTagRegex := regexp.MustCompile(`</?ri:[^>]*>`)
	text = riTagRegex.ReplaceAllString(text, "")

	// Convert common HTML tags to text
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n\n")
	text = strings.ReplaceAll(text, "</li>", "\n")
	text = strings.ReplaceAll(text, "<li>", "• ")
	text = strings.ReplaceAll(text, "</h1>", "\n\n")
	text = strings.ReplaceAll(text, "</h2>", "\n\n")
	text = strings.ReplaceAll(text, "</h3>", "\n\n")
	text = strings.ReplaceAll(text, "</tr>", "\n")
	text = strings.ReplaceAll(text, "</td>", " | ")
	text = strings.ReplaceAll(text, "</th>", " | ")

	// Strip remaining HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	text = tagRegex.ReplaceAllString(text, "")

	// Decode HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")

	// Clean up whitespace
	text = strings.TrimSpace(text)
	spaceRegex := regexp.MustCompile(`\n{3,}`)
	text = spaceRegex.ReplaceAllString(text, "\n\n")
	// Clean up multiple spaces
	multiSpaceRegex := regexp.MustCompile(`[ \t]+`)
	text = multiSpaceRegex.ReplaceAllString(text, " ")

	return text
}

// adfToPlainText converts Atlassian Document Format (ADF) JSON to plain text.
// ADF is used by the new Confluence editor.
func adfToPlainText(adf string) string {
	// ADF is JSON - extract text nodes
	// Simple extraction: find all "text" fields
	textRegex := regexp.MustCompile(`"text"\s*:\s*"([^"]*)"`)
	matches := textRegex.FindAllStringSubmatch(adf, -1)

	var texts []string
	for _, match := range matches {
		if len(match) > 1 && match[1] != "" {
			// Unescape JSON strings
			text := strings.ReplaceAll(match[1], `\\n`, "\n")
			text = strings.ReplaceAll(text, `\n`, "\n")
			text = strings.ReplaceAll(text, `\"`, "\"")
			text = strings.ReplaceAll(text, `\\`, "\\")
			texts = append(texts, text)
		}
	}

	result := strings.Join(texts, " ")

	// Clean up whitespace
	result = strings.TrimSpace(result)
	spaceRegex := regexp.MustCompile(`\n{3,}`)
	result = spaceRegex.ReplaceAllString(result, "\n\n")

	return result
}
