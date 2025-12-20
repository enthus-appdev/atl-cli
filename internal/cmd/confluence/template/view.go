package template

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// ViewOptions holds the options for the view command.
type ViewOptions struct {
	IO         *iostreams.IOStreams
	TemplateID string
	Raw        bool
	JSON       bool
}

// NewCmdView creates the view command.
func NewCmdView(ios *iostreams.IOStreams) *cobra.Command {
	opts := &ViewOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "view <template-id>",
		Short: "View a Confluence template",
		Long:  `View details and content of a Confluence content template.`,
		Example: `  # View a template
  atl confluence template view 12345678

  # View raw storage format
  atl confluence template view 12345678 --raw

  # Output as JSON
  atl confluence template view 12345678 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.TemplateID = args[0]
			return runView(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Raw, "raw", "r", false, "Show raw storage format")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// TemplateViewOutput represents the output of the view command.
type TemplateViewOutput struct {
	TemplateID  string `json:"template_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	SpaceKey    string `json:"space_key,omitempty"`
	Body        string `json:"body,omitempty"`
}

func runView(opts *ViewOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	template, err := confluence.GetTemplate(ctx, opts.TemplateID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	spaceKey := ""
	if template.Space != nil {
		spaceKey = template.Space.Key
	}

	body := ""
	if template.Body != nil && template.Body.Storage != nil {
		body = template.Body.Storage.Value
	}

	viewOutput := &TemplateViewOutput{
		TemplateID:  template.TemplateID,
		Name:        template.Name,
		Description: template.Description,
		Type:        template.TemplateType,
		SpaceKey:    spaceKey,
		Body:        body,
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, viewOutput)
	}

	fmt.Fprintf(opts.IO.Out, "# %s\n\n", template.Name)
	fmt.Fprintf(opts.IO.Out, "ID: %s\n", template.TemplateID)
	fmt.Fprintf(opts.IO.Out, "Type: %s\n", template.TemplateType)
	if spaceKey != "" {
		fmt.Fprintf(opts.IO.Out, "Space: %s\n", spaceKey)
	} else {
		fmt.Fprintf(opts.IO.Out, "Space: (global)\n")
	}
	if template.Description != "" {
		fmt.Fprintf(opts.IO.Out, "Description: %s\n", template.Description)
	}

	if opts.Raw && body != "" {
		fmt.Fprintf(opts.IO.Out, "\n## Raw Content\n\n%s\n", body)
	}

	return nil
}
