<!-- FRONTEND_SYNC: Keep this cognitive model aligned with app structure and ship doctor checks. -->
# Cognitive Model

GoShip is intentionally optimized for recall, not discovery.

The framework has one canonical flow per concern. If a change does not fit these flows, refactor the change to fit the model instead of introducing a second pattern.

## Request Flow (Always)

`router -> controller -> app/<domain> -> web/viewmodels -> views`

- Routing is declared in `app/router.go`.
- Controllers live in `app/web/controllers`.
- Domain logic lives in `app/*`.
- Template payload shapes live in `app/web/viewmodels`.
- Rendering helpers live in `app/web/ui`.
- Templ source lives in `app/views`.

## Async Flow (Always)

`job -> app/<domain> -> adapters`

- Jobs/processors live in `app/jobs`.
- Jobs delegate business logic to `app/*`.
- Infrastructure dependencies are provided through the container/adapters.

## Boot Flow (Always)

`foundation -> adapters -> app/web/jobs`

- Composition root is `app/foundation/container.go`.
- App wiring is explicit and deterministic.

## Tooling Rule

Use `ship` as the single mutation interface for common scaffolding.

- `ship make:model`
- `ship make:controller`
- `ship make:resource`
- `ship make:scaffold`
- `ship doctor`

## Non-Negotiable Conventions

- One canonical app root: `app`.
- No legacy paths (`app/goship`, `app/bootstrap`, `app/domains`, etc.).
- Package names match directory intent:
  - `app/web/ui` -> `package ui`
  - `app/web/viewmodels` -> `package viewmodels`
- Route markers in router are preserved for safe generator wiring.
