# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| latest  | :white_check_mark: |
| < latest | :x:               |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please report vulnerabilities via email:

- **Email:** security@vectrade.io
- **Subject:** `[SECURITY] vectrade-cli: <brief description>`

Include:
- Description of the vulnerability
- Steps to reproduce
- Impact assessment
- Suggested fix (if any)

We will acknowledge your report within 48 hours and provide a timeline for a fix within 5 business days.

## Security Practices

- API keys and JWT tokens are stored with `0600` permissions (owner-only)
- HTTPS is enforced for all API communication (except localhost development)
- Webhook forwarding is restricted to localhost URLs to prevent SSRF
- Binaries are signed with cosign and include SBOM via syft
- Dependencies are monitored via GitHub Dependabot
