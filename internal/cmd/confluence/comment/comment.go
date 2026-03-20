package comment

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdComment creates the comment command group.
func NewCmdComment(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage comments on Confluence pages",
		Long: `Add, edit, or view footer comments on Confluence pages.

Use subcommands to manage comments:
  list - View comments on a page
  add  - Add a new comment
  edit - Edit an existing comment`,
		Example: `  # List comments on a page
  atl confluence comment list 1234567

  # Add a comment
  atl confluence comment add 1234567 --body "<p>This looks good!</p>"

  # Reply to an existing comment
  atl confluence comment add 1234567 --body "<p>I agree</p>" --reply-to 9876543

  # Edit a comment
  atl confluence comment edit --id 9876543 --body "<p>Updated text</p>"`,
	}

	cmd.AddCommand(NewCmdList(ios))
	cmd.AddCommand(NewCmdAdd(ios))
	cmd.AddCommand(NewCmdEdit(ios))

	return cmd
}
