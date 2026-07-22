package issue

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/enthus-appdev/atl-cli/internal/api"
)

// isSystemField checks if a field name is a known Jira system field.
func isSystemField(name string) bool {
	systemFields := map[string]bool{
		"summary": true, "description": true, "issuetype": true,
		"project": true, "reporter": true, "assignee": true,
		"priority": true, "labels": true, "components": true,
		"fixversions": true, "versions": true, "duedate": true,
		"environment": true, "resolution": true, "status": true,
		"created": true, "updated": true, "parent": true,
		"security": true,
	}
	return systemFields[strings.ToLower(name)]
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
// surface the synthetic "Security Level" row. Spaces, hyphens, and underscores
// are stripped from both sides so "securitylevel" (the type this row reports)
// and "security-level" match, not only "security level".
func securityFilterMatches(fieldLower string) bool {
	norm := strings.NewReplacer(" ", "", "-", "", "_", "").Replace(fieldLower)
	return norm != "" && strings.Contains("securitylevel", norm)
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
	} else if !isSystemField(key) {
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
