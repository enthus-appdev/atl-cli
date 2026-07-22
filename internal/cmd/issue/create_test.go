package issue

import (
	"strings"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// TestNewCmdCreate_RejectsSecurityClear verifies that --security "" on create is
// rejected in RunE (before any network call), since a new issue takes the
// project default and clearing is an edit-only operation.
func TestNewCmdCreate_RejectsSecurityClear(t *testing.T) {
	cmd := NewCmdCreate(iostreams.Test())
	cmd.SetArgs([]string{"--project", "NX", "--type", "Bug", "--summary", "x", "--security", ""})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for --security \"\" on create, got nil")
	}
	if !strings.Contains(err.Error(), "clearing the security level is not supported on create") {
		t.Errorf("error = %q, want the create-clear guidance", err.Error())
	}
}
