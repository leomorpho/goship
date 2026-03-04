# Structure and Boundaries
<!-- FRONTEND_SYNC: Landing capability explorer in app/views/web/pages/landing_page.templ links here for Views and Server UI. Keep both landing copy and this doc aligned. -->

This document defines where code belongs as GoShip evolves into a Rails-like framework plus example app.

## Current Top-Level Shape (Single Repository)

- `app/`: app-specific code for the first-party GoShip app
- `cmd/`: runnable entrypoints for app processes (`web`, `worker`, `seed`)
- `tools/cli/ship/`: standalone CLI module (`ship`) that lives in this same repository
- `tools/mcp/ship/`: standalone MCP module (`ship-mcp`) for LLM-facing docs/CLI tooling
- `framework/`: reusable framework-level libraries and adapters
- `modules/`: installable framework modules (workspace-local during development)
- `tools/`: internal tooling modules and scripts (`ship`, `ship-mcp`, helper scripts)
- `infra/`: deployment and infrastructure assets (Kamal, container assets, wrappers)
- `frontend/`: frontend toolchain and source (`package.json`, `build.mjs`, JS/Svelte)
- `tests/`: e2e and cross-package test assets
- `config/`: runtime configuration
- `db/`: monolith-owned schema and migration history
- `db/ent/`: schema and generated ORM
- `docs/`: internal design and implementation documentation

Monorepo note:

- GoShip currently uses one repository with multiple Go modules:
- root framework module + `apps` module + `tools/cli/ship` module + `tools/mcp/ship` module
- `go.work` ties local development across modules together for maintainers.

## App vs Framework Rules

Use this placement rule for every new file:

- Put code in `app/...` when it encodes product behavior/UI for the GoShip app.
- Put code in `framework/...` when it is reusable as framework infrastructure across apps.

## Web Layer Layout

App web code is now app-scoped:

- `app/web/controllers`: handlers
- `app/web/wiring.go`: HTTP wiring helpers (middleware/static/dependencies)
- `app/foundation`: app composition container and app-specific runtime adapters
- `app/views`: templ components/layouts/pages/emails
- `app/views/**/gen`: generated templ Go files (same package names as source dirs)

Router source of truth:

- `app/router.go`

HTTP boundary rule:

- Route handlers in `app/web/controllers` are the controller layer (Rails/Laravel equivalent at the HTTP boundary).
- Controllers should stay thin: parse request input, call service/use-case logic, map output to HTTP response.
- Business logic should not live directly in controllers.

Service/store rule:

- App business logic should live in app-scoped domain/service packages under `app/...`.
- Services should depend on explicit interfaces (store ports) for persistence/external calls.
- Concrete adapters may use Ent/SQL/clients directly, but those details stay behind service dependencies.
- This keeps testability (mocks/fakes) without forcing repository pattern across every feature.

Data ownership rule:

- `db/schema` and `db/migrate/migrations` are monolith-level (workspace-global) data ownership, not per mini-app.
- There is one migration history for the monolith.
- Installable modules must integrate with this single DB history through explicit registration/wiring, not by owning separate histories.

## Refactor Status

Completed in this pass:

- Moved routes from `pkg/routing/routes` to `app/web/controllers`.
- Moved templ views from `templates` to `app/views`.
- Updated imports and test package paths accordingly.

Installable module extraction rule:

A package should be treated as an installable official module only if all are true:

1. It has no hard dependency on `app/web/controllers` or app views/templates.
2. Its config surface is stable and documented.
3. It can be wired through stable interfaces/adapters (no deep app internals required).
4. It has dedicated tests and docs and can be enabled through `ship` install/wire commands.

Module enablement scope:

- module enablement is workspace-global via `config/modules.yaml` (not per mini-app).

## Installable Module Contract (Monolith)

Installable modules must integrate with the monolith through explicit boundaries:

1. Registration boundary:
- modules register routes/jobs/migrations through a narrow registrar API.
- modules must not take `*foundation.Container` directly.
2. Runtime boundary:
- modules receive explicit ports (store/cache/jobs/pubsub/mailer/blob/config/logger) instead of global service locators.
3. Data boundary:
- module schema sources can live in module packages, but migration application is monolith-owned at `db/migrate/migrations`.
- there is one DB and one migration history for the monolith.
4. Policy boundary:
- app-specific business rules are passed as callbacks/interfaces from app domain packages.
- modules must not import app-only domain packages directly.

Classification policy for framework repos/adapters:

- app-specific behavior => move to `app/...` domain packages.
- reusable framework capability => keep/extract as framework module (installable by `ship`).
