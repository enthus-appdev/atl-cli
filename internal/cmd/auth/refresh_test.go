package auth

import (
	"testing"
	"time"
)

// TestFormatDuration tests the duration formatting function.
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "negative duration (expired)",
			duration: -time.Hour,
			want:     "expired",
		},
		{
			name:     "less than an hour",
			duration: 45 * time.Minute,
			want:     "45m",
		},
		{
			name:     "exactly one hour",
			duration: time.Hour,
			want:     "1h 0m",
		},
		{
			name:     "hours and minutes",
			duration: 2*time.Hour + 30*time.Minute,
			want:     "2h 30m",
		},
		{
			name:     "more than a day",
			duration: 26 * time.Hour,
			want:     "1d 2h",
		},
		{
			name:     "multiple days",
			duration: 72 * time.Hour,
			want:     "3d 0h",
		},
		{
			name:     "zero duration",
			duration: 0,
			want:     "0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

// TestNewCmdRefresh tests the refresh command creation.
func TestNewCmdRefresh(t *testing.T) {
	cmd := NewCmdRefresh(nil)

	if cmd == nil {
		t.Fatal("NewCmdRefresh() returned nil")
	}
	if cmd.Use != "refresh" {
		t.Errorf("Use = %q, want %q", cmd.Use, "refresh")
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Check hostname flag exists
	hostnameFlag := cmd.Flags().Lookup("hostname")
	if hostnameFlag == nil {
		t.Error("--hostname flag should exist")
	}
}

// TestRefreshOptions tests the RefreshOptions struct.
func TestRefreshOptions(t *testing.T) {
	opts := &RefreshOptions{
		IO:       nil,
		Hostname: "example.atlassian.net",
	}

	if opts.Hostname != "example.atlassian.net" {
		t.Errorf("Hostname = %q, want %q", opts.Hostname, "example.atlassian.net")
	}
}
