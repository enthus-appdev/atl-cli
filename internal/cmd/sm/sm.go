package sm

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdSM creates the service management command group.
func NewCmdSM(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sm",
		Aliases: []string{"service-management"},
		Short:   "Work with Jira Service Management",
		Long:    `List service desks, request types, and their fields.`,
	}

	cmd.AddCommand(NewCmdServiceDesk(ios))
	cmd.AddCommand(NewCmdRequestType(ios))

	return cmd
}
