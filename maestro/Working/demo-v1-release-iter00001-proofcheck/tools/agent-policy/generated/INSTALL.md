# Agent Command Allowlist

Source of truth: `tools/agent-policy/allowed-commands.yaml`

Generated files in this directory are for local tool import.

## Commands

- `go test` - Run Go tests.

## Setup

1. Run `ship agent:setup` to sync generated artifacts.
2. Import `agent-prefixes.txt` into your local agent tool's command-permission settings.
3. Run `ship agent:check` in CI/pre-commit to enforce parity.
