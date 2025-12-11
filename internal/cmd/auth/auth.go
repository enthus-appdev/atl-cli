package auth

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdAuth creates the auth command group.
func NewCmdAuth(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Atlassian",
		Long:  `Manage authentication state for Atlassian Cloud.`,
	}

	cmd.AddCommand(NewCmdSetup(ios))
	cmd.AddCommand(NewCmdLogin(ios))
	cmd.AddCommand(NewCmdLogout(ios))
	cmd.AddCommand(NewCmdStatus(ios))
	cmd.AddCommand(NewCmdRefresh(ios))

	return cmd
}
