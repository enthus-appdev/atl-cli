package assets

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

func newCmdAQL(ios *iostreams.IOStreams, common *commonOptions) *cobra.Command {
	var (
		countOnly bool
		limit     int
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "aql <query>",
		Short: "Run an AQL query against the Assets workspace",
		Long: `Run an Assets Query Language (AQL) query.

With --count the result is the exact number of matching objects (the command
paginates, because the Assets endpoint caps its reported total at 1000).
Otherwise the first matching objects are listed.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Exact count of every object in the workspace
  atl assets aql 'objectId > 0' --count

  # Newest objects of one object type
  atl assets aql 'objectTypeId = 36 ORDER BY created DESC' --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ql := args[0]
			client, err := common.client()
			if err != nil {
				return err
			}
			ctx := context.Background()

			if countOnly {
				n, err := client.AQLCount(ctx, ql)
				if err != nil {
					return err
				}
				if jsonOut {
					return output.JSON(ios.Out, map[string]int{"count": n})
				}
				fmt.Fprintln(ios.Out, n)
				return nil
			}

			objs, _, err := client.AQLPage(ctx, ql, 0, limit)
			if err != nil {
				return err
			}
			if jsonOut {
				return output.JSON(ios.Out, objs)
			}
			rows := make([][]string, 0, len(objs))
			for _, o := range objs {
				rows = append(rows, []string{o.ObjectKey, o.ObjectType.Name, o.Created, o.Updated, strings.TrimSpace(o.Label)})
			}
			output.SimpleTable(ios.Out, []string{"KEY", "TYPE", "CREATED", "UPDATED", "LABEL"}, rows)
			return nil
		},
	}

	cmd.Flags().BoolVar(&countOnly, "count", false, "Print only the exact total match count")
	cmd.Flags().IntVar(&limit, "limit", 25, "Max objects to list when not counting")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
