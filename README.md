# VecTrade CLI

[![License](https://img.shields.io/github/license/VecTrade-io/vectrade-cli)](LICENSE) [![Go](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)

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

## License

MIT — see [LICENSE](LICENSE).
