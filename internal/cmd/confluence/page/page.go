package page

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdPage creates the page command group.
func NewCmdPage(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page",
		Short: "Work with Confluence pages",
		Long:  `View, create, and edit Confluence pages.`,
	}

	cmd.AddCommand(NewCmdView(ios))
	cmd.AddCommand(NewCmdList(ios))
	cmd.AddCommand(NewCmdCreate(ios))
	cmd.AddCommand(NewCmdEdit(ios))
	cmd.AddCommand(NewCmdDelete(ios))
	cmd.AddCommand(NewCmdPublish(ios))
	cmd.AddCommand(NewCmdChildren(ios))
	cmd.AddCommand(NewCmdSearch(ios))
	cmd.AddCommand(NewCmdArchive(ios))
	cmd.AddCommand(NewCmdMove(ios))

	return cmd
}
