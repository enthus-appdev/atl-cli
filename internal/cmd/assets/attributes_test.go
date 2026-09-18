package assets

import (
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/api"
)

func TestAttributeFlags(t *testing.T) {
	var ordinary api.AssetObjectTypeAttribute
	ordinary.MaximumCardinality = 1
	ordinary.Editable = true

	required := ordinary
	required.MinimumCardinality = 1

	multi := ordinary
	multi.MaximumCardinality = -1

	importKey := ordinary
	importKey.System = true
	importKey.Editable = false
	importKey.UniqueAttribute = true
	importKey.MinimumCardinality = 1

	tests := []struct {
		name      string
		attribute api.AssetObjectTypeAttribute
		want      string
	}{
		{"ordinary attribute has no flags", ordinary, ""},
		{"minimum cardinality marks required", required, "required"},
		{"unbounded cardinality marks multi", multi, "multi"},
		{"import key shows every distinguishing flag", importKey, "required, unique, system, read-only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := attributeFlags(tt.attribute); got != tt.want {
				t.Errorf("attributeFlags() = %q, want %q", got, tt.want)
			}
		})
	}
}
