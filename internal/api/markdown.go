package api

import (
	"regexp"
	"strings"
)

// MarkdownToADF converts markdown text to Atlassian Document Format.
// Supports:
//   - Headings: # h1, ## h2, etc.
//   - Bold: **text** or __text__
//   - Italic: *text* or _text_
//   - Strikethrough: ~~text~~
//   - Inline code: `code`
//   - Code blocks: ```language\ncode\n```
//   - Links: [text](url)
//   - Bullet lists: - item or * item
//   - Numbered lists: 1. item
//   - Blockquotes: > text
//   - Horizontal rules: --- or *** or ___
//   - Tables: | col | col | (GFM-style)
//   - Panels: :::info, :::warning, :::error, :::note, :::success
//   - Expand: +++Title\ncontent\n+++
//   - Media: !media[id] or !media[collection:id]
func MarkdownToADF(text string) *ADF {
	if text == "" {
		return &ADF{
			Type:    "doc",
			Version: 1,
			Content: []ADFContent{},
		}
	}

	lines := strings.Split(text, "\n")
	content := parseBlocks(lines)

	return &ADF{
		Type:    "doc",
		Version: 1,
		Content: content,
	}
}

// parseBlocks parses block-level markdown elements.
func parseBlocks(lines []string) []ADFContent {
	var content []ADFContent
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// Fenced code block
		if strings.HasPrefix(line, "```") {
			block, consumed := parseCodeBlock(lines, i)
			content = append(content, block)
			i += consumed
			continue
		}

		// Panel (:::type ... :::)
		if strings.HasPrefix(strings.TrimSpace(line), ":::") {
			block, consumed := parsePanel(lines, i)
			if block.Type != "" {
				content = append(content, block)
				i += consumed
				continue
			}
		}

		// Expand (+++title ... +++)
		if strings.HasPrefix(strings.TrimSpace(line), "+++") {
			block, consumed := parseExpand(lines, i)
			if block.Type != "" {
				content = append(content, block)
				i += consumed
				continue
			}
		}

		// Table (| col | col |)
		if isTableRow(line) {
			block, consumed := parseTable(lines, i)
			if block.Type != "" {
				content = append(content, block)
				i += consumed
				continue
			}
		}

		// Heading
		if heading, ok := parseHeading(line); ok {
			content = append(content, heading)
			i++
			continue
		}

		// Horizontal rule
		if isHorizontalRule(line) {
			content = append(content, ADFContent{Type: "rule"})
			i++
			continue
		}

		// Blockquote
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			block, consumed := parseBlockquote(lines, i)
			content = append(content, block)
			i += consumed
			continue
		}

		// Bullet list
		if isBulletListItem(line) {
			block, consumed := parseBulletList(lines, i)
			content = append(content, block)
			i += consumed
			continue
		}

		// Ordered list
		if isOrderedListItem(line) {
			block, consumed := parseOrderedList(lines, i)
			content = append(content, block)
			i += consumed
			continue
		}

		// Default: paragraph
		para, consumed := parseParagraph(lines, i)
		content = append(content, para)
		i += consumed
	}

	return content
}

// parseCodeBlock parses a fenced code block (```).
func parseCodeBlock(lines []string, start int) (ADFContent, int) {
	// Extract optional language from opening fence
	openingLine := lines[start]
	lang := strings.TrimPrefix(strings.TrimSpace(openingLine), "```")

	var codeLines []string
	i := start + 1
	for i < len(lines) {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			i++ // consume closing fence
			break
		}
		codeLines = append(codeLines, lines[i])
		i++
	}

	codeText := strings.Join(codeLines, "\n")

	block := ADFContent{
		Type: "codeBlock",
		Content: []ADFContent{
			{Type: "text", Text: codeText},
		},
	}

	if lang != "" {
		block.Attrs = &ADFAttrs{Language: lang}
	}

	return block, i - start
}

// parseHeading parses a markdown heading (# to ######).
func parseHeading(line string) (ADFContent, bool) {
	trimmed := strings.TrimSpace(line)

	// Count leading # characters
	level := 0
	for _, c := range trimmed {
		if c == '#' {
			level++
		} else {
			break
		}
	}

	if level == 0 || level > 6 {
		return ADFContent{}, false
	}

	// Must have space after # (or be just ##...#)
	rest := strings.TrimPrefix(trimmed, strings.Repeat("#", level))
	if len(rest) > 0 && rest[0] != ' ' {
		return ADFContent{}, false
	}

	text := strings.TrimSpace(rest)

	return ADFContent{
		Type:    "heading",
		Attrs:   &ADFAttrs{Level: level},
		Content: parseInline(text),
	}, true
}

// isHorizontalRule checks if a line is a horizontal rule.
func isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Must be at least 3 characters
	if len(trimmed) < 3 {
		return false
	}

	// Check for ---, ***, or ___
	char := trimmed[0]
	if char != '-' && char != '*' && char != '_' {
		return false
	}

	for _, c := range trimmed {
		if c != rune(char) && c != ' ' {
			return false
		}
	}

	// Count the actual rule characters
	count := 0
	for _, c := range trimmed {
		if c == rune(char) {
			count++
		}
	}

	return count >= 3
}

// parseBlockquote parses a blockquote (>).
func parseBlockquote(lines []string, start int) (ADFContent, int) {
	var quoteLines []string
	i := start

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, ">") {
			// Check if it's a continuation (non-empty, indented)
			if trimmed == "" {
				break
			}
			break
		}

		// Remove the > prefix
		content := strings.TrimPrefix(trimmed, ">")
		content = strings.TrimPrefix(content, " ") // Remove optional space after >
		quoteLines = append(quoteLines, content)
		i++
	}

	// Parse the blockquote content recursively
	innerContent := parseBlocks(quoteLines)

	return ADFContent{
		Type:    "blockquote",
		Content: innerContent,
	}, i - start
}

// isBulletListItem checks if a line is a bullet list item.
func isBulletListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "- ") ||
		strings.HasPrefix(trimmed, "* ") ||
		strings.HasPrefix(trimmed, "+ ")
}

// parseBulletList parses a bullet list.
func parseBulletList(lines []string, start int) (ADFContent, int) {
	var items []ADFContent
	i := start
	baseIndent := countLeadingSpaces(lines[start])

	for i < len(lines) {
		line := lines[i]

		// Empty line might end the list
		if strings.TrimSpace(line) == "" {
			// Check if next non-empty line continues the list
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
				j++
			}
			if j >= len(lines) || !isBulletListItem(lines[j]) {
				break
			}
			i = j
			continue
		}

		indent := countLeadingSpaces(line)

		// If less indented than base, we're done
		if indent < baseIndent && i > start {
			break
		}

		if !isBulletListItem(line) {
			break
		}

		// Extract item text (remove bullet marker)
		trimmed := strings.TrimSpace(line)
		var text string
		if strings.HasPrefix(trimmed, "- ") {
			text = strings.TrimPrefix(trimmed, "- ")
		} else if strings.HasPrefix(trimmed, "* ") {
			text = strings.TrimPrefix(trimmed, "* ")
		} else if strings.HasPrefix(trimmed, "+ ") {
			text = strings.TrimPrefix(trimmed, "+ ")
		}

		items = append(items, ADFContent{
			Type: "listItem",
			Content: []ADFContent{
				{
					Type:    "paragraph",
					Content: parseInline(text),
				},
			},
		})
		i++
	}

	return ADFContent{
		Type:    "bulletList",
		Content: items,
	}, i - start
}

// isOrderedListItem checks if a line is an ordered list item.
func isOrderedListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Match patterns like "1. ", "12. ", etc.
	orderedPattern := regexp.MustCompile(`^\d+\.\s`)
	return orderedPattern.MatchString(trimmed)
}

// parseOrderedList parses an ordered list.
func parseOrderedList(lines []string, start int) (ADFContent, int) {
	var items []ADFContent
	i := start
	baseIndent := countLeadingSpaces(lines[start])
	orderedPattern := regexp.MustCompile(`^\d+\.\s*(.*)`)

	for i < len(lines) {
		line := lines[i]

		// Empty line might end the list
		if strings.TrimSpace(line) == "" {
			// Check if next non-empty line continues the list
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
				j++
			}
			if j >= len(lines) || !isOrderedListItem(lines[j]) {
				break
			}
			i = j
			continue
		}

		indent := countLeadingSpaces(line)

		// If less indented than base, we're done
		if indent < baseIndent && i > start {
			break
		}

		if !isOrderedListItem(line) {
			break
		}

		// Extract item text
		trimmed := strings.TrimSpace(line)
		matches := orderedPattern.FindStringSubmatch(trimmed)
		if len(matches) < 2 {
			break
		}
		text := matches[1]

		items = append(items, ADFContent{
			Type: "listItem",
			Content: []ADFContent{
				{
					Type:    "paragraph",
					Content: parseInline(text),
				},
			},
		})
		i++
	}

	return ADFContent{
		Type:    "orderedList",
		Content: items,
	}, i - start
}

// parseParagraph parses a paragraph (consecutive non-empty lines).
func parseParagraph(lines []string, start int) (ADFContent, int) {
	var paraLines []string
	i := start

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Empty line ends paragraph
		if trimmed == "" {
			break
		}

		// Special block elements end the paragraph
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "```") ||
			strings.HasPrefix(trimmed, ">") ||
			isBulletListItem(line) ||
			isOrderedListItem(line) ||
			isHorizontalRule(line) {
			break
		}

		paraLines = append(paraLines, trimmed)
		i++
	}

	text := strings.Join(paraLines, " ")

	return ADFContent{
		Type:    "paragraph",
		Content: parseInline(text),
	}, i - start
}

// countLeadingSpaces counts the number of leading spaces/tabs.
func countLeadingSpaces(line string) int {
	count := 0
	for _, c := range line {
		switch c {
		case ' ':
			count++
		case '\t':
			count += 4 // Treat tabs as 4 spaces
		default:
			return count
		}
	}
	return count
}

// hasCodeMark returns true if the content node has an inline code mark.
// In ADF, the code mark is exclusive and cannot be combined with other marks
// like strong, em, or strike. Jira will reject the document with INVALID_INPUT.
func hasCodeMark(c ADFContent) bool {
	for _, m := range c.Marks {
		if m.Type == "code" {
			return true
		}
	}
	return false
}

// addMarkToContent prepends a mark to inner content nodes, skipping nodes that
// already have a code mark (since code is exclusive in ADF).
func addMarkToContent(innerContent []ADFContent, mark ADFMark) []ADFContent {
	result := make([]ADFContent, 0, len(innerContent))
	for _, c := range innerContent {
		if !hasCodeMark(c) {
			c.Marks = append([]ADFMark{mark}, c.Marks...)
		}
		result = append(result, c)
	}
	return result
}

// parseInline parses inline markdown elements (bold, italic, code, links).
func parseInline(text string) []ADFContent {
	if text == "" {
		return nil
	}

	var content []ADFContent
	remaining := text

	for len(remaining) > 0 {
		// Try to match each inline pattern
		matched := false

		// Inline code: `code`
		if codeMatch := regexp.MustCompile("^`([^`]+)`").FindStringSubmatch(remaining); len(codeMatch) > 0 {
			content = append(content, ADFContent{
				Type: "text",
				Text: codeMatch[1],
				Marks: []ADFMark{
					{Type: "code"},
				},
			})
			remaining = remaining[len(codeMatch[0]):]
			matched = true
			continue
		}

		// Link: [text](url)
		if linkMatch := regexp.MustCompile(`^\[([^\]]+)\]\(([^)]+)\)`).FindStringSubmatch(remaining); len(linkMatch) > 0 {
			content = append(content, ADFContent{
				Type: "text",
				Text: linkMatch[1],
				Marks: []ADFMark{
					{Type: "link", Attrs: &ADFAttrs{Href: linkMatch[2]}},
				},
			})
			remaining = remaining[len(linkMatch[0]):]
			matched = true
			continue
		}

		// Media reference: !media[id] or !media[collection:id]
		if mediaMatch := regexp.MustCompile(`^!media\[([^\]]+)\]`).FindStringSubmatch(remaining); len(mediaMatch) > 0 {
			// Media is a block element, but we handle it inline for convenience
			// It will be rendered as [attachment] or similar by the display
			mediaContent := parseMediaContent(mediaMatch[1])
			content = append(content, mediaContent)
			remaining = remaining[len(mediaMatch[0]):]
			matched = true
			continue
		}

		// Bold: **text** or __text__
		if boldMatch := regexp.MustCompile(`^\*\*([^*]+)\*\*`).FindStringSubmatch(remaining); len(boldMatch) > 0 {
			// Parse inner content for nested formatting
			innerContent := parseInline(boldMatch[1])
			content = append(content, addMarkToContent(innerContent, ADFMark{Type: "strong"})...)
			remaining = remaining[len(boldMatch[0]):]
			matched = true
			continue
		}
		if boldMatch := regexp.MustCompile(`^__([^_]+)__`).FindStringSubmatch(remaining); len(boldMatch) > 0 {
			innerContent := parseInline(boldMatch[1])
			content = append(content, addMarkToContent(innerContent, ADFMark{Type: "strong"})...)
			remaining = remaining[len(boldMatch[0]):]
			matched = true
			continue
		}

		// Strikethrough: ~~text~~
		if strikeMatch := regexp.MustCompile(`^~~([^~]+)~~`).FindStringSubmatch(remaining); len(strikeMatch) > 0 {
			innerContent := parseInline(strikeMatch[1])
			content = append(content, addMarkToContent(innerContent, ADFMark{Type: "strike"})...)
			remaining = remaining[len(strikeMatch[0]):]
			matched = true
			continue
		}

		// Italic: *text* or _text_ (must not be followed by another * or _)
		if italicMatch := regexp.MustCompile(`^\*([^*]+)\*`).FindStringSubmatch(remaining); len(italicMatch) > 0 {
			innerContent := parseInline(italicMatch[1])
			content = append(content, addMarkToContent(innerContent, ADFMark{Type: "em"})...)
			remaining = remaining[len(italicMatch[0]):]
			matched = true
			continue
		}
		if italicMatch := regexp.MustCompile(`^_([^_]+)_`).FindStringSubmatch(remaining); len(italicMatch) > 0 {
			innerContent := parseInline(italicMatch[1])
			content = append(content, addMarkToContent(innerContent, ADFMark{Type: "em"})...)
			remaining = remaining[len(italicMatch[0]):]
			matched = true
			continue
		}

		// No pattern matched - consume plain text until next potential pattern
		if !matched {
			// Find the next potential pattern start
			nextPatternIdx := len(remaining)
			patterns := []string{"`", "[", "*", "_", "~", "!"}
			for _, p := range patterns {
				if idx := strings.Index(remaining[1:], p); idx >= 0 && idx+1 < nextPatternIdx {
					nextPatternIdx = idx + 1
				}
			}

			// Add plain text
			plainText := remaining[:nextPatternIdx]
			if len(content) > 0 && len(content[len(content)-1].Marks) == 0 {
				// Merge with previous plain text
				content[len(content)-1].Text += plainText
			} else {
				content = append(content, ADFContent{
					Type: "text",
					Text: plainText,
				})
			}
			remaining = remaining[nextPatternIdx:]
		}
	}

	return content
}

// isTableRow checks if a line looks like a table row.
func isTableRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|")
}

// isTableSeparator checks if a line is a table separator (|---|---|).
func isTableSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") {
		return false
	}
	// Check if it contains only |, -, :, and spaces
	for _, c := range trimmed {
		if c != '|' && c != '-' && c != ':' && c != ' ' {
			return false
		}
	}
	// Must have at least one -
	return strings.Contains(trimmed, "-")
}

// parseTable parses a GFM-style table.
func parseTable(lines []string, start int) (ADFContent, int) {
	if start >= len(lines) {
		return ADFContent{}, 0
	}

	// First line should be header row
	if !isTableRow(lines[start]) {
		return ADFContent{}, 0
	}

	// Second line should be separator
	if start+1 >= len(lines) || !isTableSeparator(lines[start+1]) {
		return ADFContent{}, 0
	}

	// Parse header row
	headerCells := parseTableCells(lines[start])
	if len(headerCells) == 0 {
		return ADFContent{}, 0
	}

	// Create header row with tableHeader cells
	headerRow := ADFContent{
		Type:    "tableRow",
		Content: make([]ADFContent, 0, len(headerCells)),
	}
	for _, cell := range headerCells {
		headerRow.Content = append(headerRow.Content, ADFContent{
			Type: "tableHeader",
			Content: []ADFContent{
				{
					Type:    "paragraph",
					Content: parseInline(cell),
				},
			},
		})
	}

	rows := []ADFContent{headerRow}
	i := start + 2 // Skip header and separator

	// Parse data rows
	for i < len(lines) && isTableRow(lines[i]) {
		cells := parseTableCells(lines[i])
		row := ADFContent{
			Type:    "tableRow",
			Content: make([]ADFContent, 0, len(cells)),
		}
		for _, cell := range cells {
			row.Content = append(row.Content, ADFContent{
				Type: "tableCell",
				Content: []ADFContent{
					{
						Type:    "paragraph",
						Content: parseInline(cell),
					},
				},
			})
		}
		rows = append(rows, row)
		i++
	}

	return ADFContent{
		Type:    "table",
		Content: rows,
	}, i - start
}

// parseTableCells extracts cells from a table row.
func parseTableCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	// Remove leading and trailing pipes
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")

	// Split by pipe
	parts := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// parsePanel parses a panel block (:::type ... :::).
// Supported types: info, note, warning, error, success
func parsePanel(lines []string, start int) (ADFContent, int) {
	if start >= len(lines) {
		return ADFContent{}, 0
	}

	line := strings.TrimSpace(lines[start])
	if !strings.HasPrefix(line, ":::") {
		return ADFContent{}, 0
	}

	// Extract panel type
	panelType := strings.TrimPrefix(line, ":::")
	panelType = strings.TrimSpace(panelType)

	// Validate panel type
	validTypes := map[string]bool{
		"info": true, "note": true, "warning": true,
		"error": true, "success": true,
	}
	if !validTypes[panelType] {
		return ADFContent{}, 0
	}

	// Find closing :::
	var contentLines []string
	i := start + 1
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == ":::" {
			i++ // consume closing
			break
		}
		contentLines = append(contentLines, lines[i])
		i++
	}

	// If no closing found, not a valid panel
	if i == len(lines) && strings.TrimSpace(lines[i-1]) != ":::" {
		return ADFContent{}, 0
	}

	// Parse content inside panel
	innerContent := parseBlocks(contentLines)

	return ADFContent{
		Type:    "panel",
		Attrs:   &ADFAttrs{PanelType: panelType},
		Content: innerContent,
	}, i - start
}

// parseExpand parses an expand/collapsible block (+++title ... +++).
func parseExpand(lines []string, start int) (ADFContent, int) {
	if start >= len(lines) {
		return ADFContent{}, 0
	}

	line := strings.TrimSpace(lines[start])
	if !strings.HasPrefix(line, "+++") {
		return ADFContent{}, 0
	}

	// Extract title (everything after +++)
	title := strings.TrimPrefix(line, "+++")
	title = strings.TrimSpace(title)

	// Find closing +++
	var contentLines []string
	i := start + 1
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "+++" {
			i++ // consume closing
			break
		}
		contentLines = append(contentLines, lines[i])
		i++
	}

	// If no closing found, not a valid expand
	if i == len(lines) && strings.TrimSpace(lines[i-1]) != "+++" {
		return ADFContent{}, 0
	}

	// Parse content inside expand
	innerContent := parseBlocks(contentLines)

	attrs := &ADFAttrs{}
	if title != "" {
		attrs.Title = title
	}

	return ADFContent{
		Type:    "expand",
		Attrs:   attrs,
		Content: innerContent,
	}, i - start
}

// parseMediaReference parses !media[...] syntax in inline content.
// This is handled in parseInline, but we define the helper here.
// Format: !media[id] or !media[collection:id]
func parseMediaContent(ref string) ADFContent {
	// Parse the reference: could be "id" or "collection:id"
	parts := strings.SplitN(ref, ":", 2)

	attrs := &ADFAttrs{
		Type: "file",
	}

	if len(parts) == 2 {
		attrs.Collection = parts[0]
		attrs.ID = parts[1]
	} else {
		attrs.ID = parts[0]
	}

	return ADFContent{
		Type: "mediaSingle",
		Content: []ADFContent{
			{
				Type:  "media",
				Attrs: attrs,
			},
		},
	}
}
