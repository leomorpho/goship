<!-- FRONTEND_SYNC: Keep this cognitive model aligned with app structure and ship doctor checks. -->
# Cognitive Model

GoShip is intentionally optimized for recall, not discovery.

The framework has one canonical flow per concern. If a change does not fit these flows, refactor the change to fit the model instead of introducing a second pattern.

## Request Flow (Always)

`router -> controller -> app/<domain> -> web/viewmodels -> views`

- Routing is declared in `apps/goship/router.go`.
- Controllers live in `apps/goship/web/controllers`.
- Domain logic lives in `apps/goship/app/*`.
- Template payload shapes live in `apps/goship/web/viewmodels`.
- Rendering helpers live in `apps/goship/web/ui`.
- Templ source lives in `apps/goship/views`.

## Async Flow (Always)

`job -> app/<domain> -> adapters`

- Jobs/processors live in `apps/goship/jobs`.
- Jobs delegate business logic to `apps/goship/app/*`.
- Infrastructure dependencies are provided through the container/adapters.

## Boot Flow (Always)

`foundation -> adapters -> app/web/jobs`

- Composition root is `apps/goship/foundation/container.go`.
- App wiring is explicit and deterministic.

## Tooling Rule

Use `ship` as the single mutation interface for common scaffolding.

- `ship make:model`
- `ship make:controller`
- `ship make:resource`
- `ship make:scaffold`
- `ship doctor`

## Non-Negotiable Conventions

- One canonical app root: `apps/goship`.
- No legacy paths (`app/goship`, `apps/goship/bootstrap`, `apps/goship/domains`, etc.).
- Package names match directory intent:
  - `apps/goship/web/ui` -> `package ui`
  - `apps/goship/web/viewmodels` -> `package viewmodels`
- Route markers in router are preserved for safe generator wiring.
