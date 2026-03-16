# Architecture

## High-Level Layout

The application follows a layered structure:

- `cmd/*`: process entrypoints (`web`, `worker`, `seed`)
- `app/foundation`: app composition container and app-bound infrastructure adapters
- `app/web/controllers`: HTTP handlers
- `app/web/wiring.go`: HTTP stack wiring (middleware/static/deps)
- `app/web/middleware`: auth/session/cache/onboarding/request middleware
- `framework/repos` + `modules/*`: reusable data access and external service adapters
- `app/web/ui`: rendering, page object, redirect helpers
- `app/views`: Templ source files (`.templ`) and generated Go (`gen/*_templ.go`)
- `db/queries` + `db/gen`: Bob SQL query sources and generated DB layer
- `app/jobs`: task processors

## Operating Modes

GoShip is designed to run in two operational contexts without changing app code:

- standalone/self-managed: the app owns its own settings, backup actions, and deployment choices
- externally managed: the app keeps the same runtime capability, but an external authority may provide secrets, managed overrides, and restore orchestration

The runtime itself must stay standalone-capable. External control-plane logic is not a required part of the app architecture.

## Web Runtime Flow

1. `cmd/web/main.go` creates container via `foundation.NewContainer()`.
2. `goship.BuildRouter(c)` is the canonical app router entrypoint and contains the route manifest.
3. Echo server starts with request timeout middleware (SSE-aware).
4. Request path executes middleware chain, route handler, and page rendering.

## Worker Runtime Flow

1. `cmd/worker/main.go` creates app container and validates that the jobs adapter supports a dedicated worker process (e.g., `asynq`). Single-binary mode (`backlite`) runs the dispatcher in-process with the web server instead.
2. Initializes the jobs adapter (driver-specific: Asynq server for Redis-backed; Backlite dispatcher for SQLite-backed).
3. Constructs repo instances needed by task processors.
4. Registers task handlers and starts the worker.

## Container Composition

`app/foundation/container.go` is the core app composition root.

Currently initialized in `NewContainer()`:

- Config
- Validator
- Echo web server + logger
- DB connection
- DB migrations + Bob query generation
- Auth client
- Mail client
- Stripe API key setup

Not currently initialized (commented out):

- Cache client
- Notifier repo
- Task client

This mismatch affects parts of the runtime that assume those dependencies exist. See `known-gaps-and-risks.md`.

## HTTP Middleware Stack

Primary middleware set in `app/web/wiring.go` includes:

- Trailing slash normalization
- Panic recovery
- Security headers
- Request ID
- Gzip
- Structured request logging
- Request timeout (SSE skipped)
- Session middleware
- Authenticated user hydration
- CSRF middleware
- Device type tagging

Additional gatekeepers:

- `RequireAuthentication`
- `RequireNoAuthentication`
- `RedirectToOnboardingIfNotComplete`
- Password token validity loader

## Rendering Model

The UI is server-rendered using Templ components.

- Base page abstraction: `app/web/ui/page.go`
- Render orchestration: `app/web/ui/controller.go`
- Layout wrappers (source): `app/views/web/layouts/*.templ`
- Page components (source): `app/views/web/pages/*.templ`
- Generated packages: `app/views/**/gen/*_templ.go`

HTMX behavior is integrated in the page object (`Page.HTMX`) and controller render logic.

## Data Layer

- Goose SQL migrations (`db/migrate/migrations`) are authoritative for schema changes.
- Bob query files (`db/queries/*.sql`) are authoritative for generated DB access code in `db/gen`.
- Repository packages encapsulate higher-level domain operations.

## Async + Notifications Architecture

- Jobs adapter handles background tasks. Driver is configurable: `asynq` (Redis-backed, separate worker process) or `backlite` (SQLite-backed, runs in-process for single-binary mode).
- Notification system is designed around:
  - persistent DB notifications
  - pub/sub events for SSE
  - push channels (PWA + FCM)
- SSE endpoint (`app/web/controllers/realtime.go`) is registered conditionally based on runtime plan web features (notifier + pubsub dependency availability).

## Frontend Asset Architecture

- `frontend/vite.config.ts` builds the vanilla bundle plus per-island JS chunks from `frontend/islands/`
- The islands runtime loads component JS and CSS on demand via `app/static/islands-manifest.json`
- Islands can be authored in vanilla JS/TS, Svelte, React, or Vue entry files under `frontend/islands/`
- Frontend build output is written to `app/static/`
- Tailwind build pipeline outputs `app/static/styles_bundle.css`

## Deployment/Operations Shape

- Local process orchestration via Overmind (`Procfile*` at project root; scaffolded by `ship new`)
- Docker Compose for Redis + Mailpit in current config
- Kamal deployment files present (`infra/deploy/`, `.kamal/`)
