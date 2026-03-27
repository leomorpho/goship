# Architecture

## High-Level Layout

GoShip is now framework-first. The canonical runtime seams live at repository root and compose framework + modules.

- `container.go`: canonical container construction seam
- `router.go`: canonical HTTP route registration seam
- `schedules.go`: canonical cron/schedule registration seam
- `cmd/*`: process entrypoints (`web`, `worker`, `seed`, `cli`)
- `framework/*`: framework runtime, web stack, adapters, templates, repositories
- `modules/*`: installable capabilities (for example auth, profile, notifications, paid subscriptions)
- `db/queries` + `db/gen`: Bob SQL sources and generated query layer
- `config/*`: runtime config schema and module/profile toggles
- `frontend/*`, `styles/*`, `static/*`: frontend toolchain and emitted assets

## Runtime Seam Contract

Root runtime seams are the only canonical mutation points for app-runtime wiring:

1. `container.go` composes the bootstrap container and schedule callback.
2. `router.go` registers static routes, middleware stacks, public/auth/external routes, and runtime-gated realtime routes.
3. `schedules.go` registers recurring job enqueue callbacks.

Any generated or module wiring that touches runtime composition must preserve these seams.

## Operating Modes

GoShip runs in two contexts without changing framework code:

- standalone/self-managed: runtime owns settings and lifecycle directly
- externally managed: runtime remains authoritative for capability execution while accepting signed managed hooks/overrides

The runtime must remain standalone-capable even when managed integrations are enabled.

## Web Runtime Flow

1. `cmd/web/main.go` creates the container via `goship.NewContainer()`.
2. `router.go` executes `goship.BuildRouter(...)` to register the full route graph.
3. Echo starts with the canonical middleware stack and framework error handler.
4. Requests run through middleware, controller handlers, and server-rendered templ output.

## Worker Runtime Flow

1. `cmd/worker/main.go` creates container + module service graph.
2. Jobs backend is selected by runtime plan (`asynq` or `backlite`).
3. Job handlers are registered and worker lifecycle starts.
4. Scheduler lifecycle is driven through the same root seam registration contract.

## Container Composition

`container.go` delegates to `framework/bootstrap` and exposes the typed container surface used by route/controller/module wiring.

Core initialized dependencies include:

- config + validation
- Echo server + structured logging
- DB + migration/query runtime integration
- cache/jobs/pubsub/storage/mailer adapters (runtime-plan dependent)
- module services surfaced to route wiring

## HTTP Middleware Stack

`router.go` applies canonical middleware through framework wiring:

- security headers
- panic recovery
- request ID + request logging
- compression + timeout behavior (SSE-aware)
- session/auth context middleware
- CSRF and request guards

Additional route groups apply auth and managed-hook verification gates where required.

## Rendering Model

Rendering is server-first via templ + framework page/viewmodel helpers.

- page/controller primitives: `framework/http/ui/*`
- controllers: `framework/http/controllers/*`
- viewmodels: `framework/http/viewmodels/*`
- templ sources + generated output: `framework/http/*` and `framework/views/*`

## Data and Async Model

- schema history is authoritative in `db/migrate/migrations`
- query contracts are authoritative in `db/queries/*.sql`
- generated DB access is in `db/gen`
- recurring jobs are registered in `schedules.go` and dispatched through `core.Jobs`

## Frontend and Assets

- Vite config lives in `frontend/vite.config.ts`
- islands sources live in `frontend/islands/`
- compiled static outputs are written to `static/`
- framework design tokens/recipes are emitted through the styles pipeline
