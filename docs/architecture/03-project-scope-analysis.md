# Project Scope Analysis
<!-- FRONTEND_SYNC: Landing capability explorer in apps/site/views/web/pages/landing_page.templ links here for Authentication and Authorization, Notifications and Mail, and File Storage. Keep both landing copy and this doc aligned. -->

## What This Project Is

GoShip is a Go + Echo + Templ + HTMX starter application that ships with:

- Session-based authentication and account lifecycle flows
- Profile and onboarding flows
- Email subscriptions and transactional email support
- Subscription billing integration via Stripe
- Notification infrastructure (DB + push + SSE-oriented architecture)
- S3-compatible file storage support with image variants
- Background task processing (Asynq worker)
- Frontend asset bundling for Svelte components and vanilla JS

The repository still carries heritage from a related product domain ("Cherie"), and some feature areas are partially wired or intentionally disabled.

## Runtime Programs

- `cmd/web/main.go`: main HTTP application server
- `cmd/worker/main.go`: asynchronous worker process for task handlers
- `cmd/seed/main.go`: seed runner for test/dev data

## Feature Areas

## 1) Authentication and Account Management

Core flows implemented in routes and services:

- Login/logout (`apps/site/web/controllers/login.go`, `logout.go`)
- Register (`register.go`)
- Forgot/reset password (`forgot_password.go`, `reset_password.go`)
- Email verification (`verify_email.go`)
- Auth middleware and session handling (`apps/site/web/middleware/auth.go`, `apps/site/foundation/auth.go`)

Key implementation choices:

- Cookie-backed session auth using Gorilla sessions via Echo middleware.
- Password reset tokens stored as bcrypt hashes.
- Email verification tokens use JWT signed with app encryption key.

## 2) Onboarding, Preferences, and Profile

- Onboarding and preferences mostly in `apps/site/web/controllers/preferences.go`
- Profile page in `profile.go`
- Mark onboarding completion (`/welcome/finish-onboarding`)
- Profile photo and gallery image routes (`profile_photo.go`, `upload_photo.go`)

## 3) Payments and Subscription Lifecycle

- Stripe checkout + customer portal + webhook in `apps/site/web/controllers/payments.go`
- Local subscription state managed in `apps/site/app/subscriptions/subscriptions.go`
- Product model currently centered on free vs pro (`pkg/domain/enum.go`)

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
- SSE route wiring is currently commented out in router (`sseRoutes` not enabled)

## 5) Email Features

- Newsletter-style email subscription flow (`email_subscribe.go`, `verify_email_subscription.go`)
- Task processor for subscription confirmation emails (`apps/site/jobs/mail.go`)
- Reusable subscription repo module (`pkg/modules/emailsubscriptions`)
- App-specific update email sender integration (`apps/site/app/emailsubscriptions`)
- Mail provider abstraction supports SMTP and Resend (`pkg/repos/mailer`)

## 6) File Storage and Images

- S3-compatible object storage through MinIO client (`pkg/repos/storage/storagerepo.go`)
- DB metadata persisted in `file_storages`
- Image size variants represented by enums and related image size records
- Signed URLs generated for image access

## 7) Background Tasks

Task processors under `apps/site/jobs`:

- Email subscription confirmation
- Email updates
- Subscription deactivation maintenance
- Daily conversation notification orchestration
- Stale notification cleanup

Worker bootstrap and registration in `cmd/worker/main.go`.

## 8) Frontend Delivery Model

- Server-rendered pages via Templ (`apps/site/views/` + `apps/site/web/ui`)
- HTMX-enhanced interactions
- Optional Svelte components bundled into `apps/site/static/svelte_bundle.js`
- Optional vanilla JS bundle into `apps/site/static/vanilla_bundle.js`

Build pipeline:

- JS via `build.mjs` + esbuild
- CSS via Tailwind CLI in Makefile

## Environments and Configuration

Config loading:

- `config/config.go` + layered YAML config (`application`, `environments/*`, `processes`)
- Production can override via env vars (Viper env binding)

Storage modes:

- Embedded SQLite (default in config)
- Standalone Postgres path exists and includes pgvector extension setup

## Testing Surface

- 70+ Go tests in `pkg/**`
- Playwright e2e folder exists (`e2e_tests/`), but specs are currently product-domain stale and marked TODO

## Operational Tooling

- `Makefile` is the primary task runner (init, watch, test, migrations, worker)
- `Procfile` for multi-process dev with Overmind
- Docker Compose currently starts Redis + Mailpit; Postgres service is commented out

## Practical Summary

This codebase is a strong "production-ready starter" foundation with authentication, payments, notifications, storage, and worker primitives. It is also in an active transitional state where some features are scaffolded but not fully wired in the web runtime.
