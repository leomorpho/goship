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
- `ship_doctor`: run `ship doctor --json` and return the structured result.
- `ship_routes`: run `ship describe` and return route inventory.
- `ship_modules`: run `ship describe` and return installed modules.
- `ship_verify`: run `ship verify --json` and return verification steps.
- `docs_search`: search markdown docs under `docs/`.
- `docs_get`: fetch one markdown doc by path.
