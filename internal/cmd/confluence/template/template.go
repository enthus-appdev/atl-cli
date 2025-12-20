package template

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdTemplate creates the template command group.
func NewCmdTemplate(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Work with Confluence templates",
		Long:  `Create, view, and update Confluence content templates.`,
	}

	cmd.AddCommand(NewCmdView(ios))
	cmd.AddCommand(NewCmdCreate(ios))
	cmd.AddCommand(NewCmdUpdate(ios))

	return cmd
}
