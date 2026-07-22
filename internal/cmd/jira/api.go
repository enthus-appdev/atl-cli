package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// APIOptions holds the options for the api command.
type APIOptions struct {
	IO   *iostreams.IOStreams
	Path string
}

// NewCmdAPI creates the api passthrough command.
func NewCmdAPI(ios *iostreams.IOStreams) *cobra.Command {
	opts := &APIOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "api [GET] <path>",
		Short: "Make a read-only GET request to the Jira REST API",
		Long: `Make a read-only GET request against the Jira Cloud REST API (v3) and print
the JSON response.

This is an escape hatch for endpoints atl does not model as first-class
commands (editmeta, project metadata, and similar read-only lookups). Only GET
is supported — atl deliberately does not expose write passthrough.

<path> is relative to the REST v3 base, with or without a leading slash:
  issue/NX-1234/editmeta
  /project/NX/securitylevel`,
		Example: `  # Inspect an issue's edit metadata (allowed fields and values)
  atl jira api GET issue/NX-1234/editmeta

  # List a project's issue security levels
  atl jira api project/NX/securitylevel

  # Pipe into jq
  atl jira api GET issue/NX-1234/editmeta | jq '.fields | keys'`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Accept an optional leading method arg; only GET is permitted.
			path := args[0]
			if len(args) == 2 {
				if !strings.EqualFold(args[0], "GET") {
					return fmt.Errorf("only GET is supported (read-only); got %q", args[0])
				}
				path = args[1]
			}
			opts.Path = path
			return runAPI(opts)
		},
	}

	return cmd
}

func runAPI(opts *APIOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	raw, err := jira.RawGet(ctx, opts.Path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	// Re-decode so output.JSON re-indents the body consistently with other
	// --json output rather than echoing the server's whitespace.
	var body interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		// Not JSON (unexpected for the REST API) — emit the raw bytes verbatim.
		fmt.Fprintln(opts.IO.Out, string(raw))
		return nil
	}

	return output.JSON(opts.IO.Out, body)
}
