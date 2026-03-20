# App Layer Guide

## Role

`app/` wires the framework and installed modules together. It owns the application composition
root, router, app-specific controllers, and foundation setup.

Do not move framework concerns into `app/`, and do not put module-owned logic here.

## Placement Rules

- Controllers live in `app/web/controllers/`.
- Route registration belongs only in `app/router.go`, at the `ship:routes:*` marker comments.
- Container wiring belongs only in `app/foundation/container.go`, at the `ship:container:*`
  marker comments.
- Views belong in `app/views/`.

## UI Rules

Follow the shared `UI_CONVENTION.md` at the monorepo root for `data-component`, `data-slot`, and
`Renders:` comment rules.

## Anti-Patterns

- No business logic in controllers; delegate to services or module APIs.
- No SQL in controllers; go through repositories or module storage interfaces.
- Do not add framework-level abstractions in `app/`.

## Verification

Run `ship verify` after every change.
