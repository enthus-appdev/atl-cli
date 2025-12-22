package comment

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdComment creates the comment command group.
func NewCmdComment(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage comments on Jira issues",
		Long: `Add, edit, delete, or view comments on Jira issues.

Use subcommands to manage comments:
  list   - View comments on an issue
  add    - Add a new comment
  edit   - Edit an existing comment
  delete - Delete a comment`,
		Example: `  # List comments on an issue
  atl issue comment list PROJ-1234

  # Add a comment
  atl issue comment add PROJ-1234 --body "This is my comment"

  # Edit a comment
  atl issue comment edit PROJ-1234 --id 12345 --body "Updated text"

  # Delete a comment
  atl issue comment delete PROJ-1234 --id 12345`,
	}

	cmd.AddCommand(NewCmdList(ios))
	cmd.AddCommand(NewCmdAdd(ios))
	cmd.AddCommand(NewCmdEdit(ios))
	cmd.AddCommand(NewCmdDelete(ios))

	return cmd
}
