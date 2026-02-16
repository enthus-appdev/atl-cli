package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

func newCmdUseContext(ios *iostreams.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "use-context <name-or-hostname>",
		Short: "Switch the active Atlassian host",
		Long:  `Switch the active Atlassian host by alias name or hostname.`,
		Example: `  # Switch using an alias
  atl config use-context prod

  # Switch using a hostname
  atl config use-context mycompany.atlassian.net`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUseContext(ios, args[0])
		},
	}
}

func runUseContext(ios *iostreams.IOStreams, nameOrHostname string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	hostname := cfg.ResolveHost(nameOrHostname)

	if cfg.GetHost(hostname) == nil {
		return fmt.Errorf("host %q not found in configuration\n\nUse 'atl auth status' to see configured hosts", hostname)
	}

	cfg.CurrentHost = hostname
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	alias := cfg.AliasForHost(hostname)
	if alias != "" {
		fmt.Fprintf(ios.Out, "Switched to context %q (%s)\n", alias, hostname)
	} else {
		fmt.Fprintf(ios.Out, "Switched to context %s\n", hostname)
	}

	return nil
}
