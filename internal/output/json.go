// Package output provides formatting utilities for CLI output.
//
// This package supports multiple output formats:
//   - JSON (pretty-printed and compact) for LLM-friendly structured output
//   - Colored text for terminal display using lipgloss styles
//
// All commands in the CLI support a --json flag that uses this package
// to emit structured output suitable for parsing by other tools or LLMs.
package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSON writes data as pretty-printed JSON to the writer with 2-space indentation.
// This is the standard output format when --json flag is used.
func JSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// JSONCompact writes data as compact JSON (no indentation) to the writer.
// Useful when output size matters more than human readability.
func JSONCompact(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(data)
}

// JSONString returns data as a pretty-printed JSON string.
// Useful when you need the JSON as a string rather than writing to a stream.
func JSONString(data interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(bytes), nil
}
