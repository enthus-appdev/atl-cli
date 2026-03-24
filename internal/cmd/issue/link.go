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
	ListLinks  bool
	DeleteID   string
	JSON       bool
}

// NewCmdLink creates the link command.
func NewCmdLink(ios *iostreams.IOStreams) *cobra.Command {
	opts := &LinkOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "link <inward-issue> [outward-issue]",
		Short: "Manage issue links",
		Long: `Create, list, or delete links between Jira issues.

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

  # List links on an issue
  atl issue link PROJ-1 --list

  # Delete a link by ID
  atl issue link PROJ-1 --delete 12345

  # List available link types
  atl issue link --list-types`,
		Args: func(cmd *cobra.Command, args []string) error {
			if opts.ListTypes {
				return nil
			}
			if opts.ListLinks || opts.DeleteID != "" {
				if len(args) != 1 {
					return fmt.Errorf("requires exactly 1 argument: <issue-key>")
				}
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
			if opts.ListLinks {
				opts.InwardKey = args[0]
				return runListLinks(opts)
			}
			if opts.DeleteID != "" {
				opts.InwardKey = args[0]
				return runDeleteLink(opts)
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
	cmd.Flags().BoolVar(&opts.ListLinks, "list", false, "List links on an issue")
	cmd.Flags().StringVar(&opts.DeleteID, "delete", "", "Delete a link by ID")
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

// IssueLinkOutput represents a single issue link in list output.
type IssueLinkOutput struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Direction string `json:"direction"`
	Relation  string `json:"relation"`
	IssueKey  string `json:"issue_key"`
	Summary   string `json:"summary"`
}

// IssueLinksOutput represents the output for listing issue links.
type IssueLinksOutput struct {
	IssueKey string             `json:"issue_key"`
	Links    []*IssueLinkOutput `json:"links"`
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

func runListLinks(opts *LinkOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	links, err := jira.GetIssueLinks(ctx, opts.InwardKey)
	if err != nil {
		return fmt.Errorf("failed to get issue links: %w", err)
	}

	linksOutput := &IssueLinksOutput{
		IssueKey: opts.InwardKey,
		Links:    make([]*IssueLinkOutput, 0, len(links)),
	}

	for _, link := range links {
		lo := &IssueLinkOutput{
			ID:   link.ID,
			Type: link.Type.Name,
		}

		if link.OutwardIssue != nil {
			lo.Direction = "outward"
			lo.Relation = link.Type.Outward
			lo.IssueKey = link.OutwardIssue.Key
			if link.OutwardIssue.Fields != nil {
				lo.Summary = link.OutwardIssue.Fields.Summary
			}
		} else if link.InwardIssue != nil {
			lo.Direction = "inward"
			lo.Relation = link.Type.Inward
			lo.IssueKey = link.InwardIssue.Key
			if link.InwardIssue.Fields != nil {
				lo.Summary = link.InwardIssue.Fields.Summary
			}
		}

		linksOutput.Links = append(linksOutput.Links, lo)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, linksOutput)
	}

	if len(linksOutput.Links) == 0 {
		fmt.Fprintf(opts.IO.Out, "No links found on %s\n", opts.InwardKey)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Links on %s:\n\n", opts.InwardKey)
	headers := []string{"ID", "RELATION", "ISSUE", "SUMMARY"}
	rows := make([][]string, 0, len(linksOutput.Links))

	for _, l := range linksOutput.Links {
		rows = append(rows, []string{l.ID, l.Relation, l.IssueKey, l.Summary})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)
	return nil
}

func runDeleteLink(opts *LinkOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Verify the link exists on this issue before deleting
	links, err := jira.GetIssueLinks(ctx, opts.InwardKey)
	if err != nil {
		return fmt.Errorf("failed to get issue links: %w", err)
	}

	var found *api.IssueLink
	for _, link := range links {
		if link.ID == opts.DeleteID {
			found = link
			break
		}
	}

	if found == nil {
		return fmt.Errorf("link ID %s not found on %s\n\nUse 'atl issue link %s --list' to see links", opts.DeleteID, opts.InwardKey, opts.InwardKey)
	}

	err = jira.DeleteIssueLink(ctx, opts.DeleteID)
	if err != nil {
		return fmt.Errorf("failed to delete link: %w", err)
	}

	var linkedKey string
	if found.OutwardIssue != nil {
		linkedKey = found.OutwardIssue.Key
	} else if found.InwardIssue != nil {
		linkedKey = found.InwardIssue.Key
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, map[string]string{
			"deleted":      opts.DeleteID,
			"issue":        opts.InwardKey,
			"linked_issue": linkedKey,
			"type":         found.Type.Name,
		})
	}

	fmt.Fprintf(opts.IO.Out, "Deleted link %s (%s %s %s)\n", opts.DeleteID, opts.InwardKey, found.Type.Name, linkedKey)
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
