# Contributing to VecTrade CLI

Thank you for your interest in contributing! This document provides guidelines for contributing to the VecTrade CLI.

## Development Setup

```bash
# Clone the repository
git clone https://github.com/VecTrade-io/vectrade-cli.git
cd vectrade-cli

# Verify Go version (see go.mod for minimum)
go version

# Run tests
go test -race ./...

# Build
go build -o vectrade .
```

## Code Standards

- Run `go vet ./...` before submitting
- Run `golangci-lint run` for comprehensive linting
- All exported symbols must have doc comments
- Use `context.Context` for cancellable operations
- Wrap errors with `fmt.Errorf("doing X: %w", err)`

## Testing

- Write table-driven tests where appropriate
- Use `t.Parallel()` for independent tests
- Aim for ≥60% coverage on new code
- Test both success and error paths

```bash
# Run with coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Pull Requests

1. Fork the repository and create a feature branch
2. Make your changes with tests
3. Ensure `go test -race ./...` passes
4. Ensure `go vet ./...` reports no issues
5. Submit a PR with a clear description

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add webhook retry support
fix: handle nil response in quote command
docs: update installation instructions
ci: add golangci-lint step
```

## Reporting Issues

- Use GitHub Issues with a clear title and reproduction steps
- Include CLI version (`vectrade version`) and OS info
- Attach relevant logs or error output

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
