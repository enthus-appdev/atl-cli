package jira

import (
	"github.com/spf13/cobra"

	assetsCmd "github.com/enthus-appdev/atl-cli/internal/cmd/assets"
	boardCmd "github.com/enthus-appdev/atl-cli/internal/cmd/board"
	issueCmd "github.com/enthus-appdev/atl-cli/internal/cmd/issue"
	smCmd "github.com/enthus-appdev/atl-cli/internal/cmd/sm"
	sprintCmd "github.com/enthus-appdev/atl-cli/internal/cmd/sprint"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdJira creates the jira command group.
func NewCmdJira(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "jira",
		Aliases: []string{"j"},
		Short:   "Work with Jira",
		Long:    `View and manage Jira issues, boards, and Service Management.`,
	}

	cmd.AddCommand(issueCmd.NewCmdIssue(ios))
	cmd.AddCommand(boardCmd.NewCmdBoard(ios))
	cmd.AddCommand(smCmd.NewCmdSM(ios))
	cmd.AddCommand(sprintCmd.NewCmdSprint(ios))
	cmd.AddCommand(assetsCmd.NewCmdAssets(ios))

	return cmd
}
