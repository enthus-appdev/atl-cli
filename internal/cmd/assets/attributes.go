package assets

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// attributeFlags renders the non-default properties of an attribute definition,
// omitting the ordinary ones so the column carries only what distinguishes this
// attribute from a plain optional single-value field.
func attributeFlags(attribute api.AssetObjectTypeAttribute) string {
	var flags []string
	if attribute.Required() {
		flags = append(flags, "required")
	}
	// Assets spells an unbounded upper cardinality as a negative number. Treat
	// only a stated bound above one, or that unbounded form, as multi-valued: an
	// absent or zero field is not evidence of anything and must not be labeled.
	if attribute.MaximumCardinality > 1 || attribute.MaximumCardinality < 0 {
		flags = append(flags, "multi")
	}
	if attribute.Label {
		flags = append(flags, "label")
	}
	if attribute.UniqueAttribute {
		flags = append(flags, "unique")
	}
	if attribute.System {
		flags = append(flags, "system")
	}
	if attribute.Hidden {
		flags = append(flags, "hidden")
	}
	if !attribute.Editable {
		flags = append(flags, "read-only")
	}
	return strings.Join(flags, ", ")
}

func newCmdAttributes(ios *iostreams.IOStreams, common *commonOptions) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:     "attributes <object-type-id>",
		Aliases: []string{"fields"},
		Short:   "List the attributes defined on an Assets object type",
		Long: `List every attribute an Assets object type defines, with its numeric id.

This reads the type, not an object. An Assets object omits each attribute it
holds no value for, so a single object can never show that an attribute is
absent from its type - only this can.

The object type's name is printed above the table. Check it names the type you
meant: a wrong id and a type that genuinely lacks an attribute look the same.`,
		Example: `  atl --context prod jira assets attributes 9
  atl --context sandbox jira assets attributes 14 --json

  # Does this type carry a Status attribute at all?
  atl --context prod jira assets attributes 9 --json | jq '.attributes[].name'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := common.client()
			if err != nil {
				return err
			}

			if err := client.RequireObjectTypeReadScopes(); err != nil {
				return err
			}

			objectType, err := client.ObjectType(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			attributes, err := client.ObjectTypeAttributes(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if jsonOut {
				return output.JSON(ios.Out, map[string]any{
					"objectType": objectType,
					"attributes": attributes,
				})
			}

			fmt.Fprintf(ios.Out, "%s\t%s\n", objectType.ID, terminalText(objectType.Name))
			rows := make([][]string, 0, len(attributes))
			for _, attribute := range attributes {
				rows = append(rows, []string{
					attribute.ID,
					terminalText(attribute.Name),
					terminalText(attribute.DefaultType.Name),
					attributeFlags(attribute),
				})
			}
			output.SimpleTable(ios.Out, []string{"ID", "ATTRIBUTE", "TYPE", "FLAGS"}, rows)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
