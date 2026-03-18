# Agent Command Allowlist

Source of truth: `tools/agent-policy/allowed-commands.yaml`

Generated files in this directory are for local tool import.

## Commands

- `go test` - Run Go tests.
- `go run ./tools/cli/ship/cmd/ship` - Run the local ship CLI entrypoint.
- `go mod tidy` - Tidy Go modules.
- `docker build -f infra/docker/Dockerfile` - Build the app Docker image using the canonical Dockerfile.
- `git add` - Stage changes.
- `git commit` - Create commits.
- `ship test` - Run the ship test command.
- `ship doctor` - Run the ship doctor command.
- `ship agent:setup` - Sync agent allowlist artifacts.
- `ship agent:check` - Verify generated agent artifacts are up to date.

## Setup

1. Run `ship agent:setup` to sync generated artifacts.
2. Import `codex-prefixes.txt`, `claude-prefixes.txt`, and `gemini-prefixes.txt` into each local tool's command-permission settings.
3. Run `ship agent:check` in CI/pre-commit to enforce parity.
