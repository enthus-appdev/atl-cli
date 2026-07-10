# Atlassian CLI (atl) - Agent Reference

This document provides guidance for LLM agents using the `atl` CLI tool.

## Overview

`atl` is a command-line tool for Jira and Confluence. All commands support `--json` for structured output, making it ideal for programmatic use.

Jira commands live under `atl jira` (`atl jira issue`, `atl jira board`, `atl jira sm`). The old top-level forms (`atl issue`, `atl board`, `atl sm`) still work as deprecated aliases but print a warning and will be removed in a future release.

## Authentication

```bash
atl auth status                         # Check authentication status
atl auth login                          # Authenticate (opens browser)
```

## Context Switching (Multi-Environment)

Switch between Atlassian instances using aliases:

```bash
# Create aliases
atl config set-alias prod                                    # alias "prod" → current host
atl config set-alias sandbox mycompany-sandbox.atlassian.net # alias → specific host

# Switch active host
atl config use-context prod       # by alias
atl config use-context mycompany.atlassian.net  # by hostname

# Show current context
atl config current-context        # prints alias + hostname
atl config current-context --json # JSON output

# Remove alias
atl config delete-alias sandbox

# View aliases
atl config list                   # shows Aliases section with (current) marker
```

Aliases also work with `--hostname` flags: `atl auth status --hostname prod`

## Jira Issues

### View Issues

```bash
atl jira issue view PROJ-1234                  # View issue details (includes custom fields)
atl jira issue view PROJ-1234 --json           # View as JSON (includes custom_fields section)
atl jira issue view PROJ-1234 --web            # Open in browser
```

### List Issues

```bash
atl jira issue list --assignee @me           # Your assigned issues
atl jira issue list --project PROJ             # Issues in project
atl jira issue list --jql "status = Open"    # Custom JQL query
atl jira issue list --jql "sprint in openSprints() AND assignee = currentUser()"
```

### Create Issues

```bash
atl jira issue create --project PROJ --type Bug --summary "Title"
atl jira issue create --project PROJ --type Task --summary "Title" --description "Details"
atl jira issue create --project PROJ --parent PROJ-123 --summary "Subtask"
```

### Edit Issues

```bash
atl jira issue edit PROJ-1234 --summary "New summary"
atl jira issue edit PROJ-1234 --description "New description content"
atl jira issue edit PROJ-1234 --description "Additional notes" --append  # Append to existing
atl jira issue edit PROJ-1234 --assignee @me
atl jira issue edit PROJ-1234 --add-label bug --remove-label wontfix
atl jira issue edit PROJ-1234 --field "Story Points=8"
atl jira issue edit PROJ-1234 --field "Custom Field=Some **markdown** text"  # Auto-converts to ADF
```

**Notes**:
- `--append` preserves existing description content (including embedded media) and adds new content at the end
- Textarea custom fields automatically convert Markdown to ADF format

### Transitions and Workflow

```bash
atl jira issue transition PROJ-1234 "In Progress"
atl jira issue transition PROJ-1234 --list     # List available transitions
atl jira issue transition PROJ-1234 "Done" --field "Resolution=Fixed"  # With required fields
```

### Comments

```bash
atl jira issue comment list PROJ-1234                    # List comments
atl jira issue comment add PROJ-1234 --body "Comment"    # Add comment
atl jira issue comment add PROJ-1234 --body-file msg.md  # Add from file (avoids shell escaping)
atl jira issue comment edit PROJ-1234 --id 123 --body "Updated"
atl jira issue comment edit PROJ-1234 --id 123 --body-file msg.md
atl jira issue comment delete PROJ-1234 --id 123
```

### Issue Links

```bash
atl jira issue link PROJ-1234 PROJ-5678                    # Link issues (default: Relates)
atl jira issue link PROJ-1234 PROJ-5678 --type Blocks      # Link with specific type
```

### Web Links

```bash
atl jira issue weblink PROJ-1234 --url "https://..." --title "Title"
```

### Sprint Management

```bash
atl jira issue sprint PROJ-1234 --sprint-id 123          # Move to sprint
atl jira issue sprint PROJ-1234 --backlog                # Move to backlog
```

### Attachments

```bash
atl jira issue attachment PROJ-1234 --list               # List attachments
atl jira issue attachment PROJ-1234 --download <id>      # Download attachment
```

### Metadata Discovery

```bash
atl jira issue types --project PROJ                      # List issue types
atl jira issue priorities                                # List available priorities
atl jira issue fields                                    # List all fields
atl jira issue fields --search "story points"            # Search for field by name
atl jira issue field-options --project PROJ --type Bug   # Show allowed values for fields
atl jira issue field-options --project PROJ --type Bug --field "Priority"  # Specific field
```

## Jira Boards

```bash
atl jira board list                                    # List all boards
atl jira board list --project PROJ                       # List boards for project
atl jira board rank PROJ-123 --before PROJ-456             # Rank issue before another
atl jira board rank PROJ-123 --after PROJ-456              # Rank issue after another
atl jira board rank PROJ-123 --top --board-id 42         # Move to top of backlog
```

## Jira Sprints

Full sprint lifecycle under `atl jira sprint`.

**Dates and duration (`start`, and `create --start`):**
- `create` without `--start` makes a future sprint — undated unless you explicitly pass `--start-date`/`--end-date`. With `--start` it is also activated.
- Start date defaults to now; end date is `start + --duration` (default `14d`). Explicit `--start-date`/`--end-date` (ISO `YYYY-MM-DD`) override.
- `--duration` accepts `Nd` (days) and `Nw` (weeks); `48h`-style Go durations also parse but days/weeks are clearer.
- Explicit `--start-date`/`--end-date` are interpreted as midnight UTC. If you need an exact calendar day in a west-of-UTC timezone, verify the result in Jira and adjust by a day if necessary.
- Editing dates on an active/closed sprint may be rejected by Jira. Sprint deletion is not supported (close instead).

**Moving issues:** prefer `--to <sprint-id>` for an exact target. `--sprint "<name>"` resolves by name (exact, else substring — same matching as `atl jira issue sprint`), so it can be ambiguous when names overlap; use `atl jira sprint list --board <id>` to find the ID.

```bash
# Create a sprint as undated future sprint (required: --board, --name)
atl jira sprint create --board 42 --name "Sprint 30" --goal "Ship MI cutover"

# Create and start immediately (--start applies default 14d duration; override with --duration)
atl jira sprint create --board 42 --name "Sprint 31" --start
atl jira sprint create --board 42 --name "Sprint 32" --start --duration 3w

# Edit sprint name, goal, or dates (date changes may fail for active/closed sprints)
# Date format: YYYY-MM-DD
atl jira sprint edit 123 --goal "Updated goal"
atl jira sprint edit 123 --name "Sprint 30 (extended)" --start-date 2026-06-23 --end-date 2026-07-14

# Start a sprint (only future sprints; sets dates via duration OR explicit dates)
# Duration-based (calculates end date; default 14d)
atl jira sprint start 123
atl jira sprint start 123 --duration 2w
# Or use explicit dates (YYYY-MM-DD)
atl jira sprint start 123 --start-date 2026-06-23 --end-date 2026-07-07

# Close a sprint (only active sprints; incomplete issues move to backlog; prompts unless --force)
atl jira sprint close 123
atl jira sprint close 123 --force

# List sprints on a board (required: --board; state: active|future|closed, default: active,future)
atl jira sprint list --board 42
atl jira sprint list --board 42 --state closed
atl jira sprint list --board 42 --state active,future,closed

# Move issues by sprint ID (RECOMMENDED: unambiguous, safe)
# Note: Issues must belong to target sprint's board; Jira will error if they don't
atl jira sprint move NX-1 NX-2 --to 123

# Move issues to their native board backlogs (works cross-board; independent of sprint)
atl jira sprint backlog NX-1 NX-2
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
atl confluence page move <id> --target <parent-id>
atl confluence page archive <id>        # Archive page (unarchive not supported via API)
```

### Templates

```bash
atl confluence template view <id>       # View template
atl confluence template create --space DOCS --name "Name" --body "<html>"
atl confluence template update <id> --body "<html>"
```

### Attachments

```bash
atl confluence page attachment <id> --list               # List attachments
atl confluence page attachment <id> --list --json        # List as JSON
atl confluence page attachment <id> --download --id <attID>  # Download specific
atl confluence page attachment <id> --download-all       # Download all
atl confluence page attachment <id> --download-all -o ./dir  # Download to directory
atl confluence page attachment <id> --upload ./file.pdf  # Upload file
atl confluence page attachment <id> --upload a.pdf --upload b.png  # Upload multiple
```

## Formatting Guidelines

### Jira Formatting (Markdown to ADF)

The atl CLI accepts Markdown for descriptions AND comments, converting to Atlassian Document Format (ADF):

- Standard Markdown: headings, bold, italic, strikethrough, code, lists, links
- Blockquotes: `> text`
- Horizontal rules: `---` or `***` or `___`
- GFM tables: `| Header | Header |` with `|---|---|` separator
- Panels: `:::info`, `:::warning`, `:::error`, `:::note`, `:::success`
- Expandable sections: `+++Title\ncontent\n+++`
- Media references: `!media[attachment-id]`
- Mentions: `@[Display Name]` or `@[id:accountId]`

**Tip**: Use `--body-file` for multi-line content to avoid shell escaping issues with backticks and special characters.

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
atl jira issue view PROJ-1234 --json | jq '.status'

# List issues and extract keys
atl jira issue list --assignee @me --json | jq '.[].key'

# Get page content
atl confluence page view 12345 --json | jq '.body'
```

## Common Workflows

### Find and Update an Issue

```bash
# Find the issue
atl jira issue list --jql "summary ~ 'login bug'" --json

# View details
atl jira issue view PROJ-1234

# Update it
atl jira issue edit PROJ-1234 --assignee @me
atl jira issue transition PROJ-1234 "In Progress"
atl jira issue comment PROJ-1234 --body "Starting work on this"
```

### Create a Linked Issue

```bash
# Create the issue
atl jira issue create --project PROJ --type Task --summary "Implement feature X"

# Link it to a parent story
atl jira issue link PROJ-1235 PROJ-1000 --type "is part of"
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
    config/              # config get|set|list|use-context|current-context|set-alias|delete-alias
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

`atl auth login`/`refresh` and the API client's auto-refresh resolve the OAuth
app credentials as a **complete pair from the first layer that supplies both**
halves — never mixed across layers (see `auth.ResolveClientCredentials`):

1. **Environment variables**: `ATLASSIAN_CLIENT_ID` **and** `ATLASSIAN_CLIENT_SECRET` (both required, else this layer is skipped)
2. **OS keychain**: `atl auth set-credentials [--client-id X] [--client-secret Y | --from-stdin]` (recommended; `--delete` removes)
3. **Config file** `~/.config/atlassian/config.yaml`: written by `atl auth setup` (interactive own-app wizard)

Client credentials in the keychain live under service `atlassian-cli-oauth` as a
single JSON entry (`internal/auth/credentials.go`). Per-user access/refresh **tokens** are stored
as files under `~/.config/atlassian/tokens/` (not the keychain — token blobs can
exceed keychain size limits).
