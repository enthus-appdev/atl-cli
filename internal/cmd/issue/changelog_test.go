package issue

import (
	"bytes"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

func TestNewCmdChangelog(t *testing.T) {
	ios := iostreams.Test()
	cmd := NewCmdChangelog(ios)

	if cmd == nil {
		t.Fatal("NewCmdChangelog() returned nil")
	}
	if cmd.Use != "changelog <issue-key>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "changelog <issue-key>")
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Check flags exist
	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("--json flag should exist")
	}
	fieldFlag := cmd.Flags().Lookup("field")
	if fieldFlag == nil {
		t.Error("--field flag should exist")
	}
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Error("--limit flag should exist")
	}
}

func TestFormatChangelogEntry(t *testing.T) {
	entry := &api.ChangelogEntry{
		ID: "10001",
		Author: &api.User{
			AccountID:   "user-123",
			DisplayName: "Jane Doe",
		},
		Created: "2026-02-03T09:15:22.000+0100",
		Items: []*api.ChangelogItem{
			{
				Field:      "priority",
				FieldType:  "jira",
				FromString: "Medium",
				ToString:   "Highest",
			},
		},
	}

	out := formatChangelogEntryOutput(entry)

	if out.Author != "Jane Doe" {
		t.Errorf("Author = %q, want %q", out.Author, "Jane Doe")
	}
	if out.Created != "2026-02-03T09:15:22.000+0100" {
		t.Errorf("Created = %q, want %q", out.Created, "2026-02-03T09:15:22.000+0100")
	}
	if len(out.Items) != 1 {
		t.Fatalf("Items count = %d, want 1", len(out.Items))
	}
	if out.Items[0].Field != "priority" {
		t.Errorf("Field = %q, want %q", out.Items[0].Field, "priority")
	}
	if out.Items[0].From != "Medium" {
		t.Errorf("From = %q, want %q", out.Items[0].From, "Medium")
	}
	if out.Items[0].To != "Highest" {
		t.Errorf("To = %q, want %q", out.Items[0].To, "Highest")
	}
}

func TestFormatChangelogEntryNoAuthor(t *testing.T) {
	entry := &api.ChangelogEntry{
		ID:      "10001",
		Created: "2026-02-03T09:15:22.000+0100",
		Items: []*api.ChangelogItem{
			{
				Field:    "status",
				ToString: "Done",
			},
		},
	}

	out := formatChangelogEntryOutput(entry)

	if out.Author != "" {
		t.Errorf("Author = %q, want empty string", out.Author)
	}
}

func TestPrintChangelog(t *testing.T) {
	outBuf := &bytes.Buffer{}
	ios := &iostreams.IOStreams{Out: outBuf}

	entries := []*ChangelogEntryOutput{
		{
			Created: "2026-02-03T09:15:22.000+0100",
			Author:  "Jane Doe",
			Items: []*ChangelogItemOutput{
				{Field: "Priority", From: "Medium", To: "Highest"},
			},
		},
		{
			Created: "2026-02-04T14:12:08.000+0100",
			Author:  "John Smith",
			Items: []*ChangelogItemOutput{
				{Field: "Status", From: "In Progress", To: "In Review"},
				{Field: "Assignee", From: "Jane Doe", To: "John Smith"},
			},
		},
	}

	printChangelog(ios, "TEST-123", entries)

	output := outBuf.String()
	expectedStrings := []string{
		"2026-02-03 09:15:22",
		"Jane Doe",
		"Priority",
		`"Medium"`,
		`"Highest"`,
		"2026-02-04 14:12:08",
		"John Smith",
		"Status",
		`"In Progress"`,
		`"In Review"`,
		"Assignee",
	}

	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("Output missing %q\nGot:\n%s", expected, output)
		}
	}
}

func TestPrintChangelogEmpty(t *testing.T) {
	outBuf := &bytes.Buffer{}
	ios := &iostreams.IOStreams{Out: outBuf}

	printChangelog(ios, "TEST-123", nil)

	output := outBuf.String()
	if !contains(output, "No changelog entries") {
		t.Errorf("Expected 'No changelog entries' message, got:\n%s", output)
	}
}

func TestFilterChangelogByField(t *testing.T) {
	entries := []*ChangelogEntryOutput{
		{
			Created: "2026-02-03 09:15:22",
			Author:  "Jane Doe",
			Items: []*ChangelogItemOutput{
				{Field: "Priority", From: "Medium", To: "Highest"},
				{Field: "Status", From: "Open", To: "In Progress"},
			},
		},
		{
			Created: "2026-02-04 14:12:08",
			Author:  "John Smith",
			Items: []*ChangelogItemOutput{
				{Field: "Assignee", From: "Jane Doe", To: "John Smith"},
			},
		},
	}

	filtered := filterChangelogByField(entries, "status")

	if len(filtered) != 1 {
		t.Fatalf("Filtered count = %d, want 1", len(filtered))
	}
	if len(filtered[0].Items) != 1 {
		t.Fatalf("Filtered items count = %d, want 1", len(filtered[0].Items))
	}
	if filtered[0].Items[0].Field != "Status" {
		t.Errorf("Field = %q, want %q", filtered[0].Items[0].Field, "Status")
	}
}

func TestFilterChangelogByFieldNoMatch(t *testing.T) {
	entries := []*ChangelogEntryOutput{
		{
			Created: "2026-02-03 09:15:22",
			Author:  "Jane Doe",
			Items: []*ChangelogItemOutput{
				{Field: "Priority", From: "Medium", To: "Highest"},
			},
		},
	}

	filtered := filterChangelogByField(entries, "nonexistent")

	if len(filtered) != 0 {
		t.Errorf("Filtered count = %d, want 0", len(filtered))
	}
}
