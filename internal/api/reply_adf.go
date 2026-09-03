package api

import (
	"fmt"
	"net/url"
	"strings"
)

const replySeparator = " · "

// BuildReplyADF represents a flat Jira reply as an author mention and a stable link to the original comment.
func BuildReplyADF(hostname, issueKey, commentID string, author *User, body *ADF) *ADF {
	commentURL := url.URL{
		Scheme: "https",
		Host:   hostname,
		Path:   "/browse/" + issueKey,
	}
	query := commentURL.Query()
	query.Set("focusedCommentId", commentID)
	commentURL.RawQuery = query.Encode()

	header := ADFContent{Type: "paragraph"}
	if author != nil {
		accountID := strings.TrimSpace(author.AccountID)
		if accountID != "" {
			displayName := strings.TrimSpace(author.DisplayName)
			if displayName == "" {
				displayName = "Jira user"
			}
			header.Content = append(header.Content,
				ADFContent{
					Type: "mention",
					Attrs: &ADFAttrs{
						ID:   accountID,
						Text: "@" + displayName,
					},
				},
				ADFContent{Type: "text", Text: replySeparator},
			)
		}
	}
	header.Content = append(header.Content, ADFContent{
		Type: "text",
		Text: fmt.Sprintf("Replying to comment %s", commentID),
		Marks: []ADFMark{{
			Type:  "link",
			Attrs: &ADFAttrs{Href: commentURL.String()},
		}},
	})

	content := []ADFContent{header}
	if body != nil {
		content = append(content, body.Content...)
	}

	return &ADF{Type: "doc", Version: 1, Content: content}
}
