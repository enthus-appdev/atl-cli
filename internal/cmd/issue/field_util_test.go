package issue

import (
	"reflect"
	"strings"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/api"
)

// TestIsSystemField_Security guards that "security" resolves as a system field.
// If it were treated as a custom field, a field-file keyed on "security" would
// be sent through name-to-ID lookup and rejected (the security field is absent
// from the field list), which was the original friction this work removes.
func TestIsSystemField_Security(t *testing.T) {
	if !isSystemField("security") {
		t.Error("isSystemField(\"security\") = false, want true")
	}
	if !isSystemField("Security") {
		t.Error("isSystemField(\"Security\") = false, want true (case-insensitive)")
	}
}

func TestProjectKeyFromIssueKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"NX-1234", "NX"},
		{"PROJ-1", "PROJ"},
		{"ABC-DEF-5", "ABC-DEF"}, // split on the last hyphen
		{"NX", ""},
		{"", ""},
		{"-5", ""},
	}
	for _, tt := range tests {
		if got := projectKeyFromIssueKey(tt.in); got != tt.want {
			t.Errorf("projectKeyFromIssueKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMatchSecurityLevel(t *testing.T) {
	levels := []*api.SecurityLevel{
		{ID: "10000", Name: "Developer only"},
		{ID: "10001", Name: "Managers"},
	}

	t.Run("by exact name", func(t *testing.T) {
		got, err := matchSecurityLevel(levels, "Developer only")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "10000" {
			t.Errorf("ID = %q, want 10000", got.ID)
		}
	})

	t.Run("by name case-insensitive", func(t *testing.T) {
		got, err := matchSecurityLevel(levels, "developer ONLY")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "10000" {
			t.Errorf("ID = %q, want 10000", got.ID)
		}
	})

	t.Run("by numeric id", func(t *testing.T) {
		got, err := matchSecurityLevel(levels, "10001")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "Managers" {
			t.Errorf("Name = %q, want Managers", got.Name)
		}
	})

	t.Run("unknown lists available", func(t *testing.T) {
		_, err := matchSecurityLevel(levels, "Nope")
		if err == nil {
			t.Fatal("expected error for unknown level")
		}
		if !strings.Contains(err.Error(), "Developer only") || !strings.Contains(err.Error(), "Managers") {
			t.Errorf("error should list available levels, got: %v", err)
		}
	})

	t.Run("no scheme", func(t *testing.T) {
		_, err := matchSecurityLevel(nil, "Developer only")
		if err == nil {
			t.Fatal("expected error when project has no levels")
		}
		if !strings.Contains(err.Error(), "no issue security scheme") {
			t.Errorf("error should mention missing scheme, got: %v", err)
		}
	})
}

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
