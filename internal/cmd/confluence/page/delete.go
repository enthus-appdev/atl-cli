package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// DeleteOptions holds the options for the delete command.
type DeleteOptions struct {
	IO      *iostreams.IOStreams
	PageIDs []string
	Force   bool
	JSON    bool
}

// NewCmdDelete creates the delete command.
func NewCmdDelete(ios *iostreams.IOStreams) *cobra.Command {
	opts := &DeleteOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "delete <page-id> [page-id...]",
		Short: "Delete Confluence pages or folders",
		Long: `Permanently delete one or more Confluence pages or folders.

WARNING: This action cannot be undone. Deleted pages are moved to trash
and will be permanently removed after the retention period.

For a reversible option, consider using 'atl confluence page archive' instead.`,
		Example: `  # Delete a single page (will prompt for confirmation)
  atl confluence page delete 123456

  # Delete multiple pages
  atl confluence page delete 123456 789012

  # Delete without confirmation prompt
  atl confluence page delete 123456 --force

  # Delete a folder
  atl confluence page delete 123456  # folders use the same command

  # Output as JSON
  atl confluence page delete 123456 --force --json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageIDs = args
			return runDelete(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// DeleteOutput represents the output of the delete command.
type DeleteOutput struct {
	PageIDs []string `json:"page_ids"`
	Deleted int      `json:"deleted"`
	Failed  int      `json:"failed"`
	Success bool     `json:"success"`
}

func runDelete(opts *DeleteOptions) error {
	// Confirm deletion unless --force is specified
	if !opts.Force && !opts.JSON {
		fmt.Fprintf(opts.IO.Out, "WARNING: This will permanently delete %d page(s)/folder(s).\n", len(opts.PageIDs))
		fmt.Fprintf(opts.IO.Out, "Page IDs: %v\n", opts.PageIDs)
		fmt.Fprint(opts.IO.Out, "Type 'yes' to confirm: ")

		var confirm string
		fmt.Fscanln(opts.IO.In, &confirm)
		if confirm != "yes" {
			return fmt.Errorf("deletion cancelled")
		}
	}

	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	// Process each page
	var deletedPages []string
	var failedPages []string

	for _, pageID := range opts.PageIDs {
		err := confluence.DeletePage(ctx, pageID)
		if err != nil {
			failedPages = append(failedPages, pageID)
			if !opts.JSON {
				fmt.Fprintf(opts.IO.Out, "Failed to delete %s: %v\n", pageID, err)
			}
		} else {
			deletedPages = append(deletedPages, pageID)
		}
	}

	deleteOutput := &DeleteOutput{
		PageIDs: deletedPages,
		Deleted: len(deletedPages),
		Failed:  len(failedPages),
		Success: len(failedPages) == 0,
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, deleteOutput)
	}

	if len(deletedPages) > 0 {
		if len(deletedPages) == 1 {
			fmt.Fprintf(opts.IO.Out, "Successfully deleted page/folder %s\n", deletedPages[0])
		} else {
			fmt.Fprintf(opts.IO.Out, "Successfully deleted %d pages/folders\n", len(deletedPages))
		}
	}

	if len(failedPages) > 0 {
		return fmt.Errorf("failed to delete %d page(s)/folder(s)", len(failedPages))
	}

	return nil
}
