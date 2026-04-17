package sm

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// NewCmdRequestType creates the request-type command group.
func NewCmdRequestType(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "request-type",
		Aliases: []string{"rt"},
		Short:   "Work with request types",
	}

	cmd.AddCommand(newCmdRequestTypeList(ios))
	cmd.AddCommand(newCmdRequestTypeFields(ios))

	return cmd
}

// ── request-type list ──────────────────────────────────────────────

// RequestTypeListOptions holds options for listing request types.
type RequestTypeListOptions struct {
	IO            *iostreams.IOStreams
	ServiceDeskID int
	JSON          bool
}

func newCmdRequestTypeList(ios *iostreams.IOStreams) *cobra.Command {
	opts := &RequestTypeListOptions{IO: ios}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List request types for a service desk",
		Example: `  # List request types for service desk 2
  atl sm request-type list --service-desk-id 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.ServiceDeskID == 0 {
				return fmt.Errorf("--service-desk-id is required")
			}
			return runRequestTypeList(opts)
		},
	}

	cmd.Flags().IntVar(&opts.ServiceDeskID, "service-desk-id", 0, "Service desk ID (required)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

func runRequestTypeList(opts *RequestTypeListOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	sm := api.NewSMService(client)

	types, err := sm.GetRequestTypes(ctx, opts.ServiceDeskID)
	if err != nil {
		return fmt.Errorf("failed to list request types: %w", err)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, types)
	}

	headers := []string{"ID", "Name", "Description"}
	rows := make([][]string, 0, len(types))
	for _, t := range types {
		desc := t.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		rows = append(rows, []string{t.ID, t.Name, desc})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)
	return nil
}

// ── request-type fields ────────────────────────────────────────────

// RequestTypeFieldsOptions holds options for listing request type fields.
type RequestTypeFieldsOptions struct {
	IO            *iostreams.IOStreams
	ServiceDeskID int
	RequestTypeID int
	JSON          bool
}

func newCmdRequestTypeFields(ios *iostreams.IOStreams) *cobra.Command {
	opts := &RequestTypeFieldsOptions{IO: ios}

	cmd := &cobra.Command{
		Use:   "fields",
		Short: "List fields for a request type",
		Example: `  # List fields for request type 26 on service desk 2
  atl sm request-type fields --service-desk-id 2 --request-type-id 26`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.ServiceDeskID == 0 {
				return fmt.Errorf("--service-desk-id is required")
			}
			if opts.RequestTypeID == 0 {
				return fmt.Errorf("--request-type-id is required")
			}
			return runRequestTypeFields(opts)
		},
	}

	cmd.Flags().IntVar(&opts.ServiceDeskID, "service-desk-id", 0, "Service desk ID (required)")
	cmd.Flags().IntVar(&opts.RequestTypeID, "request-type-id", 0, "Request type ID (required)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// RequestTypeFieldOutput is the structured output for a request type field.
type RequestTypeFieldOutput struct {
	FieldID     string `json:"field_id"`
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Visible     bool   `json:"visible"`
	Type        string `json:"type,omitempty"`
	Custom      string `json:"custom,omitempty"`
	CustomID    int    `json:"custom_id,omitempty"`
	Description string `json:"description,omitempty"`
}

func runRequestTypeFields(opts *RequestTypeFieldsOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	sm := api.NewSMService(client)

	result, err := sm.GetRequestTypeFields(ctx, opts.ServiceDeskID, opts.RequestTypeID)
	if err != nil {
		return fmt.Errorf("failed to get request type fields: %w", err)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, result)
	}

	headers := []string{"Field ID", "Name", "Required", "Type", "Custom"}
	rows := make([][]string, 0, len(result.RequestTypeFields))
	for _, f := range result.RequestTypeFields {
		required := ""
		if f.Required {
			required = "*"
		}
		fieldType := ""
		custom := ""
		if f.JiraSchema != nil {
			fieldType = f.JiraSchema.Type
			custom = f.JiraSchema.Custom
		}
		rows = append(rows, []string{f.FieldID, f.Name, required, fieldType, custom})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)
	return nil
}
