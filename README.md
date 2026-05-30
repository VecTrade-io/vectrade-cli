# VecTrade CLI

[![CI](https://github.com/VecTrade-io/vectrade-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/VecTrade-io/vectrade-cli/actions/workflows/ci.yml) [![License](https://img.shields.io/github/license/VecTrade-io/vectrade-cli)](LICENSE) [![Go](https://img.shields.io/badge/go-1.22+-00ADD8.svg)](https://go.dev/) [![Coverage](https://img.shields.io/badge/coverage-%3E90%25-brightgreen)](https://github.com/VecTrade-io/vectrade-cli/actions/workflows/ci.yml)

Cross-platform CLI for VecTrade financial data and AI analysis.

## Installation

```bash
# macOS (Homebrew)
brew install VecTrade-io/vectrade/vectrade

# Linux/macOS (shell script)
curl -fsSL https://get.vectrade.io/cli | sh

# Windows (scoop)
scoop bucket add vectrade https://github.com/VecTrade-io/scoop-vectrade
scoop install vectrade

# Go install
go install github.com/VecTrade-io/vectrade-cli@latest
```

## Usage

```bash
# Authenticate via browser-based OAuth (like gcloud auth login)
vectrade auth login
vectrade auth login --provider microsoft

# Check auth status
vectrade auth status

# Get a quote
vectrade quote AAPL

# AI analysis (streams to terminal)
vectrade ai "Compare AAPL vs MSFT for 2025 outlook"

# Manage API keys
vectrade keys create --label "my-trading-bot"
vectrade keys list
vectrade keys revoke KEY_ID

# Check usage & quota
vectrade usage

# Print access token for piping
curl -H "Authorization: Bearer $(vectrade auth token)" https://api.vectrade.io/v1/...

# Logout
vectrade auth logout
```

## Configuration

Config file: `~/.vectrade/config.yaml`

```yaml
api_key: vq_live_...
base_url: https://api.vectrade.io/v1
sandbox: false
timeout: 30
output: table  # table, json, csv
```

Environment variables: `VECTRADE_API_KEY`, `VECTRADE_BASE_URL`, `VECTRADE_SANDBOX`

### Credential Storage

After `vectrade auth login`, JWT tokens are stored with owner-only permissions:

| Platform | Path |
|----------|------|
| macOS | `~/Library/Application Support/vectrade/credentials.json` |
| Linux | `~/.config/vectrade/credentials.json` |
| Windows | `%APPDATA%/vectrade/credentials.json` |

## Documentation

Full documentation is available at [docs.vectrade.io/sdks/cli](https://docs.vectrade.io/sdks/cli).

- [API Reference](https://docs.vectrade.io/api-reference/overview)
- [All SDKs](https://docs.vectrade.io/sdks/python)

## Community

- 💬 [Discord](https://discord.gg/vectrade) — Get help, share scripts, discuss workflows
- 🤖 [MCP Server](https://github.com/VecTrade-io/vectrade-mcp) — Use VecTrade tools in AI IDEs
- 🧰 [finkit](https://github.com/VecTrade-io/finkit) — Open-source Python analysis library

## License

MIT — see [LICENSE](LICENSE).
