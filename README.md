# Atlassian CLI (atl)

A command-line tool for working with Jira and Confluence. Designed with LLM-friendly output for easy integration with AI assistants.

## Installation

### Quick Install (Recommended)

```bash
go install github.com/enthus-appdev/atl-cli/cmd/atl@latest
```

Make sure `$GOPATH/bin` (or `$HOME/go/bin`) is in your `PATH`.

### From Source

```bash
gh repo clone enthus-appdev/atl-cli
cd atl-cli
make install
```

## Quick Start

```bash
# 1. Set up OAuth (one-time, interactive wizard)
atl auth setup

# 2. Log in and name the host used by the examples below
atl auth login --hostname mycompany.atlassian.net
atl config set-alias prod mycompany.atlassian.net

# 3. Start using the CLI
atl --context prod jira issue list --assignee @me
atl --context prod confluence space list
```

## OAuth Setup

`atl auth login` needs OAuth app credentials (a client ID and secret). The
client ID and secret are a coupled pair, so they are taken **together from the
first layer that provides both** — the halves are never mixed across layers:

1. **Environment variables** — `ATLASSIAN_CLIENT_ID` **and** `ATLASSIAN_CLIENT_SECRET` (both must be set; handy for CI)
2. **OS keychain** — stored via `atl auth set-credentials` (recommended, especially for a shared app)
3. **Config file** — `~/.config/atlassian/config.yaml`, written by `atl auth setup`

If a layer supplies only one half (e.g. `ATLASSIAN_CLIENT_ID` alone), it is
skipped and the next layer is tried.

### Option A — store credentials in the OS keychain (recommended)

Keeps the secret out of plaintext config. Works with a shared/organization app
or your own. The secret can be piped in so it never hits your shell history:

```bash
atl auth set-credentials --client-id YOUR_ID --client-secret YOUR_SECRET
# or pipe the secret (e.g. straight from a secret manager)
printf '%s' "$SECRET" | atl auth set-credentials --client-id YOUR_ID --from-stdin
```

Remove them with `atl auth set-credentials --delete`.

### Option B — create your own OAuth app

`atl auth setup` guides you through it:

1. Opens https://developer.atlassian.com/console/myapps/
2. Walks you through creating an OAuth 2.0 integration
3. Helps you configure the callback URL: `http://localhost:8085/callback`
4. Stores your Client ID and Secret in `~/.config/atlassian/config.yaml`

### Option C — environment variables (useful for CI/CD)

```bash
export ATLASSIAN_CLIENT_ID="your-client-id"
export ATLASSIAN_CLIENT_SECRET="your-client-secret"
```

## Usage Examples

```bash
# View an issue
atl --context prod jira issue view PROJ-1234

# List your assigned issues
atl --context prod jira issue list --assignee @me

# Output as JSON for LLM processing
atl --context prod jira issue view PROJ-1234 --json

# View a Confluence page
atl --context prod confluence page view --space DOCS --title "Getting Started"
```

## LLM-Friendly Output

All commands support `--json` flag for structured JSON output, making it easy to parse and process with LLMs:

```bash
# Get issue data as JSON
atl --context prod jira issue view PROJ-1234 --json

# List issues as JSON
atl --context prod jira issue list --project PROJ --json

# Get spaces as JSON
atl --context prod confluence space list --json
```

Plain text output is also structured for easy parsing by LLMs.

## Markdown Formatting

Issue descriptions and comments support **Markdown syntax**, which is automatically converted to Jira's Atlassian Document Format (ADF):

```bash
# Create issue with markdown description
atl --context prod jira issue create --project PROJ --type Task --summary "Feature" --description "## Goals

- Goal 1
- Goal 2

**Important**: See [docs](https://example.com) for details."

# Add comment with markdown
atl --context prod jira issue comment add PROJ-1234 --body "## Summary

Fixed the **critical** bug in \`main.go\`.

\`\`\`go
func main() {
    fmt.Println(\"Hello\")
}
\`\`\`"
```

### Supported Markdown

| Syntax | Example |
|--------|---------|
| Headings | `# H1` through `###### H6` |
| Bold | `**bold**` or `__bold__` |
| Italic | `*italic*` or `_italic_` |
| Strikethrough | `~~deleted~~` |
| Inline code | `` `code` `` |
| Code blocks | ` ``` ` with optional language |
| Links | `[text](url)` (Jira wiki `[text|url]` is not supported) |
| Bullet lists | `- item` or `* item` |
| Numbered lists | `1. item` |
| Blockquotes | `> quote` |
| Horizontal rules | `---` or `***` |

## Commands

> Jira commands live under `atl jira` (e.g. `atl jira issue`, `atl jira board`, `atl jira sm`).
> The old top-level forms (`atl issue`, `atl board`, `atl sm`) still work as deprecated aliases
> but print a warning and will be removed in a future release.

### Authentication

```bash
atl auth login        # Authenticate with Atlassian
atl auth logout       # Remove authentication
atl auth status       # View authentication status
```

### Jira Issues

```bash
atl --context prod jira issue view <key>                    # View an issue
atl --context prod jira issue view <key> --json             # View as JSON
atl --context prod jira issue view <key> --web              # Open in browser

atl --context prod jira issue list                          # List recent issues
atl --context prod jira issue list --assignee @me           # Your assigned issues
atl --context prod jira issue list --project PROJ           # Issues in project
atl --context prod jira issue list --jql "status = Open"    # Custom JQL query
atl --context prod jira issue list --json                   # Output as JSON

atl --context prod jira issue create --project PROJ --type Bug --summary "Title"
atl --context prod jira issue create --project PROJ --type Task --summary "Title" --description "Details"
atl --context prod jira issue create --project PROJ --type Story --summary "Title" --field "Story Points=5"
atl --context prod jira issue create --project PROJ --type Task --summary "Title" --field-file fields.json
atl --context prod jira issue create --project PROJ --parent PROJ-123 --summary "Subtask"  # Auto-discovers subtask type

atl --context prod jira issue edit <key> --summary "New summary"
atl --context prod jira issue edit <key> --assignee @me
atl --context prod jira issue edit <key> --add-label bug --remove-label wontfix
atl --context prod jira issue edit <key> --field "Story Points=8"    # Set custom field by name
atl --context prod jira issue edit <key> --field-file fields.json    # Complex fields from JSON file

atl --context prod jira issue transition <key> "In Progress"
atl --context prod jira issue transition <key> --list       # List available transitions

atl --context prod jira issue comment add <key> --body "Comment text"
atl --context prod jira issue comment list <key>          # List comments
atl --context prod jira issue comment edit <key> --id 12345 --body "Updated text"
atl --context prod jira issue comment delete <key> --id 12345
atl --context prod jira issue comment add <key> --reply-to 12345 --body "Reply text"
atl --context prod jira issue comment add <key> --body "Internal note" --visibility-type role --visibility-name Developers

atl --context prod jira issue assign <key> --assignee @me
atl --context prod jira issue assign <key> --assignee -     # Unassign

atl --context prod jira issue link <key> <target-key>                    # Link issues (default: Relates)
atl --context prod jira issue link <key> <target-key> --type Blocks      # Link with specific type
atl --context prod jira issue link <key> --list-types                    # List available link types

atl --context prod jira issue weblink <key> --url "https://..." --title "Title"  # Add web link
atl --context prod jira issue weblink <key> --list                       # List web links
atl --context prod jira issue weblink <key> --delete 12345               # Delete web link by ID

atl --context prod jira issue types --project PROJ           # List issue types (shows subtask types)

atl --context prod jira issue fields                        # List all fields
atl --context prod jira issue fields --custom               # List custom fields only
atl --context prod jira issue fields --search "story"       # Search for fields by name

atl --context prod jira issue sprint <key> --sprint-id 123  # Move issue to sprint
atl --context prod jira issue sprint <key> --backlog        # Move issue to backlog
atl --context prod jira issue sprint <key> --list-sprints --board-id 1   # List sprints

atl --context prod jira issue flag <key>                    # Flag issue (mark as blocked)
atl --context prod jira issue flag <key> --unflag           # Remove flag
atl --context prod jira issue flag <key> --status           # Check if flagged

atl --context prod jira issue attachment <key> --list       # List attachments
atl --context prod jira issue attachment <key> --download --id 12345  # Download specific file
atl --context prod jira issue attachment <key> --download-all         # Download all attachments
atl --context prod jira issue attachment <key> --download-all -o ./dir  # Download to directory
```

`issue comment add --reply-to` creates a new flat Jira comment, mentions
the original author, and links to the focused original comment. It does not
copy the original body into the reply. For manually authored comments, plain
`@Display Name` is only text; use `@[Display Name]` for a real Jira mention.

### Boards

```bash
atl --context prod jira board list                          # List all boards
atl --context prod jira board list --project PROJ           # List boards for a project

atl --context prod jira board rank PROJ-123 --before PROJ-456   # Rank issue before another
atl --context prod jira board rank PROJ-123 --after PROJ-456    # Rank issue after another
atl --context prod jira board rank PROJ-1 PROJ-2 PROJ-3 --before PROJ-4  # Rank multiple issues in order
atl --context prod jira board rank PROJ-123 --top --board-id 42    # Move to top of backlog
```

### Confluence

```bash
atl --context prod confluence space list               # List spaces
atl --context prod confluence space list --json        # Output as JSON

atl --context prod confluence page view <id>           # View page by ID
atl --context prod confluence page view --space DOCS --title "Title"
atl --context prod confluence page view <id> --json    # Output as JSON
atl --context prod confluence page view <id> --web     # Open in browser

atl --context prod confluence page list --space DOCS   # List pages in space

atl --context prod confluence page create --space DOCS --title "New Page"
atl --context prod confluence page create --space DOCS --title "New Page" --body "Content"

atl --context prod confluence page edit <id> --title "Updated Title"
atl --context prod confluence page edit <id> --body "New content"

atl --context prod confluence page children <id>       # List child pages
atl --context prod confluence page children <id> --descendants  # Include all descendants

atl --context prod confluence page search "query"      # Search pages by title
atl --context prod confluence page search "query" --space DOCS  # Search within space

atl --context prod confluence page archive <id>        # Archive a page
atl --context prod confluence page archive <id> --unarchive     # Restore archived page

atl --context prod confluence page move <id> --target <parent-id>           # Move as child of target
atl --context prod confluence page move <id> --target <sibling-id> --position before  # Move before sibling
atl --context prod confluence page move <id> --space NEWSPACE               # Move to different space

atl --context prod confluence page attachment <id> --list               # List attachments
atl --context prod confluence page attachment <id> --list --json        # List as JSON
atl --context prod confluence page attachment <id> --download --id <attID>  # Download specific
atl --context prod confluence page attachment <id> --download-all       # Download all
atl --context prod confluence page attachment <id> --download-all -o ./dir  # Download to directory
atl --context prod confluence page attachment <id> --upload ./file.pdf  # Upload file
atl --context prod confluence page attachment <id> --upload a.pdf --upload b.png  # Upload multiple
```

### Configuration

```bash
atl config list                         # List all config
atl config list --json                  # Output as JSON
atl config get <key>                    # Get config value
atl config set <key> <value>            # Set config value
```

Available config keys:
- `current_host` - Active Atlassian host
- `default_output_format` - Default output format (text/json)
- `editor` - Editor for editing content
- `pager` - Pager for long output

### Explicit invocation context

The persistent `current_host` is convenient for an interactive shell, but it is
shared by every process using the same config file. Agents and automation should
select a host per invocation instead:

```bash
atl --context prod jira issue view PROJ-1234
atl --context sandbox confluence space list
atl --context sandbox jira assets object 9244 --json
```

The value may be a configured alias or hostname. `--context` does not mutate
`current_host`. `ATLASSIAN_CONTEXT=prod atl ...` provides the same process-local
override.

## Configuration

Configuration is stored in `~/.config/atlassian/config.yaml`.

Example configuration:

```yaml
version: 1
current_host: mycompany.atlassian.net
hosts:
  mycompany.atlassian.net:
    hostname: mycompany.atlassian.net
    cloud_id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
    default_project: PROJ
default_output_format: text
```

## Environment Variables

- `ATLASSIAN_CLIENT_ID` - OAuth client ID (highest-precedence source for login; otherwise OS keychain, then config file)
- `ATLASSIAN_CLIENT_SECRET` - OAuth client secret (highest-precedence source for login; otherwise OS keychain, then config file)
- `ATLASSIAN_TOKEN` - Override access token
- `ATLASSIAN_CONTEXT` - Select a configured host or alias for this invocation
- `ATLASSIAN_CONFIG_DIR` - Override config directory
- `NO_COLOR` - Disable colored output

## Shell Completion

```bash
# Bash (Linux)
atl completion bash | sudo tee /etc/bash_completion.d/atl > /dev/null

# Bash (macOS with Homebrew)
atl completion bash > $(brew --prefix)/etc/bash_completion.d/atl

# Bash (user-local alternative)
mkdir -p ~/.local/share/bash-completion/completions
atl completion bash > ~/.local/share/bash-completion/completions/atl

# Zsh
echo 'source <(atl completion zsh)' >> ~/.zshrc

# Fish
atl completion fish > ~/.config/fish/completions/atl.fish

# PowerShell
atl completion powershell >> $PROFILE
```

## Troubleshooting

### "Scope does not match" or 403 errors after updating

When the CLI adds new features that require additional OAuth scopes (like Jira Assets), you may get permission errors even after adding the scopes to your OAuth app. Existing tokens do not gain scopes retroactively.

**Solution:** Authenticate each affected site explicitly to replace its token:

```bash
atl auth login --hostname mycompany.atlassian.net
atl auth login --hostname mycompany-sandbox.atlassian.net
```

### Token expired errors

The CLI automatically refreshes expired tokens. If you see persistent token errors:

```bash
atl auth status    # Check current auth state
atl auth logout    # Clear stored tokens
atl auth login     # Re-authenticate
```

### OAuth app configuration

If authentication fails, verify your OAuth app configuration at https://developer.atlassian.com/console/myapps/:

1. **Callback URL** must be exactly: `http://localhost:8085/callback`
2. **Required scopes** for full functionality:

   **Jira API** (under "Jira API" in Developer Console):
   - Classic scopes: `read:jira-work`, `write:jira-work`, `read:jira-user`
   - Granular scopes: `read:project:jira`, `read:issue-details:jira`, `read:cmdb-object:jira`, `read:cmdb-schema:jira`
   - Granular scopes for boards/sprints/ranking: `read:board-scope:jira-software`, `write:board-scope:jira-software`, `read:issue:jira-software`, `write:issue:jira-software`, `read:sprint:jira-software`, `write:sprint:jira-software`

   **Confluence API** (under "Confluence API"):
   - Classic scopes: `read:confluence-content.all`, `write:confluence-content`
   - Granular scopes: `read:space:confluence`, `read:page:confluence`, `write:page:confluence`, `delete:page:confluence`, `read:content:confluence`, `write:content:confluence`, `read:content.metadata:confluence`, `read:hierarchical-content:confluence`, `read:folder:confluence`, `write:folder:confluence`, `delete:folder:confluence`, `read:template:confluence`, `write:template:confluence`

   > **Note:** Both Confluence classic and granular scopes are needed as the CLI uses both API versions.

## Development

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Run all checks
make check
```

## License

MIT License
