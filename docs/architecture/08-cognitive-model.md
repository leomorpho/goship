<!-- FRONTEND_SYNC: Keep this cognitive model aligned with app structure and ship doctor checks. -->
# Cognitive Model

GoShip is intentionally optimized for recall, not discovery.

The framework has one canonical flow per concern. If a change does not fit these flows, refactor the change to fit the model instead of introducing a second pattern.

## Request Flow (Always)

`router -> controller -> app/<domain> -> web/viewmodels -> views`

- Routing is declared in `apps/site/router.go`.
- Controllers live in `apps/site/web/controllers`.
- Domain logic lives in `apps/site/app/*`.
- Template payload shapes live in `apps/site/web/viewmodels`.
- Rendering helpers live in `apps/site/web/ui`.
- Templ source lives in `apps/site/views`.

## Async Flow (Always)

`job -> app/<domain> -> adapters`

- Jobs/processors live in `apps/site/jobs`.
- Jobs delegate business logic to `apps/site/app/*`.
- Infrastructure dependencies are provided through the container/adapters.

## Boot Flow (Always)

`foundation -> adapters -> app/web/jobs`

- Composition root is `apps/site/foundation/container.go`.
- App wiring is explicit and deterministic.

## Tooling Rule

Use `ship` as the single mutation interface for common scaffolding.

- `ship make:model`
- `ship make:controller`
- `ship make:resource`
- `ship make:scaffold`
- `ship doctor`

## Non-Negotiable Conventions

- One canonical app root: `apps/site`.
- No legacy paths (`app/goship`, `apps/site/bootstrap`, `apps/site/domains`, etc.).
- Package names match directory intent:
  - `apps/site/web/ui` -> `package ui`
  - `apps/site/web/viewmodels` -> `package viewmodels`
- Route markers in router are preserved for safe generator wiring.
