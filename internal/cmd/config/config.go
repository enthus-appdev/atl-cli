package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// NewCmdConfig creates the config command group.
func NewCmdConfig(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View and modify atl configuration settings.`,
	}

	cmd.AddCommand(newCmdGet(ios))
	cmd.AddCommand(newCmdSet(ios))
	cmd.AddCommand(newCmdList(ios))
	cmd.AddCommand(newCmdUseContext(ios))
	cmd.AddCommand(newCmdCurrentContext(ios))
	cmd.AddCommand(newCmdSetAlias(ios))
	cmd.AddCommand(newCmdDeleteAlias(ios))

	return cmd
}

func newCmdGet(ios *iostreams.IOStreams) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Print the value of a configuration key.

Available keys:
  current_host          - The current active Atlassian host
  default_output_format - Default output format (text or json)
  editor                - Editor to use for editing content
  pager                 - Pager to use for long output`,
		Example: `  atl config get current_host
  atl config get editor`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(ios, args[0], jsonOutput)
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func runGet(ios *iostreams.IOStreams, key string, jsonOutput bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	value := cfg.Get(key)

	if jsonOutput {
		return output.JSON(ios.Out, map[string]string{key: value})
	}

	if value == "" {
		fmt.Fprintf(ios.Out, "%s: (not set)\n", key)
	} else {
		fmt.Fprintf(ios.Out, "%s: %s\n", key, value)
	}

	return nil
}

func newCmdSet(ios *iostreams.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Available keys:
  current_host          - The current active Atlassian host
  default_output_format - Default output format (text or json)
  editor                - Editor to use for editing content
  pager                 - Pager to use for long output`,
		Example: `  atl config set current_host mycompany.atlassian.net
  atl config set editor vim
  atl config set default_output_format json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSet(ios, args[0], args[1])
		},
	}
}

func runSet(ios *iostreams.IOStreams, key, value string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Set(key, value); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(ios.Out, "Set %s = %s\n", key, value)
	return nil
}

func newCmdList(ios *iostreams.IOStreams) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all configuration values",
		Long:    `Print all configuration key-value pairs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(ios, jsonOutput)
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

// ConfigListOutput represents the config list output.
type ConfigListOutput struct {
	CurrentHost         string                     `json:"current_host,omitempty"`
	DefaultOutputFormat string                     `json:"default_output_format,omitempty"`
	Editor              string                     `json:"editor,omitempty"`
	Pager               string                     `json:"pager,omitempty"`
	Aliases             map[string]string          `json:"aliases,omitempty"`
	Hosts               map[string]*HostInfoOutput `json:"hosts,omitempty"`
	ConfigFile          string                     `json:"config_file"`
}

// HostInfoOutput represents host configuration.
type HostInfoOutput struct {
	Hostname       string `json:"hostname"`
	CloudID        string `json:"cloud_id,omitempty"`
	DefaultProject string `json:"default_project,omitempty"`
}

func runList(ios *iostreams.IOStreams, jsonOutput bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	listOutput := &ConfigListOutput{
		CurrentHost:         cfg.CurrentHost,
		DefaultOutputFormat: cfg.DefaultOutputFormat,
		Editor:              cfg.Editor,
		Pager:               cfg.Pager,
		ConfigFile:          config.ConfigFile(),
	}

	if len(cfg.Aliases) > 0 {
		listOutput.Aliases = cfg.Aliases
	}

	if len(cfg.Hosts) > 0 {
		listOutput.Hosts = make(map[string]*HostInfoOutput)
		for name, host := range cfg.Hosts {
			listOutput.Hosts[name] = &HostInfoOutput{
				Hostname:       host.Hostname,
				CloudID:        host.CloudID,
				DefaultProject: host.DefaultProject,
			}
		}
	}

	if jsonOutput {
		return output.JSON(ios.Out, listOutput)
	}

	fmt.Fprintf(ios.Out, "Config file: %s\n\n", listOutput.ConfigFile)

	fmt.Fprintln(ios.Out, "Settings:")
	printConfigValue(ios, "  current_host", listOutput.CurrentHost)
	printConfigValue(ios, "  default_output_format", listOutput.DefaultOutputFormat)
	printConfigValue(ios, "  editor", listOutput.Editor)
	printConfigValue(ios, "  pager", listOutput.Pager)

	if len(listOutput.Aliases) > 0 {
		fmt.Fprintln(ios.Out, "")
		fmt.Fprintln(ios.Out, "Aliases:")
		for alias, hostname := range listOutput.Aliases {
			current := ""
			if hostname == cfg.CurrentHost {
				current = " (current)"
			}
			fmt.Fprintf(ios.Out, "  %s: %s%s\n", alias, hostname, current)
		}
	}

	if len(listOutput.Hosts) > 0 {
		fmt.Fprintln(ios.Out, "")
		fmt.Fprintln(ios.Out, "Hosts:")
		for name, host := range listOutput.Hosts {
			fmt.Fprintf(ios.Out, "  %s:\n", name)
			if host.CloudID != "" {
				fmt.Fprintf(ios.Out, "    cloud_id: %s\n", host.CloudID)
			}
			if host.DefaultProject != "" {
				fmt.Fprintf(ios.Out, "    default_project: %s\n", host.DefaultProject)
			}
		}
	}

	return nil
}

func printConfigValue(ios *iostreams.IOStreams, key, value string) {
	if value == "" {
		fmt.Fprintf(ios.Out, "%s: (not set)\n", key)
	} else {
		fmt.Fprintf(ios.Out, "%s: %s\n", key, value)
	}
}
