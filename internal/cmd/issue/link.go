package issue

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// LinkOptions holds the options for the link command.
type LinkOptions struct {
	IO         *iostreams.IOStreams
	InwardKey  string
	OutwardKey string
	LinkType   string
	ListTypes  bool
	JSON       bool
}

// NewCmdLink creates the link command.
func NewCmdLink(ios *iostreams.IOStreams) *cobra.Command {
	opts := &LinkOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "link <inward-issue> <outward-issue>",
		Short: "Link two Jira issues",
		Long: `Create a link between two Jira issues.

Common link types:
  - Blocks      (A blocks B)
  - Cloners     (A clones B)
  - Duplicate   (A duplicates B)
  - Relates     (A relates to B)

Use --list-types to see all available link types for your Jira instance.`,
		Example: `  # Link PROJ-1 blocks PROJ-2
  atl issue link PROJ-1 PROJ-2 --type Blocks

  # Link PROJ-1 relates to PROJ-2
  atl issue link PROJ-1 PROJ-2 --type Relates

  # List available link types
  atl issue link --list-types`,
		Args: func(cmd *cobra.Command, args []string) error {
			if opts.ListTypes {
				return nil
			}
			if len(args) != 2 {
				return fmt.Errorf("requires exactly 2 arguments: <inward-issue> <outward-issue>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.ListTypes {
				return runListLinkTypes(opts)
			}
			opts.InwardKey = args[0]
			opts.OutwardKey = args[1]
			if opts.LinkType == "" {
				return fmt.Errorf("--type flag is required\n\nUse 'atl issue link --list-types' to see available link types")
			}
			return runLink(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.LinkType, "type", "t", "", "Link type (e.g., Blocks, Relates, Duplicate)")
	cmd.Flags().BoolVar(&opts.ListTypes, "list-types", false, "List available link types")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// LinkOutput represents the output after creating a link.
type LinkOutput struct {
	InwardIssue  string `json:"inward_issue"`
	OutwardIssue string `json:"outward_issue"`
	LinkType     string `json:"link_type"`
	Message      string `json:"message"`
}

// LinkTypeOutput represents a link type in the output.
type LinkTypeOutput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

// LinkTypesOutput represents the output for listing link types.
type LinkTypesOutput struct {
	Types []*LinkTypeOutput `json:"types"`
}

func runLink(opts *LinkOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Find the link type
	linkTypes, err := jira.GetIssueLinkTypes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get link types: %w", err)
	}

	var matchedType *api.IssueLinkType
	for _, lt := range linkTypes {
		if strings.EqualFold(lt.Name, opts.LinkType) ||
			strings.EqualFold(lt.Inward, opts.LinkType) ||
			strings.EqualFold(lt.Outward, opts.LinkType) {
			matchedType = lt
			break
		}
	}

	if matchedType == nil {
		return fmt.Errorf("link type not found: %s\n\nUse 'atl issue link --list-types' to see available types", opts.LinkType)
	}

	// Create the link
	err = jira.CreateIssueLink(ctx, opts.InwardKey, opts.OutwardKey, matchedType.Name)
	if err != nil {
		return fmt.Errorf("failed to create link: %w", err)
	}

	linkOutput := &LinkOutput{
		InwardIssue:  opts.InwardKey,
		OutwardIssue: opts.OutwardKey,
		LinkType:     matchedType.Name,
		Message:      fmt.Sprintf("%s %s %s", opts.InwardKey, matchedType.Outward, opts.OutwardKey),
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, linkOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Linked: %s %s %s\n", opts.InwardKey, matchedType.Outward, opts.OutwardKey)
	return nil
}

func runListLinkTypes(opts *LinkOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	linkTypes, err := jira.GetIssueLinkTypes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get link types: %w", err)
	}

	typesOutput := &LinkTypesOutput{
		Types: make([]*LinkTypeOutput, 0, len(linkTypes)),
	}

	for _, lt := range linkTypes {
		typesOutput.Types = append(typesOutput.Types, &LinkTypeOutput{
			ID:      lt.ID,
			Name:    lt.Name,
			Inward:  lt.Inward,
			Outward: lt.Outward,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, typesOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Available link types:\n\n")
	headers := []string{"NAME", "INWARD", "OUTWARD"}
	rows := make([][]string, 0, len(typesOutput.Types))

	for _, t := range typesOutput.Types {
		rows = append(rows, []string{t.Name, t.Inward, t.Outward})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)
	return nil
}
