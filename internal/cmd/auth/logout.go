package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/auth"
	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// LogoutOptions holds the options for the logout command.
type LogoutOptions struct {
	IO       *iostreams.IOStreams
	Hostname string
	All      bool
}

// NewCmdLogout creates the logout command.
func NewCmdLogout(ios *iostreams.IOStreams) *cobra.Command {
	opts := &LogoutOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of an Atlassian host",
		Long:  `Remove authentication credentials for an Atlassian host.`,
		Example: `  # Log out of the current host
  atl auth logout

  # Log out of a specific host
  atl auth logout --hostname mycompany.atlassian.net

  # Log out of all hosts
  atl auth logout --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Hostname, "hostname", "", "The hostname to log out of")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Log out of all hosts")

	return cmd
}

func runLogout(opts *LogoutOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Hosts) == 0 {
		fmt.Fprintln(opts.IO.Out, "You are not logged in to any Atlassian hosts.")
		return nil
	}

	var hostsToRemove []string

	if opts.All {
		for hostname := range cfg.Hosts {
			hostsToRemove = append(hostsToRemove, hostname)
		}
	} else if opts.Hostname != "" {
		if _, ok := cfg.Hosts[opts.Hostname]; !ok {
			return fmt.Errorf("host %s not found in configuration", opts.Hostname)
		}
		hostsToRemove = []string{opts.Hostname}
	} else {
		// Log out of current host
		if cfg.CurrentHost == "" {
			return fmt.Errorf("no current host configured. Use --hostname to specify a host or --all to log out of all hosts")
		}
		hostsToRemove = []string{cfg.CurrentHost}
	}

	for _, hostname := range hostsToRemove {
		// Delete tokens from keyring
		if err := auth.DeleteToken(hostname); err != nil {
			fmt.Fprintf(opts.IO.ErrOut, "Warning: failed to delete tokens for %s: %v\n", hostname, err)
		}

		// Remove from config
		cfg.RemoveHost(hostname)

		fmt.Fprintf(opts.IO.Out, "%s Logged out of %s\n", output.Success.Render("✓"), hostname)
	}

	// Save updated config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
