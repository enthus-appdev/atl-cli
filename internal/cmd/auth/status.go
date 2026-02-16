package auth

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// StatusOptions holds the options for the status command.
type StatusOptions struct {
	IO       *iostreams.IOStreams
	Hostname string
	JSON     bool
}

// NewCmdStatus creates the status command.
func NewCmdStatus(ios *iostreams.IOStreams) *cobra.Command {
	opts := &StatusOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "View authentication status",
		Long:  `View authentication status for Atlassian hosts.`,
		Example: `  # View authentication status for all hosts
  atl auth status

  # View authentication status for a specific host
  atl auth status --hostname mycompany.atlassian.net

  # Output as JSON
  atl auth status --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Hostname, "hostname", "", "The hostname to check status for")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// AuthStatus represents the authentication status for a host.
type AuthStatus struct {
	Hostname      string `json:"hostname"`
	CloudID       string `json:"cloud_id,omitempty"`
	Authenticated bool   `json:"authenticated"`
	TokenExpired  bool   `json:"token_expired,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	Current       bool   `json:"current"`
}

func runStatus(opts *StatusOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Hosts) == 0 {
		if opts.JSON {
			return output.JSON(opts.IO.Out, []AuthStatus{})
		}
		fmt.Fprintln(opts.IO.Out, "You are not logged in to any Atlassian hosts.")
		fmt.Fprintln(opts.IO.Out, "Run 'atl auth login' to authenticate.")
		return nil
	}

	// Resolve alias if --hostname is provided
	if opts.Hostname != "" {
		opts.Hostname = cfg.ResolveHost(opts.Hostname)
	}

	var statuses []AuthStatus

	for hostname, hostCfg := range cfg.Hosts {
		if opts.Hostname != "" && opts.Hostname != hostname {
			continue
		}

		status := AuthStatus{
			Hostname: hostname,
			CloudID:  hostCfg.CloudID,
			Current:  hostname == cfg.CurrentHost,
		}

		tokens, err := auth.GetToken(hostname)
		if err != nil {
			status.Authenticated = false
		} else if tokens == nil {
			status.Authenticated = false
		} else {
			status.Authenticated = true
			status.TokenExpired = tokens.IsExpired()
			status.ExpiresAt = tokens.ExpiresAt.Format(time.RFC3339)
		}

		statuses = append(statuses, status)
	}

	if opts.Hostname != "" && len(statuses) == 0 {
		return fmt.Errorf("host %s not found in configuration", opts.Hostname)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, statuses)
	}

	// Plain text output
	for _, status := range statuses {
		currentMarker := ""
		if status.Current {
			currentMarker = " (current)"
		}

		fmt.Fprintf(opts.IO.Out, "Host: %s%s\n", status.Hostname, currentMarker)

		if status.CloudID != "" {
			fmt.Fprintf(opts.IO.Out, "  Cloud ID: %s\n", status.CloudID)
		}

		if status.Authenticated {
			if status.TokenExpired {
				fmt.Fprintf(opts.IO.Out, "  Status: %s\n", output.Warning.Render("Token expired"))
				fmt.Fprintln(opts.IO.Out, "  Run 'atl auth refresh' to refresh the token")
			} else {
				fmt.Fprintf(opts.IO.Out, "  Status: %s\n", output.Success.Render("Authenticated"))
				fmt.Fprintf(opts.IO.Out, "  Token expires: %s\n", status.ExpiresAt)
			}
		} else {
			fmt.Fprintf(opts.IO.Out, "  Status: %s\n", output.Error.Render("Not authenticated"))
			fmt.Fprintln(opts.IO.Out, "  Run 'atl auth login' to authenticate")
		}
		fmt.Fprintln(opts.IO.Out)
	}

	return nil
}
