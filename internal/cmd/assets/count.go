package assets

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

func newCmdCount(ios *iostreams.IOStreams, common *commonOptions) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "count",
		Short: "Show object counts per schema and the workspace total",
		Long: `List every object schema with its current object count, plus the
workspace-wide total — useful for tracking how close the workspace is to the
Assets object limit.`,
		Example: `  atl assets count
  atl assets count --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := common.client()
			if err != nil {
				return err
			}

			schemas, err := client.Schemas(cmd.Context())
			if err != nil {
				return err
			}

			total := 0
			for _, s := range schemas {
				total += s.ObjectCount
			}

			if jsonOut {
				return output.JSON(ios.Out, map[string]any{
					"schemas": schemas,
					"total":   total,
				})
			}

			rows := make([][]string, 0, len(schemas)+1)
			for _, s := range schemas {
				rows = append(rows, []string{s.ObjectSchemaKey, s.Name, fmt.Sprint(s.ObjectCount), fmt.Sprint(s.ObjectTypeCount)})
			}
			rows = append(rows, []string{"", "TOTAL", fmt.Sprint(total), ""})
			output.SimpleTable(ios.Out, []string{"KEY", "SCHEMA", "OBJECTS", "TYPES"}, rows)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
