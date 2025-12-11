package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSON writes data as JSON to the writer.
func JSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// JSONCompact writes data as compact JSON to the writer.
func JSONCompact(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(data)
}

// JSONString returns data as a JSON string.
func JSONString(data interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(bytes), nil
}
