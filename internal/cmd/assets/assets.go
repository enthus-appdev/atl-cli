package assets

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// commonOptions holds the auth/target flags shared by every assets subcommand.
type commonOptions struct {
	Email     string
	Workspace string
}

func (o *commonOptions) addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&o.Email, "email", "", "Atlassian account email (default: $ATLASSIAN_EMAIL or current host user)")
	cmd.PersistentFlags().StringVar(&o.Workspace, "workspace", "", "Assets workspace id (default: $ATLASSIAN_ASSETS_WORKSPACE or auto-discovered)")
}

// client builds a Basic-auth Assets client from flags, environment, and the
// current atl host. The API token is only ever read from the environment.
func (o *commonOptions) client() (*api.AssetsClient, error) {
	token := os.Getenv("ATLASSIAN_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("ATLASSIAN_API_TOKEN is not set — Assets uses Basic auth, not atl's OAuth login; create a token at https://id.atlassian.com/manage-profile/security/api-tokens")
	}

	email := o.Email
	if email == "" {
		email = os.Getenv("ATLASSIAN_EMAIL")
	}
	workspace := o.Workspace
	if workspace == "" {
		workspace = os.Getenv("ATLASSIAN_ASSETS_WORKSPACE")
	}

	siteBase := ""
	if cfg, err := config.Load(); err == nil {
		if hc := cfg.CurrentHostConfig(); hc != nil {
			if email == "" {
				email = hc.User
			}
			proto := hc.Protocol
			if proto == "" {
				proto = "https"
			}
			if hc.Hostname != "" {
				siteBase = proto + "://" + hc.Hostname
			}
		}
	}

	if email == "" {
		return nil, fmt.Errorf("no account email — pass --email, set $ATLASSIAN_EMAIL, or log in to a host that records a user")
	}

	return api.NewAssetsClient(siteBase, email, token, workspace), nil
}

// NewCmdAssets creates the assets command group.
func NewCmdAssets(ios *iostreams.IOStreams) *cobra.Command {
	opts := &commonOptions{}

	cmd := &cobra.Command{
		Use:   "assets",
		Short: "Work with Jira Assets (CMDB)",
		Long: `Query the Jira Service Management Assets (CMDB) workspace.

Assets has its own API and uses Basic auth rather than atl's OAuth login
(the granular OAuth scopes do not cover CMDB objects). Set ATLASSIAN_API_TOKEN;
the account email and site default to your current atl host, and the workspace
id is auto-discovered if not supplied.`,
	}

	opts.addFlags(cmd)
	cmd.AddCommand(newCmdCount(ios, opts))
	cmd.AddCommand(newCmdAQL(ios, opts))

	return cmd
}
