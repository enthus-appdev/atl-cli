package template

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// CreateOptions holds the options for the create command.
type CreateOptions struct {
	IO          *iostreams.IOStreams
	Name        string
	Body        string
	Description string
	Space       string
	JSON        bool
}

// NewCmdCreate creates the create command.
func NewCmdCreate(ios *iostreams.IOStreams) *cobra.Command {
	opts := &CreateOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Confluence template",
		Long: `Create a new Confluence content template.

If --space is provided, creates a space-specific template (requires Space Admin permission).
Without --space, creates a global template (requires Confluence Administrator permission).

The body must be in Confluence storage format (HTML with Confluence macros).`,
		Example: `  # Create a space template
  atl confluence template create --space DOCS --name "Meeting Notes" --body "<h1>Meeting Notes</h1><p>Date: </p>"

  # Create with description
  atl confluence template create --space DOCS --name "Meeting Notes" --body "<p>Content</p>" --description "Template for meeting notes"

  # Create a global template (requires admin)
  atl confluence template create --name "Global Template" --body "<p>Content</p>"

  # Output as JSON
  atl confluence template create --space DOCS --name "Test" --body "<p>Test</p>" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Name == "" {
				return fmt.Errorf("--name flag is required")
			}
			if opts.Body == "" {
				return fmt.Errorf("--body flag is required")
			}
			return runCreate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Template name (required)")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Template body in storage format (required)")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Template description")
	cmd.Flags().StringVarP(&opts.Space, "space", "s", "", "Space key (creates space template; omit for global)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// TemplateCreateOutput represents the output of the create command.
type TemplateCreateOutput struct {
	TemplateID  string `json:"template_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SpaceKey    string `json:"space_key,omitempty"`
}

func runCreate(opts *CreateOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	confluence := api.NewConfluenceService(client)

	template, err := confluence.CreateTemplate(ctx, opts.Name, opts.Body, opts.Description, opts.Space)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	spaceKey := ""
	if template.Space != nil {
		spaceKey = template.Space.Key
	}

	createOutput := &TemplateCreateOutput{
		TemplateID:  template.TemplateID,
		Name:        template.Name,
		Description: template.Description,
		SpaceKey:    spaceKey,
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, createOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Created template: %s\n", createOutput.Name)
	fmt.Fprintf(opts.IO.Out, "ID: %s\n", createOutput.TemplateID)
	if spaceKey != "" {
		fmt.Fprintf(opts.IO.Out, "Space: %s\n", spaceKey)
	} else {
		fmt.Fprintf(opts.IO.Out, "Scope: global\n")
	}

	return nil
}
