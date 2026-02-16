package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/config"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

func newCmdCurrentContext(ios *iostreams.IOStreams) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "current-context",
		Short: "Show the current active host",
		Long:  `Show the current active Atlassian host and its alias (if any).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCurrentContext(ios, jsonOutput)
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

// CurrentContextOutput represents the JSON output for current-context.
type CurrentContextOutput struct {
	Hostname string `json:"hostname"`
	Alias    string `json:"alias,omitempty"`
}

func runCurrentContext(ios *iostreams.IOStreams, jsonOutput bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.CurrentHost == "" {
		if jsonOutput {
			return output.JSON(ios.Out, CurrentContextOutput{})
		}
		fmt.Fprintln(ios.Out, "No current context set.")
		fmt.Fprintln(ios.Out, "Run 'atl auth login' to authenticate or 'atl config use-context' to switch.")
		return nil
	}

	alias := cfg.AliasForHost(cfg.CurrentHost)

	if jsonOutput {
		return output.JSON(ios.Out, CurrentContextOutput{
			Hostname: cfg.CurrentHost,
			Alias:    alias,
		})
	}

	if alias != "" {
		fmt.Fprintf(ios.Out, "%s (%s)\n", alias, cfg.CurrentHost)
	} else {
		fmt.Fprintln(ios.Out, cfg.CurrentHost)
	}

	return nil
}
