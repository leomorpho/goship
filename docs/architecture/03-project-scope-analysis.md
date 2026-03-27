# Project Scope Analysis
<!-- FRONTEND_SYNC: Landing capability explorer in framework/http/pages/landing_page.templ links here for Authentication and Authorization, Notifications and Mail, and File Storage. Keep both landing copy and this doc aligned. -->

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
- Framework-owned design tokens compiled to CSS variables and recipe classes through the Vite/Tailwind asset pipeline

GoShip is maintained as a framework-first repository: canonical runtime seams live at repo root (`container.go`, `router.go`, `schedules.go`) with framework/modules/tooling packages underneath.

`ship describe --pretty` also exposes a shared-infra adoption summary so contributors can inspect
installed shared modules versus runtime-owned controller/job/command counts as a non-blocking
upstreaming metric.

`ship runtime:report --json` now also carries per-module adoption metadata so orchestration tooling
can inspect installed module identity, source, and version without parsing repo internals.
The payload also includes a stable first-party baseline with `installed=false` for non-enabled
first-party modules so adoption comparisons stay consistent across runtimes.

`ship profile:set <single-binary|standard|distributed>` now rewrites the local `.env` runtime
profile and process preset values deterministically, giving the CLI a first-class way to move
between the canonical topology modes.

`ship adapter:set <db|cache|jobs|pubsub|storage|mailer>=<impl>...` now rewrites canonical adapter
env vars in the local `.env` file and rejects invalid runtime selections before they can drift into
an unsupported runtime plan.

`ship db:promote [--dry-run] [--json]` now turns the first SQLite-to-Postgres promotion step into an
executable config change by applying the canonical `.env` mutation set for the standard profile and
Postgres/Redis/Asynq adapters, exposes `promotion-state-machine-v1`, and blocks repeated promotion
attempts once the runtime is already in a partial post-config state; export/import/verification
remain explicit manual follow-up steps.
`ship db:promote` now publishes the canonical manual runbook doc path (`docs/guides/14-sqlite-to-postgres-promotion-runbook.md`) in both text and JSON outputs so operators and agents can continue with one deterministic checklist.

`ship verify` and `ship upgrade` now reject unsupported contract-version identifiers up front, reusing
the same runtime (`runtime-contract-v1`) and upgrade (`upgrade-readiness-v1`) tokens that their
machine-readable JSON surfaces already emit.

`ship new` now ships a runnable fresh-app confidence loop by default: the scaffold includes a local
`.env`, a starter SQLite migration, a minimal web entrypoint with readiness/rendered-route smoke
surface, matching `static/` and `styles/` assets, and a trivial worker entrypoint so downstream apps
can immediately run `ship db:migrate`, `go run ./cmd/web`, and `ship verify --profile fast`.

## Runtime Programs

- `cmd/web/main.go`: main HTTP application server
- `cmd/worker/main.go`: asynchronous worker process for task handlers
- `cmd/seed/main.go`: seed runner for test/dev data
- `cmd/cli/main.go`: repository command runner for local runtime utilities
- `container.go`: framework-first runtime seam for container construction
- `router.go`: framework-first runtime seam for route and middleware registration
- `schedules.go`: framework-first runtime seam for recurring schedule registration

## Feature Areas

## 1) Authentication and Account Management

Core flows implemented in routes and services:

- Login/logout (`framework/http/controllers` auth routes)
- OAuth/social login for enabled GitHub, Google, and Discord providers (`modules/auth`)
- Optional TOTP-based two-factor authentication with recovery backup codes (`modules/2fa`)
- Register (`register.go`)
- Forgot/reset password (`forgot_password.go`, `reset_password.go`)
- Email verification (`verify_email.go`)
- Auth middleware and session handling (`framework/http/middleware/auth.go`, framework bootstrap wiring)

Key implementation choices:

- Cookie-backed session auth using Gorilla sessions via Echo middleware.
- Password reset tokens stored as bcrypt hashes.
- Email verification tokens use JWT signed with app encryption key.
- OAuth account linking reuses existing user records when the provider email already exists and stores provider access tokens encrypted at rest in `oauth_accounts`.
- Two-factor authentication defers full session creation behind a short-lived signed pending-login cookie and validates either TOTP codes or one-time backup codes.

## 2) Onboarding, Preferences, and Profile

- Onboarding and preferences mostly in `framework/http/controllers/preferences_*.go`
- Profile page in `profile.go`
- Mark onboarding completion (`/welcome/finish-onboarding`)
- Profile photo and gallery image routes (`profile_photo.go`, `upload_photo.go`)
- Preferences now include a runtime settings control panel that surfaces managed-capable keys with explicit state:
  - `editable` (standalone mode)
  - `read-only` (managed mode, locally locked)
  - `externally-managed` (managed override applied)

## 3) Payments and Subscription Lifecycle

- Stripe checkout + customer portal + webhook in framework route/controller wiring plus `modules/paidsubscriptions`
- Local subscription state is handled by paid subscriptions module services and runtime jobs wiring
- Product plans are catalog-driven through runtime composition with module/service predicates handling paid/free branching without fixed key assumptions.

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

- Notification-center routes are owned by `modules/notifications/routes` and wired through the canonical root `router.go` seam
- Active surface includes notification list, unread-count badge, mark-all-read, mark read/unread, delete, and onboarding subscription management endpoints
- SSE route wiring is runtime-gated and only enabled when notifier/pubsub dependencies are available
- Invalid realtime/runtime-plan startup combinations now fail fast instead of silently falling back to a reduced route surface
- CI now carries a dedicated Cherie compatibility smoke baseline for `/up`, `/user/login`, and auth-gated `/auth/realtime` behavior

## 5) Email Features

- Newsletter-style email subscription flow (`email_subscribe.go`, `verify_email_subscription.go`)
- Task processor for subscription confirmation emails (runtime jobs wiring through `core.Jobs`)
- Reusable subscription repo module (`modules/emailsubscriptions`)
- Runtime integration wiring for module services (root seams + framework bootstrap)
- Mail provider abstraction supports SMTP and Resend (`framework/repos/mailer`)
- Email templates render via templ components in `framework/views/emails/`, with `framework/repos/mailer.RenderEmail` producing both HTML and plain-text fallback output

## 6) File Storage and Images

- S3-compatible object storage through MinIO client (`framework/repos/storage/storagerepo.go`)
- DB metadata persisted in `file_storages`
- Image size variants represented by enums and related image size records
- Signed URLs generated for image access

## 7) Background Tasks

Task processors are wired through the runtime jobs adapter:

- Email subscription confirmation
- Email updates
- Subscription deactivation maintenance
- Daily conversation notification orchestration
- Stale notification cleanup

Worker bootstrap and registration in `cmd/worker/main.go`.
Cron schedule registration is runtime-owned in `schedules.go`; callbacks enqueue jobs
through `core.Jobs`, and the worker runtime starts/stops the scheduler lifecycle.

**Supported Backends:**
- **Asynq:** Distributed task processing via Redis. Requires a separate worker process.
- **Backlite:** SQLite-backed task processing for single-binary or zero-dependency modes. Runs in-process with the web server, removing the need for a separate worker.

## 8) Domain Events

- `framework/events` now provides a typed in-process event bus with generic subscription helpers
- `framework/events` also exposes a jobs-backed async bridge that decodes a typed envelope and republishes supported shared events into the local bus
- `container.go` bootstraps a container that exposes `EventBus`
- Auth flows publish shared events such as `UserRegistered`, `UserLoggedIn`, `UserLoggedOut`, and `PasswordChanged`

## 9) Frontend Delivery Model

- Server-rendered pages via Templ (`framework/http/*`, `framework/views/*`, `framework/http/ui`)
- HTMX-enhanced interactions
- Optional client islands built as per-component JS chunks under `frontend/islands/` and emitted to `static/`
- `ship make:island <Name>` generates the canonical island pair (`frontend/islands/<Name>.js` plus framework templ mount scaffolding) that the browser runtime mounts through `data-island` / `data-props`
- Optional vanilla JS bundle is emitted under `static/`
- Public demo route `/demo/islands` that renders regression-guarded counter islands for vanilla JS, React, Vue, and Svelte

Build pipeline:

- JS via `frontend/vite.config.ts` + Vite
- CSS tokens and recipe classes live in `styles/styles.css` + `styles/tailwind_components.css`, with Vite smoke coverage proving the emitted starter CSS bundle contains the framework-owned token surface

Blessed split-frontend contract:

- `api-only-same-origin-sveltekit-v1` is the only explicit external-frontend contract today.
- Browser auth writes stay on `same-origin auth/session` with `cookie/CSRF` protections enabled.
- Supported custom frontend scope is intentionally constrained to `SvelteKit-first` until additional contracts are hardened.

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
- Feature flags page at `/auth/admin/flags` with server-side toggle actions plus registry-backed metadata (constant key badge, code default indicator, canonical description) and orphaned DB-row warnings for keys no longer defined in code.
- Trash page at `/auth/admin/trash` for operator visibility into soft-deleted row counts by table.
- Built-in Templ components for common admin UI patterns.
- Playwright baseline smoke coverage lives at `tests/e2e/tests/admin_scaffold.spec.ts` and protects the auth/managed-settings/flags/trash flow.

## 12) Feature Flags

- `modules/flags` provides a DB-backed flag service with optional cache acceleration.
- `container.Flags` exposes flag evaluation (`Enabled`) to application code.
- Rollout checks are deterministic per `(flag key, user id)` hash bucket.

## 13) Request DTO Conventions

- Controller request DTOs now live with their owning controllers/modules instead of global shared DTO buckets.
- `ship doctor` (`DX027`) enforces typed request binding patterns and now blocks raw/untyped form parsing patterns in controllers.
- OpenAPI generation was removed from the core `ship` CLI surface in the minimal runtime cleanup stream.
- Canonical page ownership is framework-first (`framework/http/ui.Page` + `framework/http/page.Base`).

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
- `ship runtime:report --json` emits the canonical machine-readable runtime capability payload from config/runtime-plan metadata, including active profile, adapters, process plan, web features, DB runtime metadata, managed-key sources, current framework version, per-module adoption metadata, and a versioned handshake envelope.
- Managed mode runtime reports now include explicit upgrade-readiness blockers when authority or managed-hook signing prerequisites are missing.
- Runtime reports now include backup contract metadata for managed consumers (`backup-manifest-v1` plus restore-evidence schema hints, including canonical `record_links` field names).
- Runtime reports now include a decision-input contract envelope (`decision-input-contract-v1`) with runtime/upgrade contract versions plus rollout/promotion schema identifiers while keeping rollout orchestration outside GoShip runtime code.
- The same runtime report is the runtime-side input to `staged-rollout-decision-v1`, where external rollout tooling must preserve the runtime facts it consumed plus the approved `policy_input_version` instead of inventing a second runtime-specific decision payload.
- The same runtime report now carries a divergence classification contract (`divergence-classification-v1`) plus escalation policy (`divergence-escalation-v1`) so operators can distinguish ordinary extension-zone drift from protected-contract drift and repeated divergence that should be upstreamed or recovered before rollout.
- Metadata includes DB mode/driver, migration tracking table, portability profile, and SQLite-to-Postgres compatibility path (`sqlite-to-postgres-manual-v1`).
- Managed runtime metadata now carries shared registry/schema version identifiers (`managed-key-registry-v1`, `managed-key-schema-v1`) so runtime and control-plane consumers can agree on the managed-key contract.
- `config.Config.ManagedSettingStatuses()` provides normalized managed-setting access states for settings/admin surfaces, including managed-override drift and rollback-target metadata.
- Managed mode now includes a signed control-plane bridge at `/managed/status`, `/managed/backup`, and `/managed/restore`.
- Managed hook verification is configurable through `PAGODA_MANAGED_HOOKS_SECRET`, `PAGODA_MANAGED_HOOKS_MAX_SKEW_SECONDS`, and `PAGODA_MANAGED_HOOKS_NONCE_TTL_SECONDS`.
- Managed backup/restore responses now accept and echo optional `record_links` (`incident_id`, `recovery_id`, `deploy_id`) so runtime evidence can be correlated with external incident and recovery records without coupling the runtime to control-plane storage.

Security baseline:

- Dynamic web responses now include default security headers via `framework/middleware/security_headers.go`.
- CSP is nonce-based per request (`csp_nonce`) and rendered templates consume the nonce from request context.
- `PAGODA_SECURITY_HEADERS_*` (and `SECURITY_HEADERS_*` aliases) control enablement, HSTS, and full CSP override.
- Local/dev defaults include Vite HMR websocket allowances for `localhost:5173`.

Health endpoints:

- `GET /health` is now a JSON liveness endpoint (`{"status":"ok"}` when process is running).
- `GET /health/ready` is a JSON readiness endpoint that runs registered dependency checks (DB, cache, jobs inspector, required runtime env contract) and returns `503` when any critical check fails.
- The framework bootstrap container now owns the default health registry, and framework web wiring registers `/up`, `/health`, and `/health/ready` without app-specific route assembly.
- Startup now validates the framework health contract (`db`, `cache`, `jobs`, `env` checks present) and fails fast when misconfigured rather than silently serving incomplete readiness behavior.
- Startup also validates the framework-default runtime env contract (`PAGODA_APP_ENVIRONMENT`, adapter selectors, and DB location/host fields by DB mode) so missing required runtime variables fail before serving traffic.
- Startup validation failures now emit a deterministic health startup summary (`required`, `registered`, `missing`, `ready`) so operators can diagnose misconfiguration quickly.

## Testing Surface

- Go tests are distributed across `app/**`, `framework/**`, `modules/**`, and `tools/**`
- `framework/factory` + `tests/factories` now provide a typed baseline for building and inserting repeatable test data records
- `framework/testutil` now provides typed HTTP test helpers (`NewTestServer`, `PostForm` with automatic CSRF, `PostJSON`, `PostMultipart`, `AsUser`, fluent response assertions including `AssertSSEEvent`) for app-route integration tests
- Playwright e2e folder exists (`tests/e2e/`), but specs are currently product-domain stale and marked TODO

## Operational Tooling

- `Makefile` is the primary task runner (init, watch, test, migrations, worker)
- `Procfile*` at project root for multi-process dev with Overmind (generated by `ship new`)
- Docker Compose currently starts Redis + Mailpit; Postgres service is commented out

## Practical Summary

This codebase is a strong "production-ready starter" foundation with authentication, payments, notifications, storage, and worker primitives. Some features remain intentionally scaffolded or partial, but the documented model is the current canonical runtime, not a migration bridge.
