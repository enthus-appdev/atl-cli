package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// RefreshOptions holds the options for the refresh command.
type RefreshOptions struct {
	IO       *iostreams.IOStreams
	Hostname string
}

// NewCmdRefresh creates the refresh command.
func NewCmdRefresh(ios *iostreams.IOStreams) *cobra.Command {
	opts := &RefreshOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh authentication tokens",
		Long:  `Force a refresh of the authentication tokens for an Atlassian host.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRefresh(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Hostname, "hostname", "", "The hostname to refresh tokens for")

	return cmd
}

func runRefresh(opts *RefreshOptions) error {
	// TODO: Implement token refresh
	// 1. Get refresh token from keyring
	// 2. Exchange for new access token
	// 3. Store new tokens

	fmt.Fprintln(opts.IO.Out, "Token refresh not yet implemented.")
	return nil
}
