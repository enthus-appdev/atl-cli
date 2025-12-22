package issue

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/cmd/issue/comment"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdIssue creates the issue command group.
func NewCmdIssue(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "issue",
		Aliases: []string{"i"},
		Short:   "Work with Jira issues",
		Long:    `Create, view, and manage Jira issues.`,
	}

	cmd.AddCommand(NewCmdView(ios))
	cmd.AddCommand(NewCmdList(ios))
	cmd.AddCommand(NewCmdCreate(ios))
	cmd.AddCommand(NewCmdEdit(ios))
	cmd.AddCommand(NewCmdTransition(ios))
	cmd.AddCommand(comment.NewCmdComment(ios))
	cmd.AddCommand(NewCmdAssign(ios))
	cmd.AddCommand(NewCmdLink(ios))
	cmd.AddCommand(NewCmdFields(ios))
	cmd.AddCommand(NewCmdSprint(ios))
	cmd.AddCommand(NewCmdFlag(ios))
	cmd.AddCommand(NewCmdWebLink(ios))
	cmd.AddCommand(NewCmdTypes(ios))
	cmd.AddCommand(NewCmdPriorities(ios))
	cmd.AddCommand(NewCmdAttachment(ios))

	return cmd
}
