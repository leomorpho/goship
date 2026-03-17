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

When the `tools/mcp/ship` module is present in your checkout, use:

### Build local binary

```bash
go build -o ~/.local/bin/ship-mcp ./tools/mcp/ship/cmd/ship-mcp
```

### Register server

```bash
codex mcp add ship -- ~/.local/bin/ship-mcp
gemini mcp add --scope user ship ~/.local/bin/ship-mcp
claude mcp add --scope user ship -- ~/.local/bin/ship-mcp
```

## 3) Playwright MCP (Recommended for UI Work)

Gives your AI coding client a real browser: navigate pages, take screenshots it can actually see,
inspect the accessibility tree, and interact with HTMX flows on a running dev server.

### Claude Code

```bash
claude mcp add --scope user playwright -- npx -y @playwright/mcp
```

### Codex

```bash
codex mcp add playwright -- npx -y @playwright/mcp
```

### Gemini CLI

```bash
gemini mcp add --scope user playwright npx -y @playwright/mcp
```

### Usage pattern for UI dev

1. Start the dev server: `make dev` (runs on `http://localhost:8000` by default — check `config/config.yaml`).
2. Look up the component's `// Route(s):` annotation in its `.templ` file.
3. Use `browser_navigate` to go to that route on the local dev server.
4. Use `browser_screenshot` to capture the current visual state (before).
5. Use `browser_snapshot` to inspect the accessibility tree and confirm `data-component` / `data-slot` structure.
6. Make the code change.
7. After the server reloads, use `browser_screenshot` again to verify the visual result (after).

### When the route is unknown

- If `// Route(s):` says `embedded in <Parent>`, navigate to the parent component's route instead.
- If the route is genuinely unknown: use the GoShip MCP `ship_routes` tool to list all registered routes,
  then navigate candidate routes and check via `browser_snapshot` which `data-component` values appear on the page.

## Notes

1. Prefer `--scope user` for personal global setup, or project/local scope if you want repo-only registration.
2. Restart your CLI if a newly added MCP server does not appear immediately.
3. Keep this file updated whenever recommended MCP tooling changes.
