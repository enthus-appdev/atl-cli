package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// ArchiveOptions holds the options for the archive command.
type ArchiveOptions struct {
	IO        *iostreams.IOStreams
	PageIDs   []string
	Unarchive bool
	JSON      bool
}

// NewCmdArchive creates the archive command.
func NewCmdArchive(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ArchiveOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "archive <page-id> [page-id...]",
		Short: "Archive or unarchive Confluence pages",
		Long: `Archive one or more Confluence pages.

Archived pages are hidden from normal searches and navigation but can be
restored later using the --unarchive flag.`,
		Example: `  # Archive a single page
  atl confluence page archive 123456

  # Archive multiple pages
  atl confluence page archive 123456 789012 345678

  # Unarchive (restore) a page
  atl confluence page archive 123456 --unarchive

  # Output as JSON
  atl confluence page archive 123456 --json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageIDs = args
			return runArchive(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Unarchive, "unarchive", "u", false, "Unarchive (restore) pages instead of archiving")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// ArchiveOutput represents the output of the archive command.
type ArchiveOutput struct {
	PageIDs []string `json:"page_ids"`
	Action  string   `json:"action"`
	Success bool     `json:"success"`
}

func runArchive(opts *ArchiveOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	action := "archived"
	if opts.Unarchive {
		action = "unarchived"
	}

	// Process each page
	var failedPages []string
	for _, pageID := range opts.PageIDs {
		var err error
		if opts.Unarchive {
			err = confluence.UnarchivePage(ctx, pageID)
		} else {
			err = confluence.ArchivePage(ctx, pageID)
		}
		if err != nil {
			failedPages = append(failedPages, pageID)
			if !opts.JSON {
				fmt.Fprintf(opts.IO.Out, "Failed to %s page %s: %v\n", action[:len(action)-1], pageID, err)
			}
		}
	}

	successPages := make([]string, 0, len(opts.PageIDs))
	for _, pageID := range opts.PageIDs {
		found := false
		for _, failed := range failedPages {
			if pageID == failed {
				found = true
				break
			}
		}
		if !found {
			successPages = append(successPages, pageID)
		}
	}

	archiveOutput := &ArchiveOutput{
		PageIDs: successPages,
		Action:  action,
		Success: len(failedPages) == 0,
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, archiveOutput)
	}

	if len(successPages) > 0 {
		if len(successPages) == 1 {
			fmt.Fprintf(opts.IO.Out, "Successfully %s page %s\n", action, successPages[0])
		} else {
			fmt.Fprintf(opts.IO.Out, "Successfully %s %d pages\n", action, len(successPages))
		}
	}

	if len(failedPages) > 0 {
		return fmt.Errorf("failed to %s %d page(s)", action[:len(action)-1], len(failedPages))
	}

	return nil
}
