package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	authCmd "github.com/enthus-appdev/atl-cli/internal/cmd/auth"
	confluenceCmd "github.com/enthus-appdev/atl-cli/internal/cmd/confluence"
	configCmd "github.com/enthus-appdev/atl-cli/internal/cmd/config"
	issueCmd "github.com/enthus-appdev/atl-cli/internal/cmd/issue"
	worklogCmd "github.com/enthus-appdev/atl-cli/internal/cmd/worklog"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// BuildInfo contains version and build information.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// Execute runs the root command and returns an exit code.
func Execute(ios *iostreams.IOStreams, buildInfo BuildInfo) int {
	rootCmd := NewRootCmd(ios, buildInfo)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(ios.ErrOut, "Error: %s\n", err)
		return 1
	}
	return 0
}

// NewRootCmd creates the root command for the CLI.
func NewRootCmd(ios *iostreams.IOStreams, buildInfo BuildInfo) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "atl",
		Short: "Atlassian CLI - Work with Jira, Confluence, and Tempo from the command line",
		Long: `atl is a CLI tool for interacting with Atlassian products.

It provides commands for:
  - Jira: View, create, and manage issues
  - Confluence: Read and edit pages
  - Tempo: Log and manage worklogs

Get started by running 'atl auth login' to authenticate with your Atlassian account.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       buildInfo.Version,
	}

	// Set custom version template
	cmd.SetVersionTemplate(fmt.Sprintf("atl version %s\ncommit: %s\nbuilt: %s\n",
		buildInfo.Version, buildInfo.Commit, buildInfo.Date))

	// Set I/O streams
	cmd.SetIn(ios.In)
	cmd.SetOut(ios.Out)
	cmd.SetErr(ios.ErrOut)

	// Add subcommands
	cmd.AddCommand(authCmd.NewCmdAuth(ios))
	cmd.AddCommand(issueCmd.NewCmdIssue(ios))
	cmd.AddCommand(confluenceCmd.NewCmdConfluence(ios))
	cmd.AddCommand(worklogCmd.NewCmdWorklog(ios))
	cmd.AddCommand(configCmd.NewCmdConfig(ios))
	cmd.AddCommand(newVersionCmd(ios, buildInfo))
	cmd.AddCommand(newCompletionCmd(ios))

	return cmd
}

// newVersionCmd creates the version command.
func newVersionCmd(ios *iostreams.IOStreams, buildInfo BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(ios.Out, "atl version %s\n", buildInfo.Version)
			fmt.Fprintf(ios.Out, "commit: %s\n", buildInfo.Commit)
			fmt.Fprintf(ios.Out, "built: %s\n", buildInfo.Date)
		},
	}
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
