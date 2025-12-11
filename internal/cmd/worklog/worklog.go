package worklog

import (
	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// NewCmdWorklog creates the worklog command group.
func NewCmdWorklog(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "worklog",
		Aliases: []string{"wl", "log"},
		Short:   "Work with Tempo worklogs",
		Long:    `Log time, view, and manage Tempo worklogs.`,
	}

	cmd.AddCommand(NewCmdAdd(ios))
	cmd.AddCommand(NewCmdList(ios))
	cmd.AddCommand(NewCmdEdit(ios))
	cmd.AddCommand(NewCmdDelete(ios))

	return cmd
}
