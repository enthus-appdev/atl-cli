package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"

	authCmd "github.com/enthus-appdev/atl-cli/internal/cmd/auth"
	boardCmd "github.com/enthus-appdev/atl-cli/internal/cmd/board"
	configCmd "github.com/enthus-appdev/atl-cli/internal/cmd/config"
	confluenceCmd "github.com/enthus-appdev/atl-cli/internal/cmd/confluence"
	issueCmd "github.com/enthus-appdev/atl-cli/internal/cmd/issue"
	jiraCmd "github.com/enthus-appdev/atl-cli/internal/cmd/jira"
	smCmd "github.com/enthus-appdev/atl-cli/internal/cmd/sm"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// Execute runs the root command and returns an exit code.
func Execute(ios *iostreams.IOStreams, version string) int {
	rootCmd := NewRootCmd(ios, version)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(ios.ErrOut, "Error: %s\n", err)
		return 1
	}
	return 0
}

// NewRootCmd creates the root command for the CLI.
func NewRootCmd(ios *iostreams.IOStreams, version string) *cobra.Command {
	commit, date := vcsInfo()
	cmd := &cobra.Command{
		Use:   "atl",
		Short: "Atlassian CLI - Work with Jira and Confluence from the command line",
		Long: `atl is a CLI tool for interacting with Atlassian products.

It provides commands for:
  - Jira: View, create, and manage issues, boards, and Service Management (atl jira ...)
  - Confluence: Read and edit pages (atl confluence ...)

Get started by running 'atl auth login' to authenticate with your Atlassian account.

Environment variables:
  ATL_DEBUG=1    Enable debug logging (shows API requests/responses)`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	// Set custom version template
	cmd.SetVersionTemplate(fmt.Sprintf("atl version %s\ncommit: %s\nbuilt:  %s\n",
		version, commit, date))

	// Set I/O streams
	cmd.SetIn(ios.In)
	cmd.SetOut(ios.Out)
	cmd.SetErr(ios.ErrOut)

	// Add subcommands
	cmd.AddCommand(authCmd.NewCmdAuth(ios))
	cmd.AddCommand(jiraCmd.NewCmdJira(ios))
	cmd.AddCommand(confluenceCmd.NewCmdConfluence(ios))
	cmd.AddCommand(configCmd.NewCmdConfig(ios))
	cmd.AddCommand(newVersionCmd(ios, version, commit, date))
	cmd.AddCommand(newCompletionCmd(ios))

	// Deprecated top-level aliases for Jira commands moved under `atl jira`.
	// Hidden from help/completion; warn on use. Remove after the deprecation window.
	cmd.AddCommand(deprecatedAlias(issueCmd.NewCmdIssue(ios), ios, "jira issue"))
	cmd.AddCommand(deprecatedAlias(boardCmd.NewCmdBoard(ios), ios, "jira board"))
	cmd.AddCommand(deprecatedAlias(smCmd.NewCmdSM(ios), ios, "jira sm"))

	return cmd
}

// deprecatedAlias hides a relocated command and warns once on use.
//
// The warning wraps each leaf's PreRun rather than the parent's PersistentPreRun:
// cobra runs only the nearest ancestor's PersistentPreRun, so a future root-level
// PersistentPreRun would otherwise shadow (or be shadowed by) the warning. PreRun
// is per-leaf and leaves none defined today, so wrapping is collision-free.
func deprecatedAlias(cmd *cobra.Command, ios *iostreams.IOStreams, newPath string) *cobra.Command {
	cmd.Hidden = true
	warn := func() {
		fmt.Fprintf(ios.ErrOut, "warning: 'atl %s' is deprecated, use 'atl %s' instead\n", cmd.Name(), newPath)
	}
	var apply func(*cobra.Command)
	apply = func(c *cobra.Command) {
		switch {
		case c.PreRunE != nil:
			next := c.PreRunE
			c.PreRunE = func(cmd *cobra.Command, args []string) error { warn(); return next(cmd, args) }
		case c.PreRun != nil:
			next := c.PreRun
			c.PreRun = func(cmd *cobra.Command, args []string) { warn(); next(cmd, args) }
		default:
			c.PreRun = func(*cobra.Command, []string) { warn() }
		}
		for _, sub := range c.Commands() {
			apply(sub)
		}
	}
	apply(cmd)
	return cmd
}

// newVersionCmd creates the version command.
func newVersionCmd(ios *iostreams.IOStreams, version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(ios.Out, "atl version %s\n", version)
			fmt.Fprintf(ios.Out, "commit: %s\n", commit)
			fmt.Fprintf(ios.Out, "built:  %s\n", date)
		},
	}
}

func vcsInfo() (commit, date string) {
	commit, date = "unknown", "unknown"
	var modified bool
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) > 7 {
				commit = s.Value[:7]
			} else {
				commit = s.Value
			}
		case "vcs.time":
			date = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if modified {
		commit += "-dirty"
	}
	return
}

// newCompletionCmd creates the completion command for shell autocompletion.
func newCompletionCmd(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for atl.

To load completions:

Bash:
  $ source <(atl completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ atl completion bash > /etc/bash_completion.d/atl
  # macOS:
  $ atl completion bash > $(brew --prefix)/etc/bash_completion.d/atl

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ atl completion zsh > "${fpath[1]}/_atl"

Fish:
  $ atl completion fish | source
  # To load completions for each session, execute once:
  $ atl completion fish > ~/.config/fish/completions/atl.fish

PowerShell:
  PS> atl completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> atl completion powershell > atl.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(ios.Out)
			case "zsh":
				return cmd.Root().GenZshCompletion(ios.Out)
			case "fish":
				return cmd.Root().GenFishCompletion(ios.Out, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(ios.Out)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}

	return cmd
}
