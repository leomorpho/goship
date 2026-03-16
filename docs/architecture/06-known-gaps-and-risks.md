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

- Default base config now lives in `config/config.go` and can be overridden by `.env`.
- Docker Compose currently starts Redis and Mailpit only; DB service is commented out.
- Make targets include Postgres-dependent commands.

Impact:

- Contributors can experience confusion about canonical local dev DB path.

## 6) Legacy Marketing/Docs UI Artifacts Still Exist In Source (Low)

The public marketing/docs routes were removed, but some related templ source files/components remain in the tree and are no longer part of the active HTTP surface.

Impact:

- Contributors can mistake dead UI assets for active runtime behavior.
- Follow-up cleanup should remove or archive unreferenced page templates.

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

## 9) SQLite-To-Postgres Promotion Is Contracted But Still Manual (Medium)

GoShip now has a documented promotion contract and runtime DB metadata model, but the export/import/verification hooks are not implemented yet.

Impact:

- Promotion currently requires custom/manual tooling around the framework contract.
- Operators can report compatibility and promotion path, but cannot run a one-command migration flow yet.

## 10) AI Provider Coverage Is Still Narrow (Medium)

`modules/ai` now provides a stable app-facing completion boundary with Anthropic, OpenAI, and OpenRouter adapters plus persisted conversation history, but provider coverage is still incomplete and the configured driver becomes unavailable without matching credentials.

Impact:

- Apps can build against the AI seam today, but multi-provider portability is not delivered yet.
- Environments without matching provider credentials will keep a non-nil AI service that returns a clear provider-unavailable error at call time.
- Conversation persistence exists at the module layer, but there is still no first-class app UI for browsing or resuming stored AI threads.

## 11) Domain Events Are In-Process Only (Low)

`framework/events` now covers synchronous in-process publish/subscribe and a jobs enqueue helper, but it does not yet ship a generic async re-dispatch worker or delivery guarantees across processes.

Impact:

- Domain events are reliable inside a single process and easy to test.
- Cross-process fanout still requires explicit jobs or pubsub integration by the caller.

## 12) CSP Is Hardened But Still Allows Script Attributes (Low)

Default security headers and nonce-based CSP are now enabled for dynamic responses, but the default
policy still allows `script-src-attr 'unsafe-inline'` to preserve existing `onload=...` usage in
deferred stylesheet tags.

Impact:

- The app now blocks inline script blocks unless they carry the request nonce.
- Inline script attributes remain permitted until those attributes are removed/refactored.

## 13) Managed Hook Replay Cache Is Process-Local (Medium)

Managed hook signatures now enforce nonce replay protection, but the nonce cache is currently in-memory per process.

Impact:

- Replays are blocked per process instance, but not across independently running replicas.
- Process restarts clear replay history and reopen the short nonce window until entries are rebuilt.

## 14) Soft-Delete Query Guardrail Is Warning-Only (Low)

`ship doctor` now warns (`DX028`) when SQL queries in `db/queries/` reference soft-delete tables
without a `deleted_at` filter, but this remains a warning and not a hard failure.

Impact:

- Teams still need code review discipline for edge-case query intent.
- Unfiltered soft-delete queries are surfaced early but are not blocked at commit time by default.

## 15) Feature Flag Admin Surface Is Toggle-First (Low)

`/auth/admin/flags` now provides list and toggle support for stored flags, but advanced editing
(rollout percentage, targeted user IDs, metadata management) is not exposed in the admin surface yet.

Impact:

- Runtime can evaluate cached DB-backed flags today.
- Operators can quickly enable/disable flags, but still need SQL/seed workflows for richer targeting updates.

## 16) OpenAPI Generation Is Contract-First Baseline (Low)

`ship api:spec` now generates valid OpenAPI 3.0 output from `app/contracts` route comments, but the first pass intentionally keeps response envelopes and security details generic.

Impact:

- API docs are now auto-generated and drift-resistant for route/path/request schema shape.
- Teams still need follow-up refinement for richer per-operation responses/examples/auth scopes when publishing external API docs.

## 17) Factory Scaffolding Is Intentional Baseline (Low)

`ship make:factory` now creates typed test factory scaffolds, but generated files intentionally include only minimal fields and no app-specific trait library.

Impact:

- Teams get consistent factory structure and naming with less boilerplate.
- Rich domain-specific traits still require local customization after scaffold generation.

## 18) HTTP Test Helper Coverage Is Form-Focused Baseline (Low)

`framework/testutil` now covers the most common app-route testing flow (GET + form POST with CSRF, auth session cookie injection, fluent response assertions), but it does not yet include first-class helpers for JSON request bodies, multipart uploads, or websocket/SSE assertions.

Impact:

- Integration tests for HTML/form routes are shorter and less error-prone by default.
- API-heavy and realtime-heavy tests still need some manual request/transport setup.

## 19) I18n Detection Is Runtime Baseline Only (Low)

`modules/i18n` now supports locale files, middleware-based language detection, and CLI locale audits, but the app surface is only partially translated and no persistent profile-language field is wired by default.

Impact:

- Framework/runtime now has a consistent translation seam (`container.I18n.T(...)`) and deterministic request language detection.
- Teams still need to migrate hardcoded strings and optionally wire a real user-preference resolver to get full end-to-end localization coverage.

## Suggested Priority Order

1. Complete cache adapter coverage so page caching works consistently across supported backends.
2. Harden adapter/config validation so realtime capability mismatch is explicit at startup.
3. Re-enable or remove notification-center routes consistently.
4. Refresh e2e tests to match current GoShip flows.
5. Align local stack docs with actual DB mode and compose services.
6. Add shared/distributed replay storage for managed hook nonce tracking in multi-replica deployments.
