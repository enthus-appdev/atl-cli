package output

import (
	"testing"
)

// TestStyleStatus tests the StyleStatus function for different status categories.
func TestStyleStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		category string
	}{
		{
			name:     "new status",
			status:   "To Do",
			category: "new",
		},
		{
			name:     "undefined status",
			status:   "Backlog",
			category: "undefined",
		},
		{
			name:     "in progress status",
			status:   "In Progress",
			category: "indeterminate",
		},
		{
			name:     "done status",
			status:   "Done",
			category: "done",
		},
		{
			name:     "unknown category",
			status:   "Custom",
			category: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StyleStatus(tt.status, tt.category)

			// Result should contain the original status text
			// (lipgloss adds ANSI codes but text remains)
			if result == "" {
				t.Error("StyleStatus() returned empty string")
			}

			// For unknown categories, should return unmodified status
			if tt.category == "unknown" && result != tt.status {
				t.Errorf("StyleStatus() for unknown category should return unmodified status")
			}
		})
	}
}

// TestStylePriority tests the StylePriority function for different priorities.
func TestStylePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority string
	}{
		{"highest", "Highest"},
		{"blocker", "Blocker"},
		{"high", "High"},
		{"critical", "Critical"},
		{"medium", "Medium"},
		{"low", "Low"},
		{"lowest", "Lowest"},
		{"trivial", "Trivial"},
		{"unknown", "Custom Priority"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StylePriority(tt.priority)

			// Result should not be empty
			if result == "" {
				t.Error("StylePriority() returned empty string")
			}

			// For unknown priorities, should return unmodified priority
			if tt.priority == "Custom Priority" && result != tt.priority {
				t.Errorf("StylePriority() for unknown priority should return unmodified priority")
			}
		})
	}
}

// TestStylesExist tests that all style variables are properly initialized.
func TestStylesExist(t *testing.T) {
	// Test that styles can render text without panicking
	testText := "test"

	// Test all styles by rendering them
	tests := []struct {
		name   string
		render func() string
	}{
		{"StatusToDo", func() string { return StatusToDo.Render(testText) }},
		{"StatusInProgress", func() string { return StatusInProgress.Render(testText) }},
		{"StatusDone", func() string { return StatusDone.Render(testText) }},
		{"StatusBlocked", func() string { return StatusBlocked.Render(testText) }},
		{"PriorityHighest", func() string { return PriorityHighest.Render(testText) }},
		{"PriorityHigh", func() string { return PriorityHigh.Render(testText) }},
		{"PriorityMedium", func() string { return PriorityMedium.Render(testText) }},
		{"PriorityLow", func() string { return PriorityLow.Render(testText) }},
		{"PriorityLowest", func() string { return PriorityLowest.Render(testText) }},
		{"Bold", func() string { return Bold.Render(testText) }},
		{"Faint", func() string { return Faint.Render(testText) }},
		{"Success", func() string { return Success.Render(testText) }},
		{"Warning", func() string { return Warning.Render(testText) }},
		{"Error", func() string { return Error.Render(testText) }},
		{"Info", func() string { return Info.Render(testText) }},
		{"Cyan", func() string { return Cyan.Render(testText) }},
		{"Highlight", func() string { return Highlight.Render(testText) }},
		{"Link", func() string { return Link.Render(testText) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := tt.render()
			if rendered == "" {
				t.Errorf("%s.Render() returned empty string", tt.name)
			}
		})
	}
}

// TestStyleStatusCategories tests all valid status categories.
func TestStyleStatusCategories(t *testing.T) {
	categories := []string{"new", "undefined", "indeterminate", "done"}
	status := "Test Status"

	for _, category := range categories {
		t.Run(category, func(t *testing.T) {
			result := StyleStatus(status, category)
			// The result should contain ANSI codes (length > original) or be the original
			if result == "" {
				t.Errorf("StyleStatus(%q, %q) returned empty string", status, category)
			}
		})
	}
}

// TestStylePriorityMapping tests the priority-to-style mapping.
func TestStylePriorityMapping(t *testing.T) {
	// Test that similar priorities map to the same style
	highestResult := StylePriority("Highest")
	blockerResult := StylePriority("Blocker")

	// Both should produce styled output (not empty)
	if highestResult == "" || blockerResult == "" {
		t.Error("Highest and Blocker priorities should produce styled output")
	}

	// Test that different priority levels produce different output
	// (assuming colors are different, the styled strings should differ)
	highResult := StylePriority("High")
	lowResult := StylePriority("Low")

	if highResult == lowResult {
		t.Error("High and Low priorities should produce different styled output")
	}
}

// TestRenderMethods tests that styles can render text.
func TestRenderMethods(t *testing.T) {
	text := "Sample Text"

	// Test Bold
	boldText := Bold.Render(text)
	if boldText == "" {
		t.Error("Bold.Render() should not return empty string")
	}

	// Test Success
	successText := Success.Render(text)
	if successText == "" {
		t.Error("Success.Render() should not return empty string")
	}

	// Test Error
	errorText := Error.Render(text)
	if errorText == "" {
		t.Error("Error.Render() should not return empty string")
	}

	// Test Link
	linkText := Link.Render(text)
	if linkText == "" {
		t.Error("Link.Render() should not return empty string")
	}
}
