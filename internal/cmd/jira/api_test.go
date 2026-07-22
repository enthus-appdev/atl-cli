package jira

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// TestNewCmdAPI_ArgGuards covers the argument validation that runs before any
// network call: only GET is allowed, and a lone "GET" is a missing path rather
// than a path literally named "GET". These paths return before the API client
// is constructed, so they need no HTTP.
func TestNewCmdAPI_ArgGuards(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"rejects non-GET method", []string{"POST", "issue/NX-1"}, "only GET is supported"},
		{"rejects PUT method", []string{"put", "issue/NX-1"}, "only GET is supported"},
		{"lone GET is missing path", []string{"GET"}, "missing <path>"},
		{"lone lowercase get is missing path", []string{"get"}, "missing <path>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmdAPI(iostreams.Test())
			cmd.SetArgs(tt.args)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestWriteIndentedJSON_PreservesLargeIntegers is the regression guard for the
// interface{} round-trip: a JSON integer above 2^53 must survive verbatim, not
// be coerced to float64 and re-rendered.
func TestWriteIndentedJSON_PreservesLargeIntegers(t *testing.T) {
	var buf bytes.Buffer
	raw := json.RawMessage(`{"id":9007199254740993}`)
	if err := writeIndentedJSON(&buf, raw); err != nil {
		t.Fatalf("writeIndentedJSON error: %v", err)
	}
	if !strings.Contains(buf.String(), "9007199254740993") {
		t.Errorf("large integer not preserved; got: %s", buf.String())
	}
}

// TestWriteIndentedJSON_NonJSONFallback verifies a non-JSON body is emitted
// verbatim rather than erroring.
func TestWriteIndentedJSON_NonJSONFallback(t *testing.T) {
	var buf bytes.Buffer
	raw := json.RawMessage("not json at all")
	if err := writeIndentedJSON(&buf, raw); err != nil {
		t.Fatalf("writeIndentedJSON error: %v", err)
	}
	if !strings.Contains(buf.String(), "not json at all") {
		t.Errorf("non-JSON body not echoed; got: %s", buf.String())
	}
}
