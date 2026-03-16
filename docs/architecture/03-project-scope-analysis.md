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
- Product model currently centered on free vs pro (`framework/domain/enum.go`)

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

- Some notification endpoints are active (count endpoint, permission/subscription management)
- Several notification-center routes are currently commented out in router wiring
- SSE route wiring is runtime-gated and only enabled when notifier/pubsub dependencies are available

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
- `app/foundation.Container` exposes `EventBus`
- Auth flows publish shared events such as `UserRegistered`, `UserLoggedIn`, `UserLoggedOut`, and `PasswordChanged`

## 9) Frontend Delivery Model

- Server-rendered pages via Templ (`app/views/` + `app/web/ui`)
- HTMX-enhanced interactions
- Optional client islands built as per-component JS chunks under `app/static/islands/`
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
- Embedded Backlite queue monitoring.
- Managed settings status page at `/auth/admin/managed-settings` for operator visibility into effective value/source/access state.
- Built-in Templ components for common admin UI patterns.

## Environments and Configuration

Config loading:

- `config/config.go` with struct-tagged env loading via `cleanenv`
- Local `.env` and shell env vars are the application config source of truth

Storage modes:

- Embedded SQLite (default in config)
- Standalone Postgres path exists and includes pgvector extension setup

Runtime DB metadata contract:

- `config.Config.RuntimeMetadata()` now provides a normalized DB metadata snapshot for status/reporting surfaces.
- Metadata includes DB mode/driver, migration tracking table, portability profile, and SQLite-to-Postgres compatibility path (`sqlite-to-postgres-manual-v1`).
- `config.Config.ManagedSettingStatuses()` provides normalized managed-setting access states for settings/admin surfaces.
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
- Playwright e2e folder exists (`tests/e2e/`), but specs are currently product-domain stale and marked TODO

## Operational Tooling

- `Makefile` is the primary task runner (init, watch, test, migrations, worker)
- `Procfile*` at project root for multi-process dev with Overmind (generated by `ship new`)
- Docker Compose currently starts Redis + Mailpit; Postgres service is commented out

## Practical Summary

This codebase is a strong "production-ready starter" foundation with authentication, payments, notifications, storage, and worker primitives. It is also in an active transitional state where some features are scaffolded but not fully wired in the web runtime.
