# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with this repository.

## Project Overview

This is `atl`, a command-line tool for interacting with Atlassian products (Jira, Confluence, and Tempo). It's designed with LLM-friendly output - all commands support `--json` for structured output.

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
    worklog/             # worklog add|list|edit|delete (partially implemented)
    config/              # config get|set|list
  config/                # Configuration management (~/.config/atlassian/)
  iostreams/             # I/O abstraction for testability
  output/                # Output formatting (JSON, tables, colors)
```

## Key Patterns

### Adding a New Command

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

All commands should support `--json` flag:
```go
if opts.JSON {
    return output.JSON(opts.IO.Out, outputStruct)
}
// Plain text output
fmt.Fprintf(opts.IO.Out, "Result: %s\n", value)
```

## Authentication

OAuth 2.0 credentials can be configured two ways:

1. **Interactive setup (recommended)**: `atl auth setup` - walks through creating OAuth app
2. **Environment variables**: `ATLASSIAN_CLIENT_ID` and `ATLASSIAN_CLIENT_SECRET`

Credentials are stored in `~/.config/atlassian/config.yaml` under `oauth:` key.
Access tokens are stored in system keyring.

### Auth Flow

1. `atl auth setup` - Configure OAuth app credentials (one-time)
2. `atl auth login` - Opens browser for OAuth consent, stores tokens
3. `atl auth status` - Check current auth state
4. `atl auth refresh` - Manually refresh tokens
5. `atl auth logout` - Remove tokens

## Testing Commands

Without auth (will show helpful error):
```bash
./bin/atl auth login
./bin/atl issue list
./bin/atl confluence page list
```

With auth configured:
```bash
./bin/atl auth status
./bin/atl issue view PROJ-123 --json
./bin/atl confluence space list --json
```

## Incomplete Features

- `worklog` commands are stubs (Tempo API not yet implemented)
- No pagination support for list commands yet
