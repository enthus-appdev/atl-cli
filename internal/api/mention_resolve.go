package api

import (
	"context"
	"strings"
)

// MentionResolver resolves a display name to a Jira accountId.
// Returns empty string if the user cannot be found.
type MentionResolver func(ctx context.Context, displayName string) (accountID string, err error)

// ResolveMentions walks an ADF document and resolves unresolved mention nodes.
// Unresolved mentions have an empty ID and a display name in Text (e.g. "@John Doe").
// After resolution, the mention's ID is set to the accountId.
// If the resolver returns empty string, the mention is converted to a plain text node.
func ResolveMentions(ctx context.Context, doc *ADF, resolver MentionResolver) error {
	if doc == nil {
		return nil
	}
	for i := range doc.Content {
		if err := resolveInContent(ctx, &doc.Content[i], resolver); err != nil {
			return err
		}
	}
	return nil
}

func resolveInContent(ctx context.Context, node *ADFContent, resolver MentionResolver) error {
	for i := range node.Content {
		child := &node.Content[i]

		if child.Type == "mention" && child.Attrs != nil && child.Attrs.ID == "" {
			displayName := strings.TrimPrefix(child.Attrs.Text, "@")

			accountID, err := resolver(ctx, displayName)
			if err != nil {
				return err
			}

			if accountID != "" {
				child.Attrs.ID = accountID
			} else {
				// Could not resolve - convert to plain text
				child.Type = "text"
				child.Text = child.Attrs.Text
				child.Attrs = nil
			}
			continue
		}

		if err := resolveInContent(ctx, child, resolver); err != nil {
			return err
		}
	}
	return nil
}
