package issue

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/enthus-appdev/atl-cli/internal/api"
)

// systemFieldKeys maps a lowercased Jira system field name to the key the
// REST API expects in a fields payload. Jira matches those keys
// case-sensitively, so "Resolution" is rejected as an unknown field.
var systemFieldKeys = map[string]string{
	"summary": "summary", "description": "description", "issuetype": "issuetype",
	"project": "project", "reporter": "reporter", "assignee": "assignee",
	"priority": "priority", "labels": "labels", "components": "components",
	"fixversions": "fixVersions", "versions": "versions", "duedate": "duedate",
	"environment": "environment", "resolution": "resolution", "status": "status",
	"created": "created", "updated": "updated", "parent": "parent",
	"security": "security",
}

// referenceFields are system fields whose value must be a reference object
// ({"id": ...} or {"name": ...}); Jira rejects a bare string or number.
var referenceFields = map[string]bool{"resolution": true, "priority": true}

// isSystemField checks if a field name is a known Jira system field.
func isSystemField(name string) bool {
	_, ok := systemFieldKeys[strings.ToLower(name)]
	return ok
}

func referenceValue(value string) map[string]string {
	trimmed := strings.TrimSpace(value)
	if strings.Trim(trimmed, "0123456789") == "" {
		return map[string]string{"id": trimmed}
	}
	return map[string]string{"name": trimmed}
}

// projectKeyFromIssueKey returns the project key portion of an issue key
// (e.g. "NX-1234" -> "NX"). Jira project keys contain no hyphens, so the key is
// everything before the first hyphen. Returns "" when there is no hyphen.
func projectKeyFromIssueKey(issueKey string) string {
	if idx := strings.Index(issueKey, "-"); idx > 0 {
		return issueKey[:idx]
	}
	return ""
}

// securityFilterMatches reports whether a lowercased --field filter should
// surface the synthetic "Security Level" row. The filter must be a prefix of
// "securitylevel" (min 3 chars) after stripping spaces/hyphens/underscores, so
// "sec", "security", "security level", and "securitylevel" match while a short
// common letter ("s") or an unrelated substring ("level") does not — a prefix
// is how a user narrows toward this field, and it keeps a stray one-letter
// filter from triggering the (explicit-request) security fetch.
func securityFilterMatches(fieldLower string) bool {
	norm := strings.NewReplacer(" ", "", "-", "", "_", "").Replace(fieldLower)
	return len(norm) >= 3 && strings.HasPrefix("securitylevel", norm)
}

// matchSecurityLevel resolves a user-supplied name or numeric id against
// a project's issue security levels. Numeric input matches by id; others
// match by case-insensitive name. If unknown, returns an error listing
// available levels for the caller to surface.
func matchSecurityLevel(levels []*api.SecurityLevel, input string) (*api.SecurityLevel, error) {
	trimmed := strings.TrimSpace(input)
	for _, l := range levels {
		if l.ID == trimmed {
			return l, nil
		}
	}
	for _, l := range levels {
		if strings.EqualFold(l.Name, trimmed) {
			return l, nil
		}
	}

	available := make([]string, 0, len(levels))
	for _, l := range levels {
		available = append(available, l.Name)
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("security level %q not found: project has no issue security scheme", input)
	}
	return nil, fmt.Errorf("security level %q not found\n\nAvailable levels: %s", input, strings.Join(available, ", "))
}

// resolveSecurityLevelID fetches a project's security levels and resolves the
// input (name or id) to its numeric id, ready for the "security" field.
func resolveSecurityLevelID(ctx context.Context, jira *api.JiraService, projectKey, input string) (string, error) {
	levels, err := jira.GetProjectSecurityLevels(ctx, projectKey)
	if err != nil {
		return "", fmt.Errorf("failed to get security levels for project %s: %w", projectKey, err)
	}
	level, err := matchSecurityLevel(levels, input)
	if err != nil {
		return "", err
	}
	return level.ID, nil
}

// ParseCustomField resolves a key=value pair into a field ID and properly
// typed value for the Jira API. Handles name-to-ID resolution and
// type-aware value coercion (select -> {value:...}, textarea -> ADF, number).
func ParseCustomField(ctx context.Context, jira *api.JiraService, raw string) (string, interface{}, error) {
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid field format: %s (expected key=value)", raw)
	}
	key, value := parts[0], parts[1]

	var resolvedField *api.Field

	if strings.HasPrefix(key, "customfield_") {
		resolvedField, _ = jira.GetFieldByID(ctx, key)
	} else if canonical, ok := systemFieldKeys[strings.ToLower(key)]; ok {
		key = canonical
		if referenceFields[key] {
			if strings.TrimSpace(value) == "" {
				return "", nil, fmt.Errorf("field %s requires a value (id or name)", key)
			}
			return key, referenceValue(value), nil
		}
	} else {
		var err error
		resolvedField, err = jira.GetFieldByName(ctx, key)
		if err != nil {
			return "", nil, fmt.Errorf("failed to look up field '%s': %w", key, err)
		}
		if resolvedField == nil {
			return "", nil, fmt.Errorf("field not found: %s\n\nUse 'atl jira issue fields --search \"%s\"' to find available fields", key, key)
		}
		key = resolvedField.ID
	}

	fieldValue := coerceFieldValue(resolvedField, value)

	// If the field was converted to ADF (textarea), resolve any @[Name] mentions
	if adfDoc, ok := fieldValue.(*api.ADF); ok {
		if err := api.ResolveMentions(ctx, adfDoc, jira.NewMentionResolver()); err != nil {
			return "", nil, fmt.Errorf("failed to process mentions in field '%s': %w", key, err)
		}
	}

	return key, fieldValue, nil
}

// coerceFieldValue converts a string value to the appropriate type
// based on the field's schema.
func coerceFieldValue(field *api.Field, value string) interface{} {
	if field != nil && field.Schema != nil {
		customType := field.Schema.Custom
		if strings.Contains(customType, "select") || strings.Contains(customType, "radiobuttons") {
			return map[string]string{"value": value}
		}
		if strings.Contains(customType, "multiselect") || strings.Contains(customType, "multicheckboxes") {
			vals := strings.Split(value, ",")
			options := make([]map[string]string, len(vals))
			for i, v := range vals {
				options[i] = map[string]string{"value": strings.TrimSpace(v)}
			}
			return options
		}
		if strings.Contains(customType, "textarea") {
			// Support literal \n for newlines and \\ for literal backslashes.
			// Handles: "line1\nline2" → two lines, "C:\\path" → C:\path
			value = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(value, `\\`, "\x00"), `\n`, "\n"), "\x00", `\`)
			return api.TextToADF(value)
		}
		// Labels-type fields: both the standard system "labels" field
		// (Custom == "") and custom label fields (Custom contains "labels",
		// e.g. "...customfieldtypes:labels"). Detected via Schema.Type=="array"
		// with string items, or by the "labels" marker in Schema.Custom.
		isLabelsCustom := strings.Contains(customType, "labels")
		isStringArray := field.Schema.Type == "array" && field.Schema.Items == "string"
		isUntypedArray := field.Schema.Type == "array" && field.Schema.Custom == ""
		if isLabelsCustom || isStringArray || isUntypedArray {
			raw := strings.Split(value, ",")
			vals := make([]string, 0, len(raw))
			for _, v := range raw {
				if trimmed := strings.TrimSpace(v); trimmed != "" {
					vals = append(vals, trimmed)
				}
			}
			return vals
		}
	}

	if numVal, err := strconv.ParseFloat(value, 64); err == nil {
		return numVal
	}
	return value
}
