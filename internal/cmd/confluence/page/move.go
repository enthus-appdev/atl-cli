package page

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// MoveOptions holds the options for the move command.
type MoveOptions struct {
	IO       *iostreams.IOStreams
	PageID   string
	TargetID string
	Space    string
	Position string
	JSON     bool
}

// NewCmdMove creates the move command.
func NewCmdMove(ios *iostreams.IOStreams) *cobra.Command {
	opts := &MoveOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "move <page-id>",
		Short: "Move a Confluence page to a new location",
		Long: `Move a Confluence page to a new location.

You can move a page to be a child of another page, or position it
before/after a sibling page. You can also move pages between spaces.`,
		Example: `  # Move a page to be a child of another page
  atl confluence page move 123456 --target 789012

  # Move a page before a sibling (same parent as target)
  atl confluence page move 123456 --target 789012 --position before

  # Move a page after a sibling (same parent as target)
  atl confluence page move 123456 --target 789012 --position after

  # Move a page to a different space (as child of space homepage)
  atl confluence page move 123456 --space NEWSPACE

  # Output as JSON
  atl confluence page move 123456 --target 789012 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageID = args[0]

			// Validate flags
			if opts.TargetID == "" && opts.Space == "" {
				return fmt.Errorf("either --target or --space is required")
			}
			if opts.TargetID != "" && opts.Space != "" {
				return fmt.Errorf("cannot use both --target and --space")
			}

			return runMove(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.TargetID, "target", "t", "", "Target page ID to move relative to")
	cmd.Flags().StringVarP(&opts.Space, "space", "s", "", "Move to a different space (as child of homepage)")
	cmd.Flags().StringVarP(&opts.Position, "position", "p", "append", "Position relative to target: append (child), before, after")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// MoveOutput represents the output of the move command.
type MoveOutput struct {
	PageID   string `json:"page_id"`
	TargetID string `json:"target_id,omitempty"`
	Space    string `json:"space,omitempty"`
	Position string `json:"position"`
	Success  bool   `json:"success"`
}

func runMove(opts *MoveOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	var moveOutput *MoveOutput

	if opts.Space != "" {
		// Move to a different space
		err = confluence.MovePageToSpace(ctx, opts.PageID, opts.Space)
		if err != nil {
			return fmt.Errorf("failed to move page to space %s: %w", opts.Space, err)
		}

		moveOutput = &MoveOutput{
			PageID:   opts.PageID,
			Space:    opts.Space,
			Position: "child of homepage",
			Success:  true,
		}
	} else {
		// Move relative to target page
		position := api.MovePosition(opts.Position)

		// Validate position
		switch position {
		case api.MovePositionAppend, api.MovePositionBefore, api.MovePositionAfter:
			// valid
		default:
			return fmt.Errorf("invalid position %q: must be 'append', 'before', or 'after'", opts.Position)
		}

		err = confluence.MovePage(ctx, opts.PageID, position, opts.TargetID)
		if err != nil {
			return fmt.Errorf("failed to move page: %w", err)
		}

		moveOutput = &MoveOutput{
			PageID:   opts.PageID,
			TargetID: opts.TargetID,
			Position: opts.Position,
			Success:  true,
		}
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, moveOutput)
	}

	if opts.Space != "" {
		fmt.Fprintf(opts.IO.Out, "Successfully moved page %s to space %s\n", opts.PageID, opts.Space)
	} else {
		positionDesc := "as child of"
		if opts.Position == "before" {
			positionDesc = "before"
		} else if opts.Position == "after" {
			positionDesc = "after"
		}
		fmt.Fprintf(opts.IO.Out, "Successfully moved page %s %s %s\n", opts.PageID, positionDesc, opts.TargetID)
	}

	return nil
}
