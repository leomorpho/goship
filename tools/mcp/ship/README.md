# ship-mcp

Minimal MCP server for GoShip docs and CLI guidance.

## Location

This MCP server lives in the same repository as the app/framework and CLI:

- app/framework module: repo root
- CLI module: `tools/cli/ship`
- MCP module: `tools/mcp/ship`

## Run

From repository root:

```bash
go run ./tools/mcp/ship/cmd/ship-mcp
```

Optional docs root override:

```bash
SHIP_MCP_DOCS_ROOT=docs go run ./tools/mcp/ship/cmd/ship-mcp
```

## Tools

- `ship_help`: return usage text for `ship` commands.
- `docs_search`: search markdown docs under `docs/`.
- `docs_get`: fetch one markdown doc by path.
