package confluence

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/cmd/confluence/page"
	"github.com/enthus-appdev/atl-cli/internal/cmd/confluence/space"
	"github.com/enthus-appdev/atl-cli/internal/cmd/confluence/template"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdConfluence creates the confluence command group.
func NewCmdConfluence(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "confluence",
		Aliases: []string{"conf", "c"},
		Short:   "Work with Confluence",
		Long:    `Read and manage Confluence pages, spaces, and templates.`,
	}

	cmd.AddCommand(page.NewCmdPage(ios))
	cmd.AddCommand(space.NewCmdSpace(ios))
	cmd.AddCommand(template.NewCmdTemplate(ios))

	return cmd
}
