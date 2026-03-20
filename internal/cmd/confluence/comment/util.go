package comment

import (
	"fmt"
	"os"
)

// resolveBody reads body content from a file if bodyFile is set.
// Returns an error if both body and bodyFile are provided.
func resolveBody(body *string, bodyFile string) error {
	if *body != "" && bodyFile != "" {
		return fmt.Errorf("--body and --body-file are mutually exclusive")
	}
	if bodyFile != "" {
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return fmt.Errorf("failed to read body file: %w", err)
		}
		*body = string(data)
	}
	return nil
}
