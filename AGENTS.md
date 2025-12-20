# Atlassian CLI (atl) - Agent Reference

This document provides guidance for LLM agents using the `atl` CLI tool.

## Overview

`atl` is a command-line tool for Jira, Confluence, and Tempo. All commands support `--json` for structured output, making it ideal for programmatic use.

## Authentication

```bash
atl auth status                         # Check authentication status
atl auth login                          # Authenticate (opens browser)
```

## Jira Issues

### View Issues

```bash
atl issue view PROJ-1234                  # View issue details
atl issue view PROJ-1234 --json           # View as JSON (for parsing)
atl issue view PROJ-1234 --web            # Open in browser
```

### List Issues

```bash
atl issue list --assignee @me           # Your assigned issues
atl issue list --project NX             # Issues in project
atl issue list --jql "status = Open"    # Custom JQL query
atl issue list --jql "sprint in openSprints() AND assignee = currentUser()"
```

### Create Issues

```bash
atl issue create --project NX --type Bug --summary "Title"
atl issue create --project NX --type Task --summary "Title" --description "Details"
atl issue create --project NX --parent PROJ-123 --summary "Subtask"
```

### Edit Issues

```bash
atl issue edit PROJ-1234 --summary "New summary"
atl issue edit PROJ-1234 --assignee @me
atl issue edit PROJ-1234 --add-label bug --remove-label wontfix
atl issue edit PROJ-1234 --field "Story Points=8"
```

### Transitions and Workflow

```bash
atl issue transition PROJ-1234 "In Progress"
atl issue transition PROJ-1234 --list     # List available transitions
```

### Comments

```bash
atl issue comment PROJ-1234 --body "Comment text"
atl issue comment PROJ-1234 --list        # List comments
```

### Issue Links

```bash
atl issue link PROJ-1234 PROJ-5678                    # Link issues (default: Relates)
atl issue link PROJ-1234 PROJ-5678 --type Blocks      # Link with specific type
```

### Web Links

```bash
atl issue weblink PROJ-1234 --url "https://..." --title "Title"
```

### Sprint Management

```bash
atl issue sprint PROJ-1234 --sprint-id 123          # Move to sprint
atl issue sprint PROJ-1234 --backlog                # Move to backlog
```

### Attachments

```bash
atl issue attachment PROJ-1234 --list               # List attachments
atl issue attachment PROJ-1234 --download <id>      # Download attachment
```

## Jira Boards

```bash
atl board list                                    # List all boards
atl board list --project NX                       # List boards for project
atl board rank PROJ-123 --before PROJ-456             # Rank issue before another
atl board rank PROJ-123 --after PROJ-456              # Rank issue after another
atl board rank PROJ-123 --top --board-id 42         # Move to top of backlog
```

## Confluence

### Spaces

```bash
atl confluence space list               # List spaces
atl confluence space list --json        # List as JSON
```

### Pages

```bash
atl confluence page view <id>           # View page by ID
atl confluence page view --space DOCS --title "Title"
atl confluence page list --space DOCS   # List pages in space
atl confluence page list --space DOCS --status draft     # List draft pages
atl confluence page list --space DOCS --status archived  # List archived pages
atl confluence page search "query"      # Search pages
atl confluence page children <id>       # List child pages
atl confluence page create --space DOCS --title "New Page" --body "<p>Content</p>"
atl confluence page create --space DOCS --title "Draft" --draft   # Create as draft
atl confluence page edit <id> --body "<p>New content</p>"
atl confluence page delete <id>         # Delete page (prompts for confirmation)
atl confluence page delete <id> --force # Delete without confirmation
atl confluence page publish <id>        # Publish a draft page
atl confluence page move <id> --parent <parent-id>
atl confluence page archive <id>        # Archive page
atl confluence page archive <id> --unarchive
```

## Formatting Guidelines

### Jira Wiki Markup

Jira uses its own wiki markup, NOT Markdown. Common syntax:

```
h1. Heading 1
h2. Heading 2

*bold text*
_italic text_
-strikethrough-

* Bullet list
** Nested bullet
# Numbered list

{code:java}
code block
{code}

{noformat}
preformatted text
{noformat}

[Link text|https://example.com]
[PROJ-1234]                           # Auto-links to issue

||Header 1||Header 2||
|Cell 1|Cell 2|

{quote}
Quoted text
{quote}
```

**Important**: Comments (`atl issue comment`) render as plain text only - wiki markup does NOT work in comments. Use wiki markup only in issue descriptions.

### Confluence HTML

Confluence page bodies must be HTML with Confluence macros for code blocks:

```html
<h1>Heading</h1>
<p>Paragraph with <strong>bold</strong> and <code>inline code</code>.</p>
<ul>
  <li>Bullet item</li>
</ul>
<table>
  <tr><th>Header</th></tr>
  <tr><td>Cell</td></tr>
</table>
```

For code blocks, use Confluence macro format:

```html
<ac:structured-macro ac:name="code">
  <ac:plain-text-body><![CDATA[your code here
multi-line code is preserved]]></ac:plain-text-body>
</ac:structured-macro>
```

**Important**: The `--body` flag replaces the ENTIRE page content. View the page first with `atl confluence page view <id>` to understand its structure.

## JSON Output

Use `--json` flag for structured output suitable for parsing:

```bash
# Get issue data as JSON
atl issue view PROJ-1234 --json | jq '.status'

# List issues and extract keys
atl issue list --assignee @me --json | jq '.[].key'

# Get page content
atl confluence page view 12345 --json | jq '.body'
```

## Common Workflows

### Find and Update an Issue

```bash
# Find the issue
atl issue list --jql "summary ~ 'login bug'" --json

# View details
atl issue view PROJ-1234

# Update it
atl issue edit PROJ-1234 --assignee @me
atl issue transition PROJ-1234 "In Progress"
atl issue comment PROJ-1234 --body "Starting work on this"
```

### Create a Linked Issue

```bash
# Create the issue
atl issue create --project NX --type Task --summary "Implement feature X"

# Link it to a parent story
atl issue link PROJ-1235 PROJ-1000 --type "is part of"
```

### Update Confluence Documentation

```bash
# Find the page
atl confluence page search "API documentation" --json

# View current content
atl confluence page view 12345

# Update it
atl confluence page edit 12345 --body "<h1>Updated API Docs</h1><p>New content...</p>"
```

## Error Handling

The CLI returns non-zero exit codes on failure. Common errors:

- **401 Unauthorized**: Run `atl auth login` to re-authenticate
- **403 Forbidden**: Check permissions for the resource
- **404 Not Found**: Verify the issue key, page ID, or space key exists

## Limitations

- `worklog` commands are not yet implemented (Tempo API pending)
- No automatic pagination for large result sets
- Rate limiting may apply for bulk operations

---

# Development Guide

This section is for agents working on the `atl` codebase itself.

## Build Commands

```bash
make build      # Build to ./bin/atl
make install    # Install to $GOPATH/bin
make test       # Run tests
make lint       # Run golangci-lint
make check      # Run all checks (test + lint)
make clean      # Remove build artifacts
```

Quick build and run:
```bash
go build -o bin/atl ./cmd/atl
./bin/atl <command>
```

## Architecture

```
cmd/atl/main.go          # Entry point
internal/
  api/                   # Atlassian API clients
    client.go            # Base HTTP client with OAuth
    jira.go              # Jira API (issues, search, transitions, comments)
    confluence.go        # Confluence API (spaces, pages)
    resources.go         # Accessible resources discovery
  auth/                  # OAuth 2.0 flow and token management
    oauth.go             # OAuth flow with browser callback
    token.go             # Keyring-based token storage
    browser.go           # Browser launcher
  cmd/                   # Cobra command definitions
    root.go              # Root command, subcommand registration
    auth/                # auth login|logout|status
    issue/               # issue view|list|create|edit|transition|comment|assign
    confluence/          # confluence space|page subcommands
    board/               # board list|rank
    worklog/             # worklog add|list|edit|delete (stubs)
    config/              # config get|set|list
  config/                # Configuration management (~/.config/atlassian/)
  iostreams/             # I/O abstraction for testability
  output/                # Output formatting (JSON, tables, colors)
```

## Adding a New Command

1. Create file in appropriate `internal/cmd/<group>/` directory
2. Define `*Options` struct with `IO *iostreams.IOStreams` and flags
3. Create `NewCmd*` function returning `*cobra.Command`
4. Define `*Output` struct for JSON output with proper json tags
5. Implement `run*` function that:
   - Creates API client via `api.NewClientFromConfig()`
   - Calls API methods
   - Outputs via `output.JSON()` or `fmt.Fprintf(opts.IO.Out, ...)`
6. Register in parent command's `NewCmd*` function

### Error Messages for Required Flags

Don't use `cmd.MarkFlagRequired()` - it produces unhelpful errors. Instead, validate in `RunE` with helpful messages:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if opts.Space == "" {
        return fmt.Errorf("--space flag is required\n\nUse 'atl confluence space list' to see available spaces")
    }
    return runList(opts)
},
```

### API Client Usage

```go
client, err := api.NewClientFromConfig()
if err != nil {
    return err
}
ctx := context.Background()
jira := api.NewJiraService(client)
// or
confluence := api.NewConfluenceService(client)
```

### Output Format

All commands must support `--json` flag:

```go
if opts.JSON {
    return output.JSON(opts.IO.Out, outputStruct)
}
// Plain text output
fmt.Fprintf(opts.IO.Out, "Result: %s\n", value)
```

## OAuth Configuration

OAuth 2.0 credentials can be configured two ways:

1. **Interactive setup**: `atl auth setup` - walks through creating OAuth app
2. **Environment variables**: `ATLASSIAN_CLIENT_ID` and `ATLASSIAN_CLIENT_SECRET`

Credentials stored in `~/.config/atlassian/config.yaml`. Tokens stored in system keyring.
