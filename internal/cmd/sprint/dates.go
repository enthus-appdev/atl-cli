// Package sprint implements the `atl sprint` command group for managing the
// Jira sprint lifecycle.
package sprint

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const isoDate = "2006-01-02"

// parseDuration parses a sprint duration. It accepts day ("14d") and week
// ("2w") suffixes — which time.ParseDuration does not support — and otherwise
// falls back to time.ParseDuration ("48h", "90m"). The result must be positive.
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	last := s[len(s)-1]
	if last == 'd' || last == 'w' {
		n, err := strconv.Atoi(s[:len(s)-1])
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("invalid duration %q (use e.g. 14d, 2w)", s)
		}
		unit := 24 * time.Hour
		if last == 'w' {
			unit = 7 * 24 * time.Hour
		}
		return time.Duration(n) * unit, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid duration %q (use e.g. 14d, 2w, 48h)", s)
	}
	return d, nil
}

// parseISODate parses a YYYY-MM-DD date in UTC.
func parseISODate(s string) (time.Time, error) {
	t, err := time.Parse(isoDate, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q (use YYYY-MM-DD)", s)
	}
	return t, nil
}

// resolveActiveDates computes start/end for activating a sprint. start defaults
// to now when unset; end is the explicit end date when given, otherwise
// start+duration. now is injected for testability.
func resolveActiveDates(startStr, endStr string, duration time.Duration, now time.Time) (time.Time, time.Time, error) {
	start := now
	if startStr != "" {
		t, err := parseISODate(startStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		start = t
	}
	end := start.Add(duration)
	if endStr != "" {
		t, err := parseISODate(endStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end = t
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("end date must be after start date")
	}
	return start, end, nil
}

// buildCreateBody assembles the POST /sprint request body. boardID and name are
// required; goal and dates are included only when provided.
func buildCreateBody(boardID int, name, goal, startStr, endStr string) (map[string]any, error) {
	if boardID <= 0 {
		return nil, fmt.Errorf("--board is required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("--name is required")
	}
	body := map[string]any{
		"name":          name,
		"originBoardId": boardID,
	}
	if goal != "" {
		body["goal"] = goal
	}
	if startStr != "" {
		t, err := parseISODate(startStr)
		if err != nil {
			return nil, err
		}
		body["startDate"] = t.Format(time.RFC3339)
	}
	if endStr != "" {
		t, err := parseISODate(endStr)
		if err != nil {
			return nil, err
		}
		body["endDate"] = t.Format(time.RFC3339)
	}
	return body, nil
}

// buildActivateBody assembles a partial update that activates a sprint.
func buildActivateBody(start, end time.Time) map[string]any {
	return map[string]any{
		"state":     "active",
		"startDate": start.Format(time.RFC3339),
		"endDate":   end.Format(time.RFC3339),
	}
}

// buildEditBody assembles a partial sprint update from the set fields. At least
// one field must be provided.
func buildEditBody(name, goal, startStr, endStr string) (map[string]any, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if goal != "" {
		body["goal"] = goal
	}
	if startStr != "" {
		t, err := parseISODate(startStr)
		if err != nil {
			return nil, err
		}
		body["startDate"] = t.Format(time.RFC3339)
	}
	if endStr != "" {
		t, err := parseISODate(endStr)
		if err != nil {
			return nil, err
		}
		body["endDate"] = t.Format(time.RFC3339)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("nothing to update: set at least one of --name, --goal, --start-date, --end-date")
	}
	return body, nil
}

// dateOnly returns the YYYY-MM-DD prefix of an RFC3339/ISO date string.
func dateOnly(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
