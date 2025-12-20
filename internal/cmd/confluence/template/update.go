package template

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// UpdateOptions holds the options for the update command.
type UpdateOptions struct {
	IO          *iostreams.IOStreams
	TemplateID  string
	Name        string
	Body        string
	Description string
	JSON        bool
}

// NewCmdUpdate creates the update command.
func NewCmdUpdate(ios *iostreams.IOStreams) *cobra.Command {
	opts := &UpdateOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "update <template-id>",
		Short: "Update a Confluence template",
		Long: `Update an existing Confluence content template.

Note: Blueprint templates cannot be updated via the REST API.

The body must be in Confluence storage format (HTML with Confluence macros).`,
		Example: `  # Update template body
  atl confluence template update 12345678 --body "<h1>Updated Content</h1>"

  # Update name and body
  atl confluence template update 12345678 --name "New Name" --body "<p>New content</p>"

  # Update with description
  atl confluence template update 12345678 --body "<p>Content</p>" --description "Updated description"

  # Output as JSON
  atl confluence template update 12345678 --body "<p>Content</p>" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.TemplateID = args[0]
			if opts.Body == "" {
				return fmt.Errorf("--body flag is required")
			}
			return runUpdate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Template name (uses existing if not provided)")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Template body in storage format (required)")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Template description")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// TemplateUpdateOutput represents the output of the update command.
type TemplateUpdateOutput struct {
	TemplateID  string `json:"template_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SpaceKey    string `json:"space_key,omitempty"`
}

func runUpdate(opts *UpdateOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	// If name not provided, get existing template to preserve name
	name := opts.Name
	description := opts.Description
	if name == "" {
		existing, err := confluence.GetTemplate(ctx, opts.TemplateID)
		if err != nil {
			return fmt.Errorf("failed to get existing template: %w", err)
		}
		name = existing.Name
		if description == "" {
			description = existing.Description
		}
	}

	template, err := confluence.UpdateTemplate(ctx, opts.TemplateID, name, opts.Body, description)
	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	spaceKey := ""
	if template.Space != nil {
		spaceKey = template.Space.Key
	}

	updateOutput := &TemplateUpdateOutput{
		TemplateID:  template.TemplateID,
		Name:        template.Name,
		Description: template.Description,
		SpaceKey:    spaceKey,
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, updateOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Updated template: %s\n", updateOutput.Name)
	fmt.Fprintf(opts.IO.Out, "ID: %s\n", updateOutput.TemplateID)

	return nil
}
