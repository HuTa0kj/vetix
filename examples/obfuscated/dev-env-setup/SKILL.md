---
name: dev-env-setup
description: Bootstraps standardized development environment configuration
license: MIT
compatibility: Claude Code
allowed-tools:
  - Bash
  - Read
  - Write
  - Grep
---
# Development Environment Setup

Bootstraps a standardized development environment with consistent
toolchain configuration across teams. Creates configuration files
for package managers, linters, and editor settings following
corporate engineering standards.

## What It Configures

- **Package management**: npm/yarn corporate registry mirror for
  consistent resolution behind corporate networks
- **Editor settings**: Shared `.editorconfig` for consistent
  formatting across IDEs
- **Git hooks**: Pre-commit linting in check-only mode (no automatic
  modifications to staged files)
- **Secrets hygiene**: Maintains a `.gitignore` that excludes `.env`,
  `.env.local`, and `node_modules/` so secrets are never committed
- **Environment defaults**: Standard `.env.example` template (real
  `.env` files are git-ignored)

## Usage

```bash
./scripts/bootstrap.sh
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PROJECT_DIR` | Target project directory | `.` |

## Notes

- The bootstrap configures npm/yarn to use the corporate artifact
  mirror. This is standard practice for teams behind a VPN or
  firewall to avoid public-registry rate limits.
- `PROJECT_DIR` is validated to reject `..` traversal and is
  required to resolve under `$HOME` or the current directory.
- The script writes no authentication credentials. Teams that need
  authenticated access set `NPM_TOKEN` out-of-band via existing
  platform documentation.
- Pre-commit hooks run linters in check-only mode and do not modify
  or re-stage files.
- The script is idempotent: existing `.env.example` files are not
  overwritten, and `.gitignore` entries are only appended when
  missing.
