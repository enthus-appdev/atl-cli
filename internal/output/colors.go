package output

import (
	"github.com/charmbracelet/lipgloss"
)

// Color styles for CLI output.
var (
	// StatusColors for issue/workflow statuses
	StatusToDo       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))  // Gray
	StatusInProgress = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))   // Blue
	StatusDone       = lipgloss.NewStyle().Foreground(lipgloss.Color("35"))   // Green
	StatusBlocked    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))  // Red

	// Priority colors
	PriorityHighest = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
	PriorityHigh    = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // Orange
	PriorityMedium  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // Yellow
	PriorityLow     = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))  // Blue
	PriorityLowest  = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // Gray

	// General styles
	Bold      = lipgloss.NewStyle().Bold(true)
	Faint     = lipgloss.NewStyle().Faint(true)
	Success   = lipgloss.NewStyle().Foreground(lipgloss.Color("35"))  // Green
	Warning   = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // Yellow
	Error     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
	Info      = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))  // Blue
	Cyan      = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))  // Cyan
	Highlight = lipgloss.NewStyle().Foreground(lipgloss.Color("141")) // Purple

	// Link style
	Link = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Underline(true)
)

// StyleStatus returns a styled string based on status category.
func StyleStatus(status string, category string) string {
	switch category {
	case "new", "undefined":
		return StatusToDo.Render(status)
	case "indeterminate":
		return StatusInProgress.Render(status)
	case "done":
		return StatusDone.Render(status)
	default:
		return status
	}
}

// StylePriority returns a styled string based on priority.
func StylePriority(priority string) string {
	switch priority {
	case "Highest", "Blocker":
		return PriorityHighest.Render(priority)
	case "High", "Critical":
		return PriorityHigh.Render(priority)
	case "Medium":
		return PriorityMedium.Render(priority)
	case "Low":
		return PriorityLow.Render(priority)
	case "Lowest", "Trivial":
		return PriorityLowest.Render(priority)
	default:
		return priority
	}
}
