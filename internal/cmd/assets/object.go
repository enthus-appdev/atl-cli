package assets

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

func attributeValues(values []api.AssetAttributeValue) []string {
	formatted := make([]string, 0, len(values))
	for _, value := range values {
		if value.DisplayValue != "" {
			formatted = append(formatted, terminalText(value.DisplayValue))
		} else if value.Value != nil {
			formatted = append(formatted, terminalText(fmt.Sprint(value.Value)))
		}
	}
	return formatted
}

func terminalText(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
}

func newCmdObject(ios *iostreams.IOStreams, common *commonOptions) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "object <id>",
		Short: "Get an Assets object and its attributes",
		Example: `  atl --context sandbox jira assets object 9244
  atl --context prod jira assets object 9244 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := common.client()
			if err != nil {
				return err
			}

			object, err := client.Object(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if jsonOut {
				return output.JSON(ios.Out, object)
			}

			fmt.Fprintf(ios.Out, "%s\t%s\t%s\n", terminalText(object.ObjectKey), terminalText(object.ObjectType.Name), strings.TrimSpace(terminalText(object.Label)))
			rows := make([][]string, 0, len(object.Attributes))
			for _, attribute := range object.Attributes {
				name := terminalText(attribute.ObjectTypeAttribute.Name)
				if name == "" {
					name = terminalText(attribute.ObjectTypeAttributeID)
				}
				values := attributeValues(attribute.ObjectAttributeValues)
				rows = append(rows, []string{name, strings.Join(values, ", ")})
			}
			output.SimpleTable(ios.Out, []string{"ATTRIBUTE", "VALUE"}, rows)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
