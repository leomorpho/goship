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

`GET /auth/realtime` is now conditionally registered based on resolved runtime web features. Invalid runtime-plan resolution or missing realtime dependencies now fail startup instead of silently falling back.

Impact:

- Realtime remains unavailable unless runtime adapters and notifier wiring are configured correctly.
- Misconfigured environments now fail fast during startup rather than appearing healthy with missing realtime endpoints.

## 3) Notification Surface Drift Still Needs Route Inventory Discipline (Low)

The notification module route surface is now wired and canonicalized through shared route-name constants, but future module extractions can still drift if route inventories and view callsites are not kept in sync.

Impact:

- Notification-center behavior is reachable and documented today.
- Future refactors should keep using route-inventory tests plus doc sync to avoid reintroducing hidden route-surface drift.

## 4) Stale/Inconsistent E2E Coverage (Medium)

`tests/e2e/tests/goship.spec.ts` is marked with TODO and contains stale product/domain assumptions.

Impact:

- End-to-end test confidence for current GoShip behavior is limited.

## 5) Dev Runtime Drift Between Config and Docker Compose (Medium)

- Default base config now lives in `config/config.go` and can be overridden by `.env`.
- Docker Compose currently starts Redis and Mailpit only; DB service is commented out.
- Make targets include Postgres-dependent commands.
- `ship dev` auto-mode selection still keys off the jobs adapter instead of the resolved runtime profile, so the canonical single-node app-on loop is not yet enforced by one rule.

Impact:

- Contributors can experience confusion about canonical local dev DB path.
- Local runtime still needs tighter alignment around the single-node default versus the compose-backed workflow, especially in developer docs and automation outside config resolution.

## 6) Residual Dead UI Artifacts Can Reappear Without Reachability Audits (Low)

Known unreachable artifacts from the app-minimalization stream (for example legacy contact/invitations/healthcheck page assets) were deleted, but dead page/template drift remains possible over time without explicit route/callsite audits.

Impact:

- Contributors can still mistake unreferenced UI assets for active runtime behavior if drift accumulates.
- Destructive cleanup streams should keep using route inventory + static reachability proof before deletions.

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

## 16) No Built-In OpenAPI Generation Command (Informational)

The `ship api:spec` command and `app/contracts`-based spec flow were removed in the app-minimalization cleanup stream.

Impact:

- CLI surface is smaller and no longer implies framework-owned OpenAPI generation.
- Teams that need OpenAPI must use an external/spec-specific workflow for now.

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

## 19) I18n Adoption Is Runtime-Ready But App Migration Is Incomplete (Low)

`modules/i18n` now supports locale files (canonical TOML with temporary YAML dual-read), middleware-based language detection, profile-language persistence wiring, and CLI migration/audit commands.

Impact:

- Framework/runtime now has a consistent translation seam (`container.I18n.T(...)`) and deterministic request language detection.
- Teams still need to migrate hardcoded strings and complete string-key coverage to get full end-to-end localization coverage.
- `ship new` scaffolds baseline `en`/`fr` locale files, but production-grade translation completeness still requires follow-up product work per app.

## 20) Doctor Policy Baseline for `tmp/` and Templ Comments (Informational)

`ship doctor` policy is now explicitly aligned with the local dev loop and templ annotation conventions:

- top-level `tmp/` is intentional local dev output (`air` build artifacts) and is allowlisted under `DX013`;
- exported templ functions must include `// Renders:` in the contiguous pre-function comment block (`DX023`), with order-agnostic support when paired with `// Route(s):` and optional blank lines.

Impact:

- running `ship dev` no longer creates a guaranteed `DX013` failure via `tmp/`;
- templ comment annotations are less brittle to harmless ordering/spacing differences while preserving convention clarity.

## 21) Module Isolation Exceptions Still Exist for Installable Modules (Medium)

`DX020` now blocks direct root-package imports from installable module runtime code with no temporary allowlist escape hatch, so the remaining violations are explicit cleanup work instead of tolerated drift.

Impact:

- Installable-module portability remains partially constrained by the remaining violating files, but they are now surfaced as blocking policy failures.
- Structural repo-shape drift for unpaired markers (`DX005`) and raw controller form parsing (`DX027`) is now blocked by default verification instead of surfacing as warnings.

## Suggested Priority Order

1. Complete cache adapter coverage so page caching works consistently across supported backends.
2. Keep tightening adapter/config validation so additional startup capability mismatches fail before route registration.
3. Re-enable or remove notification-center routes consistently.
4. Refresh e2e tests to match current GoShip flows.
5. Align local stack docs with actual DB mode and compose services.
6. Add shared/distributed replay storage for managed hook nonce tracking in multi-replica deployments.
