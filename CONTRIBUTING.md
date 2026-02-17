# Contributing to n8n-cli

Thanks for your interest in contributing! Here's how to get started.

## Development Setup

```bash
git clone https://github.com/enthus-appdev/n8n-cli.git
cd n8n-cli
go build -o bin/n8nctl ./cmd/n8nctl   # Build
go test -v ./...                        # Run tests
golangci-lint run                       # Lint
go fmt ./...                            # Format code
```

Requires Go 1.25+ and [golangci-lint](https://golangci-lint.run/).

## Making Changes

1. Fork the repository and create a feature branch from `main`
2. Write your code and add tests where appropriate
3. Run `golangci-lint run && go test ./...` to ensure all checks pass
4. Commit with a clear message describing the change
5. Open a pull request against `main`

## Code Style

- Run `go fmt ./...` before committing
- Follow standard Go conventions
- Use [cobra](https://github.com/spf13/cobra) for new commands

## Reporting Bugs

Open a [GitHub issue](https://github.com/enthus-appdev/n8n-cli/issues) with:
- Steps to reproduce
- Expected vs actual behavior
- CLI version and OS

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
