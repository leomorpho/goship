# Known Gaps and Risks
<!-- FRONTEND_SYNC: Landing capability explorer in apps/site/views/web/pages/landing_page.templ links here for Events and Realtime. Keep both landing copy and this doc aligned. -->

This list is based on direct code inspection and is intended to guide contributor priorities.

## 1) Container Initialization Mismatch (High)

In `apps/site/foundation/container.go`, `NewContainer()` does not call:

- `initCache()`
- `initNotifier()`
- `initTasks()`

Yet runtime code assumes these dependencies exist in multiple places, and `Shutdown()` calls `c.Tasks.Close()` and `c.Cache.Close()` unconditionally.

Impact:

- Potential nil-pointer panics on shutdown or during feature paths that rely on tasks/notifier/cache.
- Web and worker behavior can diverge from intended architecture.

## 2) Realtime Dependency Requirements (High for distributed realtime features)

`GET /auth/realtime` is now conditionally registered based on resolved runtime web features. If notifier/pubsub dependencies are unavailable, realtime routes are intentionally disabled.

Impact:

- Realtime is unavailable unless runtime adapters are configured correctly.
- Misconfigured environments can appear healthy while silently missing realtime endpoints.

## 3) Notification Center Endpoints Partially Disabled (Medium)

Route handlers exist in `apps/site/web/controllers/notifications.go`, but several are commented out during route wiring.

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

Example: home feed button counts are hardcoded in `apps/site/web/controllers/home_feed.go`.

Impact:

- UI may represent scaffolding rather than production data behavior in some sections.

## Suggested Priority Order

1. Fix container/service initialization and safe shutdown semantics.
2. Harden adapter/config validation so realtime capability mismatch is explicit at startup.
3. Re-enable or remove notification-center routes consistently.
4. Refresh e2e tests to match current GoShip flows.
5. Align local stack docs with actual DB mode and compose services.
