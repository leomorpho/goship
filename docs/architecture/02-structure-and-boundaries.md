# Structure and Boundaries
<!-- FRONTEND_SYNC: Landing capability explorer in app/goship/views/web/pages/landing_page.templ links here for Views and Server UI. Keep both landing copy and this doc aligned. -->

This document defines where code belongs as GoShip evolves into a Rails-like framework plus example app.

## Current Top-Level Shape (Single Repository)

- `app/goship/`: app-specific code for the first-party GoShip app
- `cmd/`: runnable entrypoints for the app module (`web`, `worker`, `seed`)
- `cli/ship/`: standalone CLI module (`ship`) that lives in this same repository
- `mcp/ship/`: standalone MCP module (`ship-mcp`) for LLM-facing docs/CLI tooling
- `pkg/`: reusable framework-level libraries and adapters
- `config/`: runtime configuration
- `ent/`: schema and generated ORM
- `docs/`: internal design and implementation documentation

Monorepo note:

- GoShip currently uses one repository with multiple Go modules:
- root app/framework module + `cli/ship` module + `mcp/ship` module
- `go.work` ties local development across modules together for maintainers.

## App vs Framework Rules

Use this placement rule for every new file:

- Put code in `app/goship/...` when it encodes product behavior/UI for the GoShip app.
- Put code in `pkg/...` when it is reusable as framework infrastructure across apps.

## Web Layer Layout

App web code is now app-scoped:

- `app/goship/web/controllers`: handlers
- `app/goship/web/wiring.go`: HTTP wiring helpers (middleware/static/dependencies)
- `app/goship/services`: app composition container and app-specific runtime adapters
- `app/goship/views`: templ components/layouts/pages/emails
- `app/goship/views/**/gen`: generated templ Go files (same package names as source dirs)

Router source of truth:

- `app/goship/router.go`

HTTP boundary rule:

- Route handlers in `app/goship/web/controllers` are the controller layer (Rails/Laravel equivalent at the HTTP boundary).
- Controllers should stay thin: parse request input, call service/use-case logic, map output to HTTP response.
- Business logic should not live directly in controllers.

Service/store rule:

- App business logic should live in app-scoped domain/service packages under `app/goship/...`.
- Services should depend on explicit interfaces (store ports) for persistence/external calls.
- Concrete adapters may use Ent/SQL/clients directly, but those details stay behind service dependencies.
- This keeps testability (mocks/fakes) without forcing repository pattern across every feature.

## Refactor Status

Completed in this pass:

- Moved routes from `pkg/routing/routes` to `app/goship/web/controllers`.
- Moved templ views from `templates` to `app/goship/views`.
- Updated imports and test package paths accordingly.

Installable module extraction rule:

A package should be treated as an installable official module only if all are true:

1. It has no hard dependency on `app/goship/web/controllers` or app views/templates.
2. Its config surface is stable and documented.
3. It can be wired through stable interfaces/adapters (no deep app internals required).
4. It has dedicated tests and docs and can be enabled through `ship` install/wire commands.

Classification policy for current `pkg/repos/*`:

- app-specific behavior => move to `app/goship/...` domain packages.
- reusable framework capability => keep/extract as framework module (installable by `ship`).
