package sprint

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"14d", 14 * 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"48h", 48 * time.Hour, false},
		{"", 0, true},
		{"abc", 0, true},
		{"14x", 0, true},
		{"-3d", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseDuration(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseDuration(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveActiveDates(t *testing.T) {
	now := time.Date(2026, 6, 23, 9, 0, 0, 0, time.UTC)
	dur := 14 * 24 * time.Hour

	// default start = now, end = now+duration
	start, end, err := resolveActiveDates("", "", dur, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !start.Equal(now) {
		t.Errorf("start = %v, want %v", start, now)
	}
	if !end.Equal(now.Add(dur)) {
		t.Errorf("end = %v, want %v", end, now.Add(dur))
	}

	// explicit start, duration-derived end
	start, end, err = resolveActiveDates("2026-07-01", "", dur, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(want) {
		t.Errorf("start = %v, want %v", start, want)
	}
	if !end.Equal(want.Add(dur)) {
		t.Errorf("end = %v, want %v", end, want.Add(dur))
	}

	// explicit both
	_, end, err = resolveActiveDates("2026-07-01", "2026-07-10", dur, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if dateOnly(end.Format(time.RFC3339)) != "2026-07-10" {
		t.Errorf("end = %v, want 2026-07-10", end)
	}

	// end before start → error
	if _, _, err := resolveActiveDates("2026-07-10", "2026-07-01", dur, now); err == nil {
		t.Error("expected error when end before start")
	}
}

func TestBuildCreateBody(t *testing.T) {
	// minimal: only required fields
	body, err := buildCreateBody(42, "Sprint 30", "", "", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if body["name"] != "Sprint 30" || body["originBoardId"] != 42 {
		t.Errorf("body = %v", body)
	}
	if _, ok := body["goal"]; ok {
		t.Error("goal should be omitted when empty")
	}
	if _, ok := body["startDate"]; ok {
		t.Error("startDate should be omitted when empty")
	}

	// with goal + explicit start date
	body, err = buildCreateBody(42, "Sprint 30", "Ship MI cutover", "2026-06-23", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if body["goal"] != "Ship MI cutover" {
		t.Errorf("goal = %v", body["goal"])
	}
	if dateOnly(body["startDate"].(string)) != "2026-06-23" {
		t.Errorf("startDate = %v", body["startDate"])
	}

	// validation
	if _, err := buildCreateBody(0, "x", "", "", ""); err == nil {
		t.Error("expected error for missing board")
	}
	if _, err := buildCreateBody(42, "", "", "", ""); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestBuildEditBody(t *testing.T) {
	body, err := buildEditBody("New Name", "New Goal", "", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if body["name"] != "New Name" || body["goal"] != "New Goal" {
		t.Errorf("body = %v", body)
	}
	// goal-only
	body, err = buildEditBody("", "Only Goal", "", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, ok := body["name"]; ok {
		t.Error("name should be omitted")
	}
	// nothing → error
	if _, err := buildEditBody("", "", "", ""); err == nil {
		t.Error("expected error when no fields set")
	}
}

func TestBuildActivateBody(t *testing.T) {
	start := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	end := start.Add(14 * 24 * time.Hour)
	body := buildActivateBody(start, end)
	if body["state"] != "active" {
		t.Errorf("state = %v", body["state"])
	}
	if dateOnly(body["startDate"].(string)) != "2026-06-23" {
		t.Errorf("startDate = %v", body["startDate"])
	}
	if dateOnly(body["endDate"].(string)) != "2026-07-07" {
		t.Errorf("endDate = %v", body["endDate"])
	}
}
