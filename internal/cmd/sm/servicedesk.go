package sm

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// NewCmdServiceDesk creates the service-desk command group.
func NewCmdServiceDesk(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "service-desk",
		Aliases: []string{"sd"},
		Short:   "Work with service desks",
	}

	cmd.AddCommand(newCmdServiceDeskList(ios))

	return cmd
}

// ServiceDeskListOptions holds options for listing service desks.
type ServiceDeskListOptions struct {
	IO   *iostreams.IOStreams
	JSON bool
}

func newCmdServiceDeskList(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ServiceDeskListOptions{IO: ios}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List service desks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceDeskList(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

func runServiceDeskList(opts *ServiceDeskListOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	sm := api.NewSMService(client)

	desks, err := sm.GetServiceDesks(ctx)
	if err != nil {
		return fmt.Errorf("failed to list service desks: %w", err)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, desks)
	}

	headers := []string{"ID", "Project Key", "Project Name"}
	rows := make([][]string, 0, len(desks))
	for _, d := range desks {
		rows = append(rows, []string{d.ID, d.ProjectKey, d.ProjectName})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)
	return nil
}
