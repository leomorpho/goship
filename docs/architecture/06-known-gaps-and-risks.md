# Known Gaps and Risks
<!-- FRONTEND_SYNC: Landing capability explorer in app/views/web/pages/landing_page.templ links here for Events and Realtime. Keep both landing copy and this doc aligned. -->

This list is based on direct code inspection and is intended to guide contributor priorities.

## 1) Cache Adapter Coverage Is Incomplete (Medium)

`app/foundation/container.go` now initializes the legacy repo-level cache client only when the
selected cache adapter is Redis. That avoids nil-pointer panics and accidental Redis dials in the
default `memory` cache profile, but it also means page-cache middleware remains unavailable until a
memory-backed cache implementation exists behind the same seam.

Impact:

- Local/default runtime does not currently exercise page-cache paths.
- Cache behavior differs between `memory` and `redis` adapter selections beyond pure backend choice.

## 2) Realtime Dependency Requirements (High for distributed realtime features)

`GET /auth/realtime` is now conditionally registered based on resolved runtime web features. If notifier/pubsub dependencies are unavailable, realtime routes are intentionally disabled.

Impact:

- Realtime is unavailable unless runtime adapters are configured correctly.
- Misconfigured environments can appear healthy while silently missing realtime endpoints.

## 3) Notification Center Endpoints Partially Disabled (Medium)

Route handlers exist in `app/web/controllers/notifications.go`, but several are commented out during route wiring.

Impact:

- Notification center behavior is incomplete from an HTTP exposure perspective.
- Some notifier capabilities are not reachable from active routes.

## 4) Stale/Inconsistent E2E Coverage (Medium)

`tests/e2e/tests/goship.spec.ts` is marked with TODO and contains stale product/domain assumptions.

Impact:

- End-to-end test confidence for current GoShip behavior is limited.

## 5) Dev Runtime Drift Between Config and Docker Compose (Medium)

- Default base config currently keeps embedded SQLite defaults in `config/application.yaml`.
- Docker Compose currently starts Redis and Mailpit only; DB service is commented out.
- Make targets include Postgres-dependent commands.

Impact:

- Contributors can experience confusion about canonical local dev DB path.

## 6) In-App Docs Are Present But Sparse (Low)

`/docs/*` pages exist, but architecture/getting-started sections are mostly placeholders.

Impact:

- Existing user-facing docs routes do not currently reflect true implementation depth.

## 7) Some Feature Paths Still Use Placeholder Data (Low)

Example: home feed button counts are hardcoded in `app/web/controllers/home_feed.go`.

Impact:

- UI may represent scaffolding rather than production data behavior in some sections.

## 8) Docker-heavy integration tests tax iteration speed (Medium)

- Many integration packages drop into `testcontainers-go` and spin up Docker containers every time `make test-integration` runs.
- That end-to-end surface is important, but it makes the default integration loop take several minutes, especially for the `app/foundation`/`app/profile` packages.
- Where possible, migrate the behavior being exercised (SQL migrations, module wiring, notification/email flows) into smaller unit tests that mock infrastructure so contributors can run fast, deterministic builds without constantly creating/tearing down containers.
- Until those migrations happen, keep per-package tooling (like the new `INTEGRATION_PARALLEL` runner) and reusable helper contexts so we can run the heaviest suites less frequently.
- Priority candidate: `app/profile/profile_test.go` currently spins up Postgres+pgvector containers for every happy-path scenario. Replace most of those cases with SQLite-backed unit tests (or mocks) that exercise `ProfileService` logic via `dbgen` while keeping a single Postgres/pgvector test for schema wiring.

## Suggested Priority Order

1. Complete cache adapter coverage so page caching works consistently across supported backends.
2. Harden adapter/config validation so realtime capability mismatch is explicit at startup.
3. Re-enable or remove notification-center routes consistently.
4. Refresh e2e tests to match current GoShip flows.
5. Align local stack docs with actual DB mode and compose services.
