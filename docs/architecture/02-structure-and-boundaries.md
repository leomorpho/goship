# Structure and Boundaries

This document defines where code belongs as GoShip evolves into a Rails-like framework plus example app.

## Current Top-Level Shape

- `app/goship/`: app-specific code for the first-party GoShip app
- `cmd/`: runnable entrypoints (`web`, `worker`, `seed`, CLI)
- `pkg/`: reusable framework-level libraries and adapters
- `config/`: runtime configuration
- `ent/`: schema and generated ORM
- `docs/`: internal design and implementation documentation

## App vs Framework Rules

Use this placement rule for every new file:

- Put code in `app/goship/...` when it encodes product behavior/UI for the GoShip app.
- Put code in `pkg/...` when it is reusable as framework infrastructure across apps.

## Web Layer Layout

App web code is now app-scoped:

- `app/goship/web/routes`: route composition + handlers
- `app/goship/views`: templ components/layouts/pages/emails

Router source of truth:

- `app/goship/web/routes/router.go`

## Refactor Status

Completed in this pass:

- Moved routes from `pkg/routing/routes` to `app/goship/web/routes`.
- Moved templ views from `templates` to `app/goship/views`.
- Updated imports and test package paths accordingly.

Still intentionally centralized (next phase):

- `pkg/repos`
- `pkg/services`

These remain framework-level until each package is classified as either:

- app-specific (move under `app/goship/...`), or
- reusable framework module (stay in `pkg/...` or move to future dedicated framework modules).

