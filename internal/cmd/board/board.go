package board

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdBoard creates the board command group.
func NewCmdBoard(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "board",
		Short: "Work with Jira boards",
		Long:  `List boards and manage issue ranking on boards.`,
	}

	cmd.AddCommand(NewCmdList(ios))
	cmd.AddCommand(NewCmdRank(ios))

	return cmd
}
