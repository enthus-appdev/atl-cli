package space

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdSpace creates the space command group.
func NewCmdSpace(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "space",
		Short: "Work with Confluence spaces",
		Long:  `View and manage Confluence spaces.`,
	}

	cmd.AddCommand(NewCmdList(ios))

	return cmd
}
