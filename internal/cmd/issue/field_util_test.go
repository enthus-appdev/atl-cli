package issue

import (
	"reflect"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/api"
)

// TestCoerceFieldValue_LabelsCustomField reproduces a bug where label-typed
// custom fields (e.g. NX `Repo`, `Application`) were rejected by Jira because
// `--field "Repo=API"` sent the bare string "API" instead of the required
// `["API"]` array. The schema for label custom fields has:
//
//	type   = "array"
//	items  = "string"
//	custom = "com.atlassian.jira.plugin.system.customfieldtypes:labels"
//
// The previous code only handled labels when Schema.Custom was empty, missing
// the custom-label case entirely.
func TestCoerceFieldValue_LabelsCustomField(t *testing.T) {
	field := &api.Field{
		ID:   "customfield_10410",
		Name: "Repo",
		Schema: &api.FieldSchema{
			Type:   "array",
			Items:  "string",
			Custom: "com.atlassian.jira.plugin.system.customfieldtypes:labels",
		},
	}

	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"single label", "API", []string{"API"}},
		{"multiple labels comma-separated", "API,GUI,Portal", []string{"API", "GUI", "Portal"}},
		{"with spaces around commas", "API, GUI ,Portal", []string{"API", "GUI", "Portal"}},
		{"double commas dropped", "API,,GUI", []string{"API", "GUI"}},
		{"trailing comma dropped", "API,GUI,", []string{"API", "GUI"}},
		{"leading comma dropped", ",API,GUI", []string{"API", "GUI"}},
		{"whitespace-only entry dropped", "API, ,GUI", []string{"API", "GUI"}},
		{"all empty returns empty slice", ",,,", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coerceFieldValue(field, tt.value)
			gotSlice, ok := got.([]string)
			if !ok {
				t.Fatalf("expected []string, got %T (%v)", got, got)
			}
			if !reflect.DeepEqual(gotSlice, tt.want) {
				t.Errorf("coerceFieldValue(%q) = %v, want %v", tt.value, gotSlice, tt.want)
			}
		})
	}
}

// TestCoerceFieldValue_StandardLabelsField verifies the existing behavior for
// the standard system "labels" field (Schema.Custom == "") is preserved.
func TestCoerceFieldValue_StandardLabelsField(t *testing.T) {
	field := &api.Field{
		ID:   "labels",
		Name: "Labels",
		Schema: &api.FieldSchema{
			Type:   "array",
			Items:  "string",
			Custom: "",
		},
	}

	got := coerceFieldValue(field, "bug,urgent")
	want := []string{"bug", "urgent"}
	gotSlice, ok := got.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", got)
	}
	if !reflect.DeepEqual(gotSlice, want) {
		t.Errorf("got %v, want %v", gotSlice, want)
	}
}

// TestCoerceFieldValue_SelectStillWorks verifies select fields are not
// regressed by the new label detection.
func TestCoerceFieldValue_SelectStillWorks(t *testing.T) {
	field := &api.Field{
		ID:   "customfield_10412",
		Name: "Ursprung des Fehlverhaltens",
		Schema: &api.FieldSchema{
			Type:   "option",
			Custom: "com.atlassian.jira.plugin.system.customfieldtypes:select",
		},
	}

	got := coerceFieldValue(field, "aktuell")
	want := map[string]string{"value": "aktuell"}
	gotMap, ok := got.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", got)
	}
	if !reflect.DeepEqual(gotMap, want) {
		t.Errorf("got %v, want %v", gotMap, want)
	}
}

// TestCoerceFieldValue_RadioStillWorks verifies radiobutton fields are not regressed.
func TestCoerceFieldValue_RadioStillWorks(t *testing.T) {
	field := &api.Field{
		ID:   "customfield_10413",
		Name: "Fehlverhalten",
		Schema: &api.FieldSchema{
			Type:   "option",
			Custom: "com.atlassian.jira.plugin.system.customfieldtypes:radiobuttons",
		},
	}

	got := coerceFieldValue(field, "Ja")
	want := map[string]string{"value": "Ja"}
	gotMap, ok := got.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", got)
	}
	if !reflect.DeepEqual(gotMap, want) {
		t.Errorf("got %v, want %v", gotMap, want)
	}
}
