# Project Scope Analysis
<!-- FRONTEND_SYNC: Landing capability explorer in app/views/web/pages/landing_page.templ links here for Authentication and Authorization, Notifications and Mail, and File Storage. Keep both landing copy and this doc aligned. -->

## What This Project Is

GoShip is a Go + Echo + Templ + HTMX starter application that ships with:

- Session-based authentication and account lifecycle flows
- Profile and onboarding flows
- Email subscriptions and transactional email support
- Subscription billing integration via Stripe
- Notification infrastructure (DB + push + SSE-oriented architecture)
- S3-compatible file storage support with image variants
- Background task processing (Asynq worker)
- Provider-agnostic AI completion service with an Anthropic adapter
- Frontend asset bundling for vanilla JS plus Svelte/React/Vue islands

GoShip is maintained as a single-app repository: one canonical runtime app under `app/` + `cmd/`, plus framework/modules/tooling packages.

`ship describe --pretty` also exposes a shared-infra adoption summary so contributors can inspect
installed shared modules versus app-owned controller/job/command counts as a non-blocking
upstreaming metric.

`ship profile:set <single-binary|standard|distributed>` now rewrites the local `.env` runtime
profile and process preset values deterministically, giving the CLI a first-class way to move
between the canonical topology modes.

`ship adapter:set <db|cache|jobs|pubsub|storage|mailer>=<impl>...` now rewrites canonical adapter
env vars in the local `.env` file and rejects invalid runtime selections before they can drift into
an unsupported runtime plan.

`ship db:promote [--dry-run] [--json]` now turns the first SQLite-to-Postgres promotion step into an
executable config change by applying the canonical `.env` mutation set for the standard profile and
Postgres/Redis/Asynq adapters; export/import/verification remain explicit manual follow-up steps.

## Runtime Programs

- `cmd/web/main.go`: main HTTP application server
- `cmd/worker/main.go`: asynchronous worker process for task handlers
- `cmd/seed/main.go`: seed runner for test/dev data
- `cmd/cli/main.go`: app-level command runner (`app/commands/*`)

## Feature Areas

## 1) Authentication and Account Management

Core flows implemented in routes and services:

- Login/logout (`app/web/controllers/login.go`, `logout.go`)
- OAuth/social login for enabled GitHub, Google, and Discord providers (`modules/auth`)
- Optional TOTP-based two-factor authentication with recovery backup codes (`modules/2fa`)
- Register (`register.go`)
- Forgot/reset password (`forgot_password.go`, `reset_password.go`)
- Email verification (`verify_email.go`)
- Auth middleware and session handling (`app/web/middleware/auth.go`, `app/foundation/auth.go`)

Key implementation choices:

- Cookie-backed session auth using Gorilla sessions via Echo middleware.
- Password reset tokens stored as bcrypt hashes.
- Email verification tokens use JWT signed with app encryption key.
- OAuth account linking reuses existing user records when the provider email already exists and stores provider access tokens encrypted at rest in `oauth_accounts`.
- Two-factor authentication defers full session creation behind a short-lived signed pending-login cookie and validates either TOTP codes or one-time backup codes.

## 2) Onboarding, Preferences, and Profile

- Onboarding and preferences mostly in `app/web/controllers/preferences.go`
- Profile page in `profile.go`
- Mark onboarding completion (`/welcome/finish-onboarding`)
- Profile photo and gallery image routes (`profile_photo.go`, `upload_photo.go`)
- Preferences now include a runtime settings control panel that surfaces managed-capable keys with explicit state:
  - `editable` (standalone mode)
  - `read-only` (managed mode, locally locked)
  - `externally-managed` (managed override applied)

## 3) Payments and Subscription Lifecycle

- Stripe checkout + customer portal + webhook in `app/web/controllers/payments.go`
- Local subscription state is handled by the paid subscriptions module and app jobs (`modules/paidsubscriptions`, `app/jobs/subscriptions.go`)
- Product plans are catalog-driven from app runtime composition (`app/foundation/subscription_catalog.go`), with module/service predicates handling paid/free branching without fixed key assumptions.

Webhook flow currently handles:

- `customer.subscription.created`
- `customer.subscription.updated`
- `customer.subscription.deleted`

## 4) Notifications and Realtime Capabilities

Implemented infrastructure includes:

- Notification domain and storage
- Notification permissions by platform (push, fcm_push, email, sms)
- PWA and FCM push subscription storage/sending
- SSE pub/sub abstractions

Status of exposure:

- Notification-center routes are owned by `modules/notifications/routes` and wired through the canonical app router
- Active surface includes notification list, unread-count badge, mark-all-read, mark read/unread, delete, and onboarding subscription management endpoints
- SSE route wiring is runtime-gated and only enabled when notifier/pubsub dependencies are available
- Invalid realtime/runtime-plan startup combinations now fail fast instead of silently falling back to a reduced route surface
- CI now carries a dedicated Cherie compatibility smoke baseline for `/up`, `/user/login`, and auth-gated `/auth/realtime` behavior

## 5) Email Features

- Newsletter-style email subscription flow (`email_subscribe.go`, `verify_email_subscription.go`)
- Task processor for subscription confirmation emails (`app/jobs/mail.go`)
- Reusable subscription repo module (`modules/emailsubscriptions`)
- App integration wiring for module services (`app/web/wiring.go`, `app/jobs/mail.go`)
- Mail provider abstraction supports SMTP and Resend (`framework/repos/mailer`)
- Email templates render via templ components in `app/views/emails/`, with `framework/repos/mailer.RenderEmail` producing both HTML and plain-text fallback output

## 6) File Storage and Images

- S3-compatible object storage through MinIO client (`framework/repos/storage/storagerepo.go`)
- DB metadata persisted in `file_storages`
- Image size variants represented by enums and related image size records
- Signed URLs generated for image access

## 7) Background Tasks

Task processors under `app/jobs`:

- Email subscription confirmation
- Email updates
- Subscription deactivation maintenance
- Daily conversation notification orchestration
- Stale notification cleanup

Worker bootstrap and registration in `cmd/worker/main.go`.
Cron schedule registration is app-owned in `app/schedules/schedules.go`; callbacks enqueue jobs
through `core.Jobs`, and the worker runtime starts/stops the scheduler lifecycle.

**Supported Backends:**
- **Asynq:** Distributed task processing via Redis. Requires a separate worker process.
- **Backlite:** SQLite-backed task processing for single-binary or zero-dependency modes. Runs in-process with the web server, removing the need for a separate worker.

## 8) Domain Events

- `framework/events` now provides a typed in-process event bus with generic subscription helpers
- `framework/events` also exposes a jobs-backed async bridge that decodes a typed envelope and republishes supported shared events into the local bus
- `app/foundation.Container` exposes `EventBus`
- Auth flows publish shared events such as `UserRegistered`, `UserLoggedIn`, `UserLoggedOut`, and `PasswordChanged`

## 9) Frontend Delivery Model

- Server-rendered pages via Templ (`app/views/` + `app/web/ui`)
- HTMX-enhanced interactions
- Optional client islands built as per-component JS chunks under `app/static/islands/`
- `ship make:island <Name>` generates the canonical island pair (`frontend/islands/<Name>.js` plus `app/views/web/components/<name>_island.templ`) that the browser runtime mounts through `data-island` / `data-props`
- Optional vanilla JS bundle into `app/static/vanilla_bundle.js`
- Public demo route `/demo/islands` that renders regression-guarded counter islands for vanilla JS, React, Vue, and Svelte

Build pipeline:

- JS via `frontend/vite.config.ts` + Vite
- CSS via Tailwind CLI in Makefile

## 10) AI Integration

- `modules/ai` exposes a provider-agnostic completion boundary via `container.AI`
- Anthropic, OpenAI, and OpenRouter are wired providers
- The service supports unary completion, token streaming, and structured JSON decoding into Go types
- The module also includes a persistence layer for storing conversation threads and message history in `ai_conversations` / `ai_messages`
- Non-production builds expose an authenticated `/auth/ai-demo` page that demonstrates HTMX + SSE streaming against the AI service

## 11) Admin Panel

Reflection-based administrative interface for managing database resources.

- Resource registration and CRUD operations for Bob-generated models.
- Queue monitoring reads through `core.JobsInspector`, with unsupported backends surfacing an explicit unavailable state in the admin UI.
- Managed settings status page at `/auth/admin/managed-settings` for operator visibility into effective value/source/access state.
- Feature flags page at `/auth/admin/flags` with server-side toggle actions.
- Trash page at `/auth/admin/trash` for operator visibility into soft-deleted row counts by table.
- Built-in Templ components for common admin UI patterns.
- Playwright baseline smoke coverage lives at `tests/e2e/tests/admin_scaffold.spec.ts` and protects the auth/managed-settings/flags/trash flow.

## 12) Feature Flags

- `modules/flags` provides a DB-backed flag service with optional cache acceleration.
- `container.Flags` exposes flag evaluation (`Enabled`) to application code.
- Rollout checks are deterministic per `(flag key, user id)` hash bucket.

## 13) Request DTO Conventions

- Controller request DTOs now live with their owning controllers/modules instead of a global `app/contracts` package.
- `ship doctor` (`DX027`) enforces typed request binding patterns and now blocks raw/untyped form parsing patterns in controllers.
- OpenAPI generation was removed from the core `ship` CLI surface in the app-minimalization cleanup stream.
- `app/controller` was removed; canonical app page ownership is `app/web/ui.Page`, which now embeds reusable framework-owned base fields/behavior from `framework/web/page.Base`.

## 14) Internationalization Baseline

- `modules/i18n` provides a locale service exposed through the `core.I18n` seam (`container.I18n`) and request middleware for language detection.
- Language detection priority is: `?lang=<code>` query parameter, authenticated user profile preference, `lang` cookie, `Accept-Language` header, then default locale (`en`).
- Query-driven language switches now persist the normalized language to the authenticated profile record (`profiles.preferred_language`) and still set the `lang` cookie.
- Canonical locale sources are TOML files in `locales/` (`*.toml`, dotted keys such as `auth.login.title`); runtime still dual-reads YAML during migration windows.
- Runtime toggle: `PAGODA_I18N_ENABLED=false` disables i18n service initialization (safe English fallback path, no startup panic).
- Runtime default language is configurable via `PAGODA_I18N_DEFAULT_LANGUAGE`.
- Runtime enforcement mode is configurable via `PAGODA_I18N_STRICT_MODE=off|warn|error` and consumed by `ship doctor` (`DX029`), with optional `.i18n-allowlist` (stable `I18N-S-*` selectors preferred; legacy `path:line` compatibility retained).
- Canonical coverage scope, exclusions, and command-to-enforcement mapping are documented in `docs/guides/10-i18n-llm-migration-workflow.md`.
- `ship make:locale <code>` scaffolds a new locale file from `locales/en.toml` (falls back to legacy YAML source when migrating).
- `ship i18n:init` bootstraps baseline locale files (`en.toml`, `fr.toml`) for apps that started without i18n and prints a deterministic migration command loop.
- `ship i18n:scan --format json` emits deterministic diagnostics for hardcoded user-facing literals (`--paths`, `--limit` supported) without failing on findings; Go sources are scanned via AST/token positions with guardrails for logs, SQL literals, and `_test.go` files, and islands scanning is scoped to `frontend/islands/` (`.js`, `.ts`, `.jsx`, `.tsx`, `.svelte`, `.vue`) entry paths.
- `ship i18n:instrument` provides a deterministic migration plan from scanner findings; `--apply` currently rewrites high-confidence Go controller `*.String` literals (for example `c.String`, `ctx.String`) into i18n calls and seeds missing keys in baseline locale catalogs.
- `ship i18n:migrate` converts legacy YAML locale catalogs to canonical TOML catalogs.
- `ship i18n:normalize` rewrites TOML catalogs to deterministic canonical ordering for stable diffs.
- `ship i18n:compile` generates typed key artifacts from baseline locale catalogs for Go (`app/i18nkeys/keys_gen.go`) and islands TypeScript (`frontend/islands/i18n-keys.ts`).
- `ship i18n:ci` provides a deterministic strict-mode CI gate (scanner + doctor `DX029`) for i18n-enabled apps.
- `ship i18n:missing` reports missing/empty translations versus English source keys and emits plural/select completeness diagnostics for `I18n.TC(...)`/`I18n.TS(...)` usage.
- `ship i18n:unused` reports locale keys not referenced in `.go`/`.templ` `I18n.T(...)` usage.
- Islands bootstrap contract now passes locale/i18n data through `data-props` as `{ i18n: { locale: "<code>", messages: { ... } } }`, and islands resolve labels from that payload for hydration parity.
- `framework/api` now includes localized error helpers (`NotFoundLocalized`, `UnauthorizedLocalized`, `ValidationLocalized`) that keep machine `code` values stable while localizing human `message` values from request locale context.
- Navbar now includes the `language-switcher` component, and switch links preserve current route/query while toggling `lang`.
- Core HTML layouts/pages now bind `<html lang>` from request locale (`page.Language()`), so rendered document language tracks locale switches (`?lang=...`, cookie/header fallback chain).
- `ship new <app>` now supports i18n-aware scaffold startup: interactive prompt (or `--i18n` / `--no-i18n`) and optional starter locale file creation (`locales/en.toml`, `locales/fr.toml`), with explicit messaging that i18n can be enabled/migrated later.

## Environments and Configuration

Config loading:

- `config/config.go` with struct-tagged env loading via `cleanenv`
- Local `.env` and shell env vars are the application config source of truth

Storage modes:

- Embedded SQLite (default in config)
- Standalone Postgres path exists and includes pgvector extension setup

Runtime DB metadata contract:

- `config.Config.RuntimeMetadata()` now provides a normalized DB metadata snapshot for status/reporting surfaces.
- `ship db:export --json` exposes a structured SQLite export report with a typed `backup-manifest-v1` payload, checksum evidence, suggested next commands, and a planning-only note for agents/tooling.
- `ship runtime:report --json` emits the canonical machine-readable runtime capability payload from config/runtime-plan metadata, including active profile, adapters, process plan, web features, DB runtime metadata, managed-key sources, a versioned handshake envelope, and the `managed-divergence-v1` classification contract for rollback vs repeated upstream-module-candidate escalation.
- Metadata includes DB mode/driver, migration tracking table, portability profile, and SQLite-to-Postgres compatibility path (`sqlite-to-postgres-manual-v1`).
- Managed runtime metadata now carries shared registry/schema version identifiers (`managed-key-registry-v1`, `managed-key-schema-v1`) so runtime and control-plane consumers can agree on the managed-key contract.
- Managed divergence output now classifies drifted managed keys and records `immediate_action=rollback` plus `repeated_action=upstream-module-candidate-review` so downstream tooling can distinguish one-off recovery from repeat-escalation workflow.
- `config.Config.ManagedSettingStatuses()` provides normalized managed-setting access states for settings/admin surfaces, including managed-override drift and rollback-target metadata.
- Managed mode now includes a signed control-plane bridge at `/managed/status`, `/managed/backup`, and `/managed/restore`.
- Managed hook verification is configurable through `PAGODA_MANAGED_HOOKS_SECRET`, `PAGODA_MANAGED_HOOKS_MAX_SKEW_SECONDS`, and `PAGODA_MANAGED_HOOKS_NONCE_TTL_SECONDS`.

Security baseline:

- Dynamic web responses now include default security headers via `framework/middleware/security_headers.go`.
- CSP is nonce-based per request (`csp_nonce`) and rendered templates consume the nonce from request context.
- `PAGODA_SECURITY_HEADERS_*` (and `SECURITY_HEADERS_*` aliases) control enablement, HSTS, and full CSP override.
- Local/dev defaults include Vite HMR websocket allowances for `localhost:5173`.

Health endpoints:

- `GET /health` is now a JSON liveness endpoint (`{"status":"ok"}` when process is running).
- `GET /health/ready` is a JSON readiness endpoint that runs registered dependency checks (DB, cache, jobs inspector) and returns `503` when any critical check fails.

## Testing Surface

- Go tests are distributed across `app/**`, `framework/**`, `modules/**`, and `tools/**`
- `framework/factory` + `tests/factories` now provide a typed baseline for building and inserting repeatable test data records
- `framework/testutil` now provides typed HTTP test helpers (`NewTestServer`, `PostForm` with automatic CSRF, `AsUser`, fluent response assertions) for app-route integration tests
- Playwright e2e folder exists (`tests/e2e/`), but specs are currently product-domain stale and marked TODO

## Operational Tooling

- `Makefile` is the primary task runner (init, watch, test, migrations, worker)
- `Procfile*` at project root for multi-process dev with Overmind (generated by `ship new`)
- Docker Compose currently starts Redis + Mailpit; Postgres service is commented out

## Practical Summary

This codebase is a strong "production-ready starter" foundation with authentication, payments, notifications, storage, and worker primitives. Some features remain intentionally scaffolded or partial, but the documented model is the current canonical runtime, not a migration bridge.
