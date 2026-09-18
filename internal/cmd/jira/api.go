package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// APIOptions holds the options for the api command.
type APIOptions struct {
	IO         *iostreams.IOStreams
	Path       string
	APIVersion string
}

// NewCmdAPI creates the api passthrough command.
func NewCmdAPI(ios *iostreams.IOStreams) *cobra.Command {
	opts := &APIOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "api [GET] <path>",
		Short: "Make a read-only GET request to the Jira REST API",
		Long: `Make a read-only GET request against the Jira Cloud platform REST API and
print the JSON response.

This is an escape hatch for endpoints atl does not model as first-class
commands (editmeta, project metadata, and similar read-only lookups). Only GET
is supported — atl deliberately does not expose write passthrough.

<path> is relative to the REST base, with or without a leading slash:
  issue/NX-1234/editmeta
  /project/NX/securitylevel

--api-version selects the platform API version. Both expose the same resources
and differ in how they carry rich text: v3 uses ADF, v2 uses wiki markup. Read
through v2 when a plain-text description is easier to work with than ADF.`,
		Example: `  # Inspect an issue's edit metadata (allowed fields and values)
  atl jira api GET issue/NX-1234/editmeta

  # List a project's issue security levels
  atl jira api project/NX/securitylevel

  # Read a description as wiki markup instead of ADF
  atl --context prod jira api --api-version 2 issue/NX-1234?fields=description

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
			// A lone "GET" is the method with no path, not a path named "GET".
			if strings.EqualFold(path, "GET") {
				return fmt.Errorf("missing <path>\n\nExample: atl jira api GET issue/NX-1234/editmeta")
			}
			// Validated here as well as in the API layer so an unsupported version is
			// reported as such, rather than behind whatever error building the
			// authenticated client happens to produce first.
			if err := api.ValidateJiraAPIVersion(opts.APIVersion); err != nil {
				return err
			}
			opts.Path = path
			return runAPI(opts)
		},
	}

	cmd.Flags().StringVar(&opts.APIVersion, "api-version", api.DefaultJiraAPIVersion,
		fmt.Sprintf("Jira platform REST API version (%s)", strings.Join(api.SupportedJiraAPIVersions, ", ")))

	return cmd
}

func runAPI(opts *APIOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	raw, err := jira.RawGetVersion(ctx, opts.APIVersion, opts.Path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	return writeIndentedJSON(opts.IO.Out, raw)
}

// writeIndentedJSON pretty-prints raw JSON bytes, falling back to verbatim
// output when the body is not JSON. It indents the raw bytes rather than
// decoding into interface{} and re-encoding: a round-trip through interface{}
// coerces every JSON number to float64, silently corrupting integers above 2^53
// (Jira ids can exceed it). json.Indent reflows whitespace without touching the
// values.
func writeIndentedJSON(w io.Writer, raw json.RawMessage) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		// Not JSON (unexpected for the REST API) — emit the raw bytes verbatim.
		fmt.Fprintln(w, string(raw))
		return nil
	}
	buf.WriteByte('\n')
	_, err := w.Write(buf.Bytes())
	return err
}
