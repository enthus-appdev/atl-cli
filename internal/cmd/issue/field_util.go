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
	}
	return systemFields[strings.ToLower(name)]
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
			return "", nil, fmt.Errorf("field not found: %s\n\nUse 'atl issue fields --search \"%s\"' to find available fields", key, key)
		}
		key = resolvedField.ID
	}

	fieldValue := coerceFieldValue(resolvedField, value)
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
			return api.TextToADF(value)
		}
		if field.Schema.Type == "array" && field.Schema.Custom == "" {
			// Labels-type array of strings.
			vals := strings.Split(value, ",")
			for i := range vals {
				vals[i] = strings.TrimSpace(vals[i])
			}
			return vals
		}
	}

	if numVal, err := strconv.ParseFloat(value, 64); err == nil {
		return numVal
	}
	return value
}
