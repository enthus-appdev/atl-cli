# Contributing to atl-cli

Thanks for your interest in contributing! Here's how to get started.

## Development Setup

```bash
git clone https://github.com/enthus-appdev/atl-cli.git
cd atl-cli
make build    # Build the binary
make test     # Run tests
make lint     # Run golangci-lint
make check    # Run all checks (fmt, vet, lint, test)
```

Requires Go 1.25+ and [golangci-lint](https://golangci-lint.run/).

## Making Changes

1. Fork the repository and create a feature branch from `main`
2. Write your code and add tests where appropriate
3. Run `make check` to ensure all checks pass
4. Commit with a clear message describing the change
5. Open a pull request against `main`

## Code Style

- Run `make fmt` (uses `goimports`) before committing
- Follow standard Go conventions
- All commands must support `--json` output
- See `AGENTS.md` for architecture details and patterns

## Reporting Bugs

Open a [GitHub issue](https://github.com/enthus-appdev/atl-cli/issues) with:
- Steps to reproduce
- Expected vs actual behavior
- CLI version (`atl --version`) and OS

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
