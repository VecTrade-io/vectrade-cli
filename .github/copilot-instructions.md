# VecTrade CLI — Copilot Instructions

## Workflow

All agents follow the standard workflow defined in `instructions/agent-workflow.instructions.md`:
**Implement → Verify → Changelog → Commit**

## Agents

| Agent | When to Use |
|-------|------------|
| `@vt-cli-dev` | Adding commands, implementing features |
| `@vt-cli-tester` | Writing/fixing Go tests |

## Conventions

- Go 1.22+
- Cobra for CLI framework
- Table-driven tests with httptest mocking
- Support `--format table|json|csv` on all commands
- Return errors, never panic
- Shell completions for all commands

## Build & Test

```bash
make build                 # Build binary
make test                  # Run tests
make lint                  # golangci-lint
make completions           # Generate shell completions
```

## Release

GoReleaser on tag push. Binaries for darwin/linux/windows × amd64/arm64.
