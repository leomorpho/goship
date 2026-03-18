# AI Agent Guide

This guide is for code agents making changes in this repository.

## Start Here

1. Read `docs/architecture/03-project-scope-analysis.md` and `docs/architecture/06-known-gaps-and-risks.md`.
2. Every major layer (`framework/`, `app/`) and every module (`modules/<name>/`) contains its own `CLAUDE.md`. Read the scoped guide before making any changes to its directory.
3. If no scoped `CLAUDE.md` exists for the area you are editing, fall back to this guide.
4. Inspect route wiring in `app/router.go` before editing handlers.
5. Inspect `app/foundation/container.go` before assuming a dependency is initialized.

## Architectural Conventions

- HTTP handlers live in `app/web/controllers`.
- Domain logic should prefer repository/module packages (`framework/repos/...`, `modules/...`) over route-level DB logic.
- Rendering is typically done via `controller.Page` + templ components.
- Enums/constants are centralized in `framework/domain`.
- Module-specific work should start by reading that module's `CLAUDE.md`. New modules scaffolded via `ship make:module` automatically include a `CLAUDE.md` from the standard template.

## Safe Change Workflow

1. Identify layer to change:
- routing
- repository/service
- domain/schema
- template/frontend

2. Check for related tests:
- `rg "func Test" app/... framework/... modules/...`
- route tests in `app/web/controllers/*_test.go`

3. Implement minimal, local change first.
4. Run targeted tests, then broader tests if needed.
5. Update docs in `docs/` when behavior or architecture changes.

## Key Files By Concern

Runtime bootstrap:

- `cmd/web/main.go`
- `cmd/worker/main.go`
- `cmd/seed/main.go`

Dependency wiring:

- `app/foundation/container.go`
- `app/foundation/auth.go`
- `app/foundation/core_jobs_adapter.go`

Routing and middleware:

- `app/router.go`
- `app/web/middleware/*.go`

Data and domain:

- `db/queries/*.sql`
- `framework/repos/**/*.go`
- `framework/domain/*.go`

UI and rendering:

- `app/web/ui/*.go`
- `app/views/**/*.templ`
- `frontend/javascript/**/*`

## Common Pitfalls

- Assuming cache/notifier/task clients are initialized in the container.
- Implementing a route but not registering it in `router.go`.
- Adding schema logic without checking migration/generation workflow.
- Updating frontend behavior without checking templ + JS integration points.

## Commands Commonly Used

- `make dev` (default local dev: canonical app-on loop)
- `make dev-full` (web + worker + JS/CSS watchers)
- `make test` (Go tests)
- `make test-integration`
- `go run ./tools/cli/ship/cmd/ship test`
- `go run ./tools/cli/ship/cmd/ship test --integration`
- `make build-js`
- `make build-css`
- `make templ-gen`
- `go run ./tools/cli/ship/cmd/ship db:generate`
- `go run ./tools/cli/ship/cmd/ship db:make your_change`
- `go run ./tools/cli/ship/cmd/ship db:migrate`
- `go run ./tools/cli/ship/cmd/ship db:status`

## Documentation Rule

When code behavior changes, update at least:

- `docs/architecture/03-project-scope-analysis.md` if capability changed
- `docs/architecture/04-http-routes.md` if route surface changed
- `docs/architecture/06-known-gaps-and-risks.md` if a risk was added/removed

## Test Tagging Rule

- Integration tests must use `//go:build integration`.
- Keep default tests (`ship test`) stateless and fast by tagging infra-dependent tests.

## Nil Safety

- `nilaway` is part of CI and should be treated as part of the default correctness bar.
- `ship doctor` surfaces `nilaway` findings as warnings so agents can see nil-flow risks before merge.
- When touching pointer-heavy code, prefer explicit nil guards or value-based types over relying on downstream callers.
- `app/web/viewmodels/` must use value types only. Do not add pointer fields or slices of pointers there.
- Controllers and route handlers own domain-to-viewmodel mapping, including all nil handling for nullable domain data.
- Templ components should accept viewmodels or primitives, not `*domain` models passed through from handlers.

## UI Agent Convention

Before any UI work, read `docs/ui/style-guide.md` for design system context.

The full `data-*` attribute rules and templ comment rules are defined in `docs/ui/convention.md`.

For Playwright MCP setup, see `MCP_TOOLS.md` at the repo root.

### LLM Workflow for a UI Task

**Without a running dev server (grep-only mode):**
1. Read `docs/ui/style-guide.md` for design system context.
2. Grep `data-component="<name>"` to locate the component file.
3. Read the `// Renders:` comment to understand the visual output.
4. Grep `data-slot="<slot>"` to locate the target sub-element.
5. Make the change, preserving existing Tailwind class patterns.
6. Never use `data-component`, `data-slot`, or `data-action` in CSS selectors.

**With a running dev server + Playwright MCP (preferred):**
1. Read `docs/ui/style-guide.md` for design system context.
2. Grep `data-component="<name>"` to locate the component file.
3. Read the `// Route(s):` comment above the component to know where to navigate.
4. Use `browser_navigate` to go to that route on the local dev server (`make dev`).
5. Use `browser_screenshot` to capture the current state (before).
6. Use `browser_snapshot` to inspect the accessibility tree and confirm component structure.
7. Make the code change.
8. After server reload, use `browser_screenshot` again to verify the visual result (after).

**When the route is unknown:**
- If `// Route(s):` says `embedded in <Parent>`, navigate to the parent's route instead.
- If route is genuinely unknown: use the GoShip MCP `ship_routes` tool to list all registered routes,
  then navigate candidate routes and confirm via `browser_snapshot` which `data-component` values appear.

`data-*` attributes are semantic only — never for styling.
