# MCP Server (ship-mcp)

This document defines the minimal MCP server currently shipped in this repository.

## Status

Current status:

1. Not actively used in day-to-day framework development right now.
2. Kept as a future extension point for LLM-facing tooling.
3. Near-term priority is high-quality human + LLM-friendly documentation under `docs/`.

## Location

- Module: `tools/mcp/ship`
- Entrypoint: `tools/mcp/ship/cmd/ship-mcp/main.go`
- Workspace wiring: `go.work`

This is a standalone Go module in the same monorepo as:

- app/framework module at repo root
- CLI module at `tools/cli/ship`

## Purpose

Provide an LLM-facing interface for:

1. `ship` command usage guidance.
2. focused access to docs under `docs/`.

## Current Tool Set (V1)

1. `ship_help`
- Returns usage/help text for core `ship` commands.
- Input: optional `topic` (`general`, `dev`, `test`, `db`).

2. `ship_doctor`
- Runs `ship doctor --json` and returns the structured result.
- Input: none.

3. `ship_routes`
- Runs `ship describe` and returns the route inventory.
- Input: optional `filter` (`public`, `auth`, `admin`).

4. `ship_modules`
- Runs `ship describe` and returns the installed module list.
- Input: none.

5. `ship_verify`
- Runs `ship verify --json` and returns step-by-step verification results.
- Input: optional `skip_tests` boolean.

6. `docs_search`
- Searches markdown files under `docs/`.
- Input: `query` (required), `limit` (optional, default 20, max 50).

7. `docs_get`
- Returns a single markdown document from `docs/`.
- Input: `path` (required), relative to `docs/`.

## Runtime Notes

- Transport: stdio with MCP JSON-RPC framing (`Content-Length` headers).
- Default docs root: `docs`.
- Override docs root with `SHIP_MCP_DOCS_ROOT`.

Run from repo root:

```bash
go run ./tools/mcp/ship/cmd/ship-mcp
```

## Install In MCP Clients

Build a reusable local binary first:

```bash
go build -o ~/.local/bin/ship-mcp ./tools/mcp/ship/cmd/ship-mcp
```

Use an absolute docs path:

```bash
export SHIP_MCP_DOCS_ROOT=/path/to/goship/docs
```

Register in your MCP-compatible agent CLI:

```bash
<agent-cli> mcp add --scope user -e SHIP_MCP_DOCS_ROOT=/path/to/goship/docs ship -- ~/.local/bin/ship-mcp
<agent-cli> mcp list
```

Notes:

1. `--scope user` makes this MCP server available globally; use project/local scope for repo-only registration.
2. Restart your CLI session after adding a new MCP server.
3. If `~/.local/bin` is not on `PATH`, keep using the absolute binary path.

## Release Binaries (MCP Internals)

`ship-mcp` prebuilt binaries are published from this repository using GoReleaser and GitHub Actions.

Internal release files:

- `.goreleaser.ship-mcp.yml`
- `.github/workflows/release-ship-mcp.yml`

Workflow trigger policy:

1. Tag push only: `ship-mcp/v*`
2. Manual trigger allowed: `workflow_dispatch`
3. No release on regular branch pushes

Tag example:

```bash
git tag ship-mcp/v0.1.0
git push origin ship-mcp/v0.1.0
```

What gets published:

1. `ship-mcp` binaries for `linux`, `darwin`, `windows`
2. Architectures: `amd64`, `arm64`
3. Checksums file: `ship-mcp_checksums.txt`

## Safety Rules

1. `docs_get` only reads inside `docs/` (path traversal blocked).
2. large doc payloads are truncated.
3. unknown methods/tools return structured JSON-RPC errors.

## Near-Term Extensions

1. Add a `ship_run` tool with allowlisted commands.
2. Add doc IDs that map to numbered filenames for stable retrieval.
3. Add cross-doc link graph output for faster agent navigation.
