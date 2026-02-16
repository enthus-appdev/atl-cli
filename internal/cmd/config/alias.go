package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

func newCmdSetAlias(ios *iostreams.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "set-alias <alias> [hostname]",
		Short: "Create or update a host alias",
		Long: `Create or update a named alias for a hostname.

If hostname is omitted, the current host is used.`,
		Example: `  # Alias the current host as "prod"
  atl config set-alias prod

  # Alias a specific host as "sandbox"
  atl config set-alias sandbox mycompany-sandbox.atlassian.net`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			hostname := ""
			if len(args) > 1 {
				hostname = args[1]
			}
			return runSetAlias(ios, args[0], hostname)
		},
	}
}

func runSetAlias(ios *iostreams.IOStreams, alias, hostname string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if hostname == "" {
		hostname = cfg.CurrentHost
	}
	if hostname == "" {
		return fmt.Errorf("no hostname specified and no current host configured\n\nUse 'atl auth login' first or provide a hostname argument")
	}

	if err := cfg.SetAlias(alias, hostname); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(ios.Out, "Alias %q set to %s\n", alias, hostname)
	return nil
}

func newCmdDeleteAlias(ios *iostreams.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-alias <alias>",
		Short: "Remove a host alias",
		Long:  `Remove a named alias from the configuration.`,
		Example: `  atl config delete-alias sandbox`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteAlias(ios, args[0])
		},
	}
}

func runDeleteAlias(ios *iostreams.IOStreams, alias string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Aliases == nil || cfg.Aliases[alias] == "" {
		return fmt.Errorf("alias %q not found", alias)
	}

	cfg.RemoveAlias(alias)

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(ios.Out, "Alias %q removed\n", alias)
	return nil
}
