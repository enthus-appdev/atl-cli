# Atlassian CLI (atl)

A command-line tool for working with Jira, Confluence, and Tempo. Designed with LLM-friendly output for easy integration with AI assistants.

## Installation

### Quick Install (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/enthus-appdev/atl-cli/main/install.sh | bash
```

### From Source

```bash
go install github.com/enthus-appdev/atl-cli/cmd/atl@latest
```

### Using Make

```bash
git clone https://github.com/enthus-appdev/atl-cli.git
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

atl issue edit <key> --summary "New summary"
atl issue edit <key> --assignee @me
atl issue edit <key> --add-label bug --remove-label wontfix

atl issue transition <key> "In Progress"
atl issue transition <key> --list       # List available transitions

atl issue comment <key> --body "Comment text"
atl issue comment <key> --list          # List comments

atl issue assign <key> --assignee @me
atl issue assign <key> --assignee -     # Unassign
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
# Bash
atl completion bash > /etc/bash_completion.d/atl

# Zsh
atl completion zsh > "${fpath[1]}/_atl"

# Fish
atl completion fish > ~/.config/fish/completions/atl.fish

# PowerShell
atl completion powershell > atl.ps1
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
   - `read:sprint:jira-software`
   - `write:sprint:jira-software`

   **Confluence API** (under "Confluence API"):
   - Granular scopes: `read:space:confluence`, `read:page:confluence`, `write:page:confluence`, `read:content:confluence`, `write:content:confluence`, `read:content.metadata:confluence`, `read:hierarchical-content:confluence`

   > **Note:** Each product's scopes must be added under that specific product in the Developer Console. Jira Software has no classic scopes - only granular.

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
