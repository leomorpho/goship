# Architecture
<!-- FRONTEND_SYNC: Landing capability explorer in app/views/web/pages/landing_page.templ links here for Validation and Forms, Jobs and Scheduling, and Events and Realtime. Keep both landing copy and this doc aligned. -->

## High-Level Layout

The application follows a layered structure:

- `cmd/*`: process entrypoints (`web`, `worker`, `seed`)
- `app/foundation`: app composition container and app-bound infrastructure adapters
- `app/web/controllers`: HTTP handlers
- `app/web/wiring.go`: HTTP stack wiring (middleware/static/deps)
- `pkg/middleware`: auth/session/cache/onboarding/request middleware
- `pkg/repos`: data access and external service adapters
- `app/web/ui`: rendering, page object, redirect helpers
- `app/views`: Templ source files (`.templ`) and generated Go (`gen/*_templ.go`)
- `ent`: schema + generated ORM
- `pkg/tasks`: Asynq task processors

## Web Runtime Flow

1. `cmd/web/main.go` creates container via `foundation.NewContainer()`.
2. `goship.BuildRouter(c)` is the canonical app router entrypoint and contains the route manifest.
3. Echo server starts with request timeout middleware (SSE-aware).
4. Request path executes middleware chain, route handler, and page rendering.

## Worker Runtime Flow

1. `cmd/worker/main.go` creates app container and validates that jobs adapter is `asynq` via `c.Config`.
2. Starts Asynq server from cache config and builds router (for reverse route URLs in tasks).
3. Constructs repo instances needed by task processors.
4. Registers handlers on Asynq mux and runs worker.

## Container Composition

`app/foundation/container.go` is the core app composition root.

Currently initialized in `NewContainer()`:

- Config
- Validator
- Echo web server + logger
- DB connection
- Ent ORM
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

- Ent ORM (`ent`) is authoritative for schema and query generation.
- Schema create/migrate is invoked in app startup via `c.ORM.Schema.Create(...)`.
- Repository packages encapsulate higher-level domain operations.

## Async + Notifications Architecture

- Asynq handles background jobs with Redis backend.
- Notification system is designed around:
  - persistent DB notifications
  - pub/sub events for SSE
  - push channels (PWA + FCM)
- SSE endpoint (`app/web/controllers/realtime.go`) is registered conditionally based on runtime plan web features (notifier + pubsub dependency availability).

## Frontend Asset Architecture

- `frontend/build.mjs` bundles Svelte entrypoints under `frontend/javascript/svelte/*.js`
- Also bundles vanilla JS from `frontend/javascript/vanilla/main.js`
- Outputs static bundles and meta files in `app/static/`
- Tailwind build pipeline outputs `app/static/styles_bundle.css`

## Deployment/Operations Shape

- Local process orchestration via Overmind (`Procfile*` at project root; scaffolded by `ship new`)
- Docker Compose for Redis + Mailpit in current config
- Kamal deployment files present (`infra/deploy/`, `.kamal/`)
