# Recommended MCP Tools

This file is the canonical list of MCP tools we recommend for GoShip contributors.

## 1) GitHub MCP (Recommended)

Use this to inspect repositories, PRs, issues, branches, and history directly from your AI coding client.

### Codex

```bash
codex mcp add github -- npx -y @modelcontextprotocol/server-github
codex mcp list
```

### Gemini CLI

```bash
gemini mcp add --scope user github npx -y @modelcontextprotocol/server-github
gemini mcp list
```

### Claude Code

```bash
claude mcp add --scope user github -- npx -y @modelcontextprotocol/server-github
claude mcp list
```

## 2) GoShip MCP (Optional, Future-Facing)

This is intended to expose GoShip-specific docs and CLI help to LLM agents.

Current default priority remains high-quality markdown docs in the repository.

When the `mcp/ship` module is present in your checkout, use:

### Build local binary

```bash
go build -o ~/.local/bin/ship-mcp ./mcp/ship/cmd/ship-mcp
```

### Register server

```bash
codex mcp add ship -- ~/.local/bin/ship-mcp
gemini mcp add --scope user ship ~/.local/bin/ship-mcp
claude mcp add --scope user ship -- ~/.local/bin/ship-mcp
```

## Notes

1. Prefer `--scope user` for personal global setup, or project/local scope if you want repo-only registration.
2. Restart your CLI if a newly added MCP server does not appear immediately.
3. Keep this file updated whenever recommended MCP tooling changes.
