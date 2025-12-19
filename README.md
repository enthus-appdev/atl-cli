# Atlassian CLI (atl)

A command-line tool for working with Jira, Confluence, and Tempo. Designed with LLM-friendly output for easy integration with AI assistants.

## Installation

### Quick Install (Recommended)

```bash
gh api repos/enthus-appdev/atl-cli/contents/install.sh -q '.content' | base64 -d | bash
```

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

# 2. Log in to your Atlassian account
atl auth login

# 3. Start using the CLI
atl issue list --assignee @me
atl confluence space list
```

## OAuth Setup

The `atl auth setup` command will guide you through creating an OAuth app. Here's what it does:

1. Opens https://developer.atlassian.com/console/myapps/
2. Walks you through creating an OAuth 2.0 integration
3. Helps you configure the callback URL: `http://localhost:8085/callback`
4. Stores your Client ID and Secret securely in `~/.config/atlassian/config.yaml`

Alternatively, you can set environment variables (useful for CI/CD):
```bash
export ATLASSIAN_CLIENT_ID="your-client-id"
export ATLASSIAN_CLIENT_SECRET="your-client-secret"
```

## Usage Examples

```bash
# View an issue
atl issue view PROJ-1234

# List your assigned issues
atl issue list --assignee @me

# Output as JSON for LLM processing
atl issue view PROJ-1234 --json

# View a Confluence page
atl confluence page view --space DOCS --title "Getting Started"
```

## LLM-Friendly Output

All commands support `--json` flag for structured JSON output, making it easy to parse and process with LLMs:

```bash
# Get issue data as JSON
atl issue view PROJ-1234 --json

# List issues as JSON
atl issue list --project PROJ --json

# Get spaces as JSON
atl confluence space list --json
```

Plain text output is also structured for easy parsing by LLMs.

## Commands

### Authentication

```bash
atl auth login        # Authenticate with Atlassian
atl auth logout       # Remove authentication
atl auth status       # View authentication status
```

### Jira Issues

```bash
atl issue view <key>                    # View an issue
atl issue view <key> --json             # View as JSON
atl issue view <key> --web              # Open in browser

atl issue list                          # List recent issues
atl issue list --assignee @me           # Your assigned issues
atl issue list --project PROJ           # Issues in project
atl issue list --jql "status = Open"    # Custom JQL query
atl issue list --json                   # Output as JSON

atl issue create --project PROJ --type Bug --summary "Title"
atl issue create --project PROJ --type Task --summary "Title" --description "Details"
atl issue create --project PROJ --type Story --summary "Title" --field "Story Points=5"
atl issue create --project PROJ --type Task --summary "Title" --field-file fields.json
atl issue create --project PROJ --parent PROJ-123 --summary "Subtask"  # Auto-discovers subtask type

atl issue edit <key> --summary "New summary"
atl issue edit <key> --assignee @me
atl issue edit <key> --add-label bug --remove-label wontfix
atl issue edit <key> --field "Story Points=8"    # Set custom field by name
atl issue edit <key> --field-file fields.json    # Complex fields from JSON file

atl issue transition <key> "In Progress"
atl issue transition <key> --list       # List available transitions

atl issue comment <key> --body "Comment text"
atl issue comment <key> --list          # List comments
atl issue comment <key> --edit --comment-id 12345 --body "Updated text"
atl issue comment <key> --delete --comment-id 12345
atl issue comment <key> --reply-to 12345 --body "Reply text"
atl issue comment <key> --body "Internal note" --visibility-type role --visibility-name Developers

atl issue assign <key> --assignee @me
atl issue assign <key> --assignee -     # Unassign

atl issue link <key> <target-key>                    # Link issues (default: Relates)
atl issue link <key> <target-key> --type Blocks      # Link with specific type
atl issue link <key> --list-types                    # List available link types

atl issue weblink <key> --url "https://..." --title "Title"  # Add web link
atl issue weblink <key> --list                       # List web links
atl issue weblink <key> --delete 12345               # Delete web link by ID

atl issue types --project PROJ           # List issue types (shows subtask types)

atl issue fields                        # List all fields
atl issue fields --custom               # List custom fields only
atl issue fields --search "story"       # Search for fields by name

atl issue sprint <key> --sprint-id 123  # Move issue to sprint
atl issue sprint <key> --backlog        # Move issue to backlog
atl issue sprint <key> --list-sprints --board-id 1   # List sprints

atl issue flag <key>                    # Flag issue (mark as blocked)
atl issue flag <key> --unflag           # Remove flag
atl issue flag <key> --status           # Check if flagged

atl issue attachment <key> --list       # List attachments
atl issue attachment <key> --download --id 12345  # Download specific file
atl issue attachment <key> --download-all         # Download all attachments
atl issue attachment <key> --download-all -o ./dir  # Download to directory
```

### Boards

```bash
atl board list                          # List all boards
atl board list --project PROJ           # List boards for a project

atl board rank NX-123 --before NX-456   # Rank issue before another
atl board rank NX-123 --after NX-456    # Rank issue after another
atl board rank NX-1 NX-2 NX-3 --before NX-4  # Rank multiple issues in order
atl board rank NX-123 --top --board-id 42    # Move to top of backlog
```

### Confluence

```bash
atl confluence space list               # List spaces
atl confluence space list --json        # Output as JSON

atl confluence page view <id>           # View page by ID
atl confluence page view --space DOCS --title "Title"
atl confluence page view <id> --json    # Output as JSON
atl confluence page view <id> --web     # Open in browser

atl confluence page list --space DOCS   # List pages in space

atl confluence page create --space DOCS --title "New Page"
atl confluence page create --space DOCS --title "New Page" --body "Content"

atl confluence page edit <id> --title "Updated Title"
atl confluence page edit <id> --body "New content"

atl confluence page children <id>       # List child pages
atl confluence page children <id> --descendants  # Include all descendants

atl confluence page search "query"      # Search pages by title
atl confluence page search "query" --space DOCS  # Search within space

atl confluence page archive <id>        # Archive a page
atl confluence page archive <id> --unarchive     # Restore archived page

atl confluence page move <id> --target <parent-id>           # Move as child of target
atl confluence page move <id> --target <sibling-id> --position before  # Move before sibling
atl confluence page move <id> --space NEWSPACE               # Move to different space
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

- `ATLASSIAN_CLIENT_ID` - OAuth client ID (required for login)
- `ATLASSIAN_CLIENT_SECRET` - OAuth client secret (required for login)
- `ATLASSIAN_TOKEN` - Override access token
- `ATLASSIAN_HOST` - Override default host
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

When the CLI adds new features that require additional OAuth scopes (like sprint management), you may get permission errors even after adding the scopes to your OAuth app.

**Solution:** Perform a full logout and login to refresh your token with the new scopes:

```bash
atl auth logout
atl auth login
```

Simply running `atl auth login` again may not be sufficient as the existing token retains its original scopes.

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
   - Granular scopes: `read:project:jira`

   **Jira Software API** (under "Jira Software API" - granular only, no classic scopes exist):
   - `read:board-scope:jira-software`
   - `write:board-scope:jira-software`
   - `read:issue:jira-software`
   - `write:issue:jira-software`
   - `read:sprint:jira-software`
   - `write:sprint:jira-software`

   **Confluence API** (under "Confluence API"):
   - Classic scopes: `read:confluence-content.all`, `write:confluence-content`
   - Granular scopes: `read:space:confluence`, `read:page:confluence`, `write:page:confluence`, `read:content:confluence`, `write:content:confluence`, `read:content.metadata:confluence`, `read:hierarchical-content:confluence`

   > **Note:** Each product's scopes must be added under that specific product in the Developer Console. Jira Software has no classic scopes - only granular. Both Confluence classic and granular scopes are needed as the CLI uses both API versions.

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
