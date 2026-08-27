package assets

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// commonOptions holds the auth/target flags shared by every assets subcommand.
type commonOptions struct {
	Workspace string
}

func (o *commonOptions) addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&o.Workspace, "workspace", "", "Assets workspace id (default: $ATLASSIAN_ASSETS_WORKSPACE or auto-discovered)")
}

// client builds an Assets client from the current host's OAuth session.
func (o *commonOptions) client() (*api.AssetsClient, error) {
	workspace := o.Workspace
	if workspace == "" {
		workspace = os.Getenv("ATLASSIAN_ASSETS_WORKSPACE")
	}

	client, err := api.NewClientFromConfig()
	if err != nil {
		return nil, err
	}
	return api.NewAssetsClient(client, workspace), nil
}

// NewCmdAssets creates the assets command group.
func NewCmdAssets(ios *iostreams.IOStreams) *cobra.Command {
	opts := &commonOptions{}

	cmd := &cobra.Command{
		Use:   "assets",
		Short: "Work with Jira Assets (CMDB)",
		Long: `Query the Jira Service Management Assets (CMDB) workspace.

Assets uses the current atl host and OAuth login. The workspace id is
auto-discovered if it is not supplied.

Existing tokens do not gain newly configured CMDB scopes automatically. If an
Assets request returns 403 after the app scopes changed, re-run
'atl auth login --hostname <host>' for that site.`,
	}

	opts.addFlags(cmd)
	cmd.AddCommand(newCmdCount(ios, opts))
	cmd.AddCommand(newCmdAQL(ios, opts))
	cmd.AddCommand(newCmdObject(ios, opts))

	return cmd
}
