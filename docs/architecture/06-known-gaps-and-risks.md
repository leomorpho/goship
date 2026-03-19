# Known Gaps and Risks
<!-- FRONTEND_SYNC: Landing capability explorer in app/views/web/pages/landing_page.templ links here for Events and Realtime. Keep both landing copy and this doc aligned. -->

This list is based on direct code inspection and is intended to guide contributor priorities.

## 1) Cache Adapter Coverage Is Incomplete (Medium)

`app/foundation/container.go` now initializes the shared cache seam for both in-memory (`otter` /
`memory`) and Redis-backed adapters, and the cache contract is now pinned by one shared backend
matrix. Grouped objects, tag invalidation, TTL expiry, and the raw byte/prefix seam now run through
the same executable coverage for both adapters.

Impact:

- Local/default runtime can exercise page-cache paths through the in-memory adapter.
- Positive cache TTLs are normalized to second precision so `memory` and `redis` expire entries on the same schedule.

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

## Route Composition Style Still Needs Ongoing Discipline (Low)

`app/router.go` now owns one canonical static-registration path through `appweb.RegisterStaticRoutes(c)`,
and `BuildRouter` fails fast when the app container is nil. The remaining risk is drift from future
route extractions if new registration styles are introduced outside that composition root.

Impact:

- The route manifest and static asset surface are centralized today.
- Future route extraction work should keep using route-composition contract tests to avoid reintroducing split registration styles.

## Startup and Route Contract Suites Now Cover Failure Paths (Low)

Startup failure cases now have a dedicated contract suite covering nil containers, missing module
dependencies, invalid runtime profiles, and realtime capability mismatches. Route inventory output
also has explicit public/auth contract coverage over `method`, `path`, `auth`, `handler`, and
`file` metadata.

Impact:

- Startup regressions now fail through one executable contract instead of being spread across ad hoc tests.
- CLI and agent consumers of `ship routes --json` now have an explicit grouped-route surface contract.

## Canonical Docs Now Enforce Hard-Cut Wording (Low)

The canonical docs set is now guarded by doctor policy checks that reject selected legacy-cutover
phrases in architecture, CLI, and roadmap docs, with file:line diagnostics and an allowlist for
intentional historical references outside the canonical set. The remaining risk is scope drift if
new docs are added to the canonical set without extending that policy.

Impact:

- The canonical docs now describe the current runtime model directly instead of migration-era ambiguity.
- Future canonical-doc additions should extend the doctor check so the hard-cut wording rule stays comprehensive.

## 5) Dev Runtime Drift Between Config and Docker Compose (Medium)

- Default base config now lives in `config/config.go` and can be overridden by `.env`.
- Docker Compose currently starts Redis and Mailpit only; DB service is commented out.
- Makefile help and contributor docs now describe the same single-node default plus optional accessory-services contract.

Impact:

- The default local story is now consistent: app-on single-node first, with optional Redis/Mailpit compose accessories.
- Residual compose-backed Postgres helper commands remain optional escape hatches rather than the default local contract.

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

GoShip now surfaces the promotion contract through `ship db:promote` and runtime DB metadata. The command now applies the canonical `.env` mutation step for SQLite-to-Postgres promotion (`standard` profile plus `db=postgres cache=redis jobs=asynq`) and supports `--dry-run` / `--json` previews of that exact mutation plan. The CLI still exposes separate `ship db:export --json`, `ship db:import --json`, and `ship db:verify-import --json` hooks for the remaining export/import workflow.
The underlying import/verification engine is still manual-first, so the next step is wiring those hooks to the actual framework path instead of just surfacing the CLI contract.

Impact:

- Promotion still requires manual export/import orchestration around the framework contract.
- Operators can apply the canonical config flip with `ship db:promote`, but the data move and post-import verification are not a one-command migration flow yet.

## 10) AI Provider Coverage Is Still Narrow (Medium)

`modules/ai` now provides a stable app-facing completion boundary with Anthropic, OpenAI, and OpenRouter adapters plus persisted conversation history, but provider coverage is still incomplete and the configured driver becomes unavailable without matching credentials.

Impact:

- Apps can build against the AI seam today, but multi-provider portability is not delivered yet.
- Environments without matching provider credentials will keep a non-nil AI service that returns a clear provider-unavailable error at call time.
- Conversation persistence exists at the module layer, but there is still no first-class app UI for browsing or resuming stored AI threads.

## 11) Runtime Capability Reporting Is Now Canonicalized (Low)

`ship runtime:report --json` now exposes the effective profile, adapters, process plan, web
features, DB runtime metadata, and managed-key sources in one machine-readable payload. The
INT1-01 will extend that payload with a versioned handshake envelope for orchestration preflight.
Managed runtime metadata now carries explicit registry/schema version identifiers so runtime and
control-plane consumers can agree on the authoritative key mapping. The remaining risk is payload
drift if future runtime metadata is added without extending the report.

INT1-02 will add a named orchestration contract-mismatch preflight gate on top of `ship verify`
so unsupported deploy/upgrade/promote combinations fail before the orchestration layer starts.

Managed settings now expose drift and rollback-target metadata for managed overrides, but the
remaining risk is UI/operator workflow drift if future settings surfaces fail to render that
metadata consistently.

Impact:

- Operators and agents now have one canonical CLI report for runtime capability inspection.
- Future runtime metadata additions should extend the report and its contract tests in the same change stream.

## Upgrade Readiness Report Still Needs a Blocker Schema (Low)

`ship upgrade` still needs a machine-readable readiness report that carries the upgrade target,
blocking conditions, and remediation hints in a shared blocker schema. The remaining risk is that
upgrade orchestration tooling cannot yet preflight version bumps with one canonical contract.

Impact:

- Operators will have a single preflight report for upgrade safety once the contract lands.
- Future upgrade automation should extend the same blocker schema instead of inventing a parallel plan format.

## Cross-Lane Dependency Matrix Still Needs A Single Contract Map (Low)

The GoShip, Interaction, and Control-plane ticket streams still rely on multiple docs and ticket
parents to express sequencing. The cross-lane dependency matrix should explicitly name the
must-finish-before contract map so ticket ordering stays deterministic.
This cross-lane dependency matrix language is the canonical coordination contract for the docs lane.

This cross-lane dependency matrix contract should stay named exactly that way in the canonical docs
so coordination and verification tooling can grep for one phrase.

Impact:

- Ticket sequencing remains readable, but it still depends on documentation discipline.
- Future coordination tickets should extend the same matrix language instead of inventing a parallel map.

## 12) Domain Events Are In-Process Only (Low)

`framework/events` now covers synchronous in-process publish/subscribe and a jobs enqueue helper, but it does not yet ship a generic async re-dispatch worker or delivery guarantees across processes.

Impact:

- Domain events are reliable inside a single process and easy to test.
- Cross-process fanout still requires explicit jobs or pubsub integration by the caller.

## 13) CSP Is Hardened But Still Allows Script Attributes (Low)

Default security headers and nonce-based CSP are now enabled for dynamic responses, but the default
policy still allows `script-src-attr 'unsafe-inline'` to preserve existing `onload=...` usage in
deferred stylesheet tags.

Impact:

- The app now blocks inline script blocks unless they carry the request nonce.
- Inline script attributes remain permitted until those attributes are removed/refactored.

## 14) Managed Hook Replay Cache Is Process-Local (Medium)

Managed hook signatures now expose a shared/distributed replay-store contract, but the active runtime still uses the default in-memory nonce store until app wiring adopts a shared backend.

Impact:

- Replays are blocked per process instance, but not across independently running replicas.
- Process restarts clear replay history and reopen the short nonce window until entries are rebuilt.

## 15) Soft-Delete Query Guardrail Is Warning-Only (Low)

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

## 16) Quality CLI Surface Is Now Canonicalized (Low)

The duplicate `ship check` path has been removed, and the top-level quality contract now funnels
fast checks through `ship test` and tiered repository verification through
`ship verify --profile fast|standard|strict`.

Impact:

- Humans and agents now have one canonical top-level command path per quality concern plus an
  explicit verify tier selection.
- Future quality-loop changes should update `ship test`, `ship verify`, and their contract docs/tests in the same change stream.

## 17) Job Generator Contract Is Now Canonicalized (Low)

`ship make:job <Name>` now exposes a canonical scaffold contract for app jobs built around
`core.Jobs` and `core.JobHandler`.

Impact:

- App teams can generate a consistent `app/jobs` baseline instead of hand-rolling job files and tests.
- Future job-surface changes should extend the generator, jobs guide, and CLI contract together so the scaffold stays aligned with the runtime seam.

## 18) Mailer Generator Contract Is Now Canonicalized (Low)

`ship make:mailer <Name>` now exposes a canonical scaffold contract for templ email views and
preview wiring under the existing `/dev/mail/*` surface.

## 19) Island Generator Contract Is Now Canonicalized (Low)

`ship make:island <Name>` now exposes a canonical frontend island scaffold contract built around
`frontend/islands/<Name>.js` and `app/views/web/components/<name>_island.templ`. The scaffold
locks the current runtime seam in place: generated island modules export `mount(el, props)`, and
the templ mount component renders the matching `data-island` / `data-props` target.

Intentional manual follow-up remains explicit:

- `ship templ generate --file app/views/web/components/<name>_island.templ`
- `make build-js`
- render the generated `@components.<Name>Island(...)` seam from the page/component that should host the island

Impact:

- Teams can generate a consistent email-template baseline instead of hand-rolling view and preview wiring.
- Future mailer-surface changes should extend the generator, preview controller, and CLI docs together so generated mailers stay aligned with the dev preview contract.

## 19) Generator Output Drift Is Reduced But Matrix Coverage Must Keep Up (Low)

`ship make:*` commands now share one stdout contract built around `Created:`, `Updated:`,
`Preview:`, and `Next:` sections, and the generators package carries a consolidated idempotency
matrix suite.

Impact:

- Humans and agents can parse generator results through one predictable report shape instead of
  generator-specific wording.
- Future generators still need to join the shared output contract and matrix tests when new scaffold
  surfaces are added.

## 19) Task-Oriented OSS Workflow Docs Are Now Canonicalized (Low)

GoShip now has a canonical OSS workflow-docs set for adding an endpoint, adding a module, and
adding a background job, with those guides surfaced from the docs hub and playbook.

Impact:

- New contributors and agents now have one indexed workflow set for common extension tasks.
- Future contributor workflows should extend the canonical guide set and the docs index/playbook together.

## 20) Shared-Infra Adoption Trend Reporting Is Now Canonicalized (Low)

`ship describe --pretty` now exposes a shared-infra adoption summary showing shared module adoption
alongside app-owned controller/job/command counts.

Impact:

- Module upstreaming decisions now have one machine-readable baseline metric.
- Future shared-infra shifts should extend the describe payload and the scope/CLI docs in the same change stream.

## 21) Pagoda Intake Governance Is Now Canonicalized (Low)

GoShip now documents a recurring Pagoda intake cadence and exposes a canonical adopt/adapt/skip log.

Impact:

- Upstream review now has an explicit recurring process instead of memory-based drift.
- Contributors have one canonical place to record intake outcomes and follow-up actions.

## 22) Module Install/Remove Is Now Deterministic For Standalone Batteries (Low)

`ship module:add` and `ship module:remove` now manage local standalone-battery wiring through
structured `go.mod` / `go.work` edits plus `config/modules.yaml` and marker-based app snippets.
The new `modules/storage` battery joins the first-class installable set around the canonical
`core.BlobStorage` seam.

Impact:

- Installable batteries now have one deterministic local-workspace wiring path instead of ad hoc
  manual `go.mod` / `go.work` edits.
- Remove flows now fail with exact blocker file paths when app code still imports a module,
  preventing partial unwiring.

## 23) Generator Drift Is Now Guarded By A Dedicated CI Contract Lane (Low)

GoShip now carries a dedicated generator contract lane that combines per-generator golden
snapshots for the shared report shape with the consolidated generator idempotency matrix.

Impact:

- Merge protection now catches generator output drift separately from the broader backend test lane.
- Local reruns can isolate the duplicate-generation matrix through `make test-generator-idempotency`
  without conflating it with snapshot refresh work.
- Intentional report-format changes require an explicit snapshot refresh (`UPDATE_GENERATOR_SNAPSHOTS=1`)
  and snapshot commit instead of silently changing CI expectations.

## 24) Managed Backup/Restore Contract Is Now More Explicit (Low)

The managed backup/restore seam now locks `backup-manifest-v1` to SQLite-first metadata and a
strict SHA-256 checksum contract, and managed restore responses now return machine-readable
`restore_evidence` with an explicit accepted-manifest field.

Impact:

- Control-plane and runtime integrations now have one explicit manifest/evidence shape instead of
  inferring restore success from a bare `accepted` status.
- Future backup schema changes should extend the typed contract and evidence payloads together
  rather than widening the current v1 semantics silently.

## 25) Alpha Surface Freeze Is Now Explicitly Gated (Low)

GoShip now freezes the `v0.1.0-alpha` public surface through snapshots of root CLI help and route
inventory, with a dedicated CI lane and an explicit approved-review refresh path.

Impact:

- Public command and route drift now fails in a named lane instead of surfacing later as release-note
  cleanup or downstream integration breakage.
- Intentional alpha-surface changes require an explicit snapshot refresh and review trail, which
  keeps the freeze policy visible instead of implicit.

## 26) Core CLI Output Contracts Now Carry Golden Coverage (Low)

GoShip now snapshots stable command surfaces for `doctor`, `routes`, `describe`, and `verify`
through in-package golden tests.

Impact:

- Human and JSON output drift for the main operator-facing CLI surfaces now shows up as direct test
  failures instead of review-only formatting regressions.
- Intentional output changes require an explicit `UPDATE_CLI_GOLDENS=1` refresh path, keeping
  snapshot updates deliberate.

## 27) Doc-Sync And Dead-Route Guardrails Are Now Named CI Lanes (Low)

GoShip now runs separate CI lanes for route/scope doc sync and dead-route regression around the
canonical route inventory.

Impact:

- Stale docs such as removed route entries now fail in a named lane instead of lingering until
  humans notice the mismatch.
- Route-inventory regressions and doc drift are isolated from broader backend failures, which makes
  remediation faster and more obvious.

## 28) Module Compatibility Policy Is Now Enforced By Verify (Low)

GoShip now treats installable standalone batteries as one canonical contract: if root `go.mod`
depends on a local battery, that dependency must stay on `v0.0.0`, point at the repo-local
`replace`, and keep the matching `go.work use` entry. Module-local `go.mod` files must also keep
the declared module path aligned with the battery catalog.

Impact:

- Cherie-facing module version policy is now executable instead of living only in roadmap notes.
- Local battery drift now fails in `ship verify` before it becomes downstream release confusion.

## 29) Canonical Architecture Docs Now Share The Hard-Cut Lint (Low)

The hard-cut wording lint now treats the canonical architecture set as one contract surface instead
of checking only the scope/routes subset.

Impact:

- Historical wording drift in the primary architecture docs now fails through the same `DX030`
  path as CLI and roadmap docs.
- Historical references remain possible outside the canonical set through the existing allowlist,
  keeping migration notes separate from the current architecture contract.

## 30) Verify And Doctor Now Expose The No-Compatibility/No-Deprecation Policy Explicitly (Low)

The existing hard-cut wording checks are now named and reported as an explicit
no-compatibility/no-deprecation invariant in the `ship verify` step output and `DX030`
diagnostics.

Impact:

- Operator-facing quality output now describes the policy directly instead of hiding it behind
  generic wording.
- Reviewers can distinguish this policy from other doc lint failures without reading the
  implementation details first.

## 31) Starter Bootstrap Budget Is Now A Named Regression Gate (Low)

GoShip now carries a dedicated bootstrap budget check for the canonical starter flow: scaffold a
fresh app with `ship new`, then prove the generated `cmd/web` entrypoint runs within the named CI
budget.

Impact:

- Starter DX regressions now show up in a specific lane instead of surfacing later as anecdotal
  "ship new feels slower" feedback.
- The budget and rerun knob are explicit, which keeps runner variance discussions separate from
  actual bootstrap regressions.

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

## 22) Module Isolation Is Now Guarded By A Dedicated CI Lane (Low)

The repo now exposes module isolation as a named CI lane via `make test-module-isolation`. The gate
reports offending module/file context for direct root imports and rejects stale entries in the
allowlist, while SQL portability remains tracked by its separate dedicated lane.

Impact:

- installable-module boundary regressions now fail in a named suite with actionable diagnostics;
- allowlist entries can be trimmed as soon as temporary exceptions are removed, so drift is visible instead of silently accumulating;
- SQL portability (`sql-core-v1`) continues to have one canonical CI entrypoint, which keeps future automation and downstream reuse predictable. That lane now checks the runtime metadata contract plus branch annotations and placeholder conventions in the canonical migration/query SQL sources, so portability drift fails with file/query diagnostics instead of generic test noise.

## 23) Cherie Compatibility Smoke Coverage Is Still Generic (Medium)

CI currently runs one generic Playwright smoke spec for startup and basic app serving, but it does not yet expose a dedicated Cherie-oriented compatibility lane for the downstream-critical boot, auth, and realtime path trio.

Impact:

- framework changes can keep the generic smoke green while still drifting from the narrower downstream compatibility baseline;
- Cherie-specific upgrade confidence remains weaker than the roadmap policy requires until the smoke baseline is named and enforced in CI.

## 24) Extension Zones Vs Protected Contract Zones Are Not Codified In One Manifest (Medium)

The repo has boundary rules and marker checks, but it still lacks one canonical manifest that says where app/framework consumers can diverge freely and which contract seams are protected by doctor enforcement.

Impact:

- contributors must infer safe customization boundaries from several docs and policy checks instead of one source of truth;
- new guardrails risk growing inconsistently because protected seams are not documented and enforced from the same manifest.

## 25) Standalone Exportability Lacks A Named Verification Gate (Medium)

The repo documents standalone capability, but `ship verify` does not yet expose a dedicated run-anywhere/exportability gate proving generated apps stay free of control-plane dependency assumptions.

Impact:

- standalone regression risk is spread across implicit docs and ad hoc review rather than one explicit release gate;
- generated-app exportability can drift without a named verification step catching it early.

## Suggested Priority Order

1. Complete cache adapter coverage so page caching works consistently across supported backends.
2. Keep tightening adapter/config validation so additional startup capability mismatches fail before route registration.
3. Re-enable or remove notification-center routes consistently.
4. Refresh e2e tests to match current GoShip flows.
5. Align local stack docs with actual DB mode and compose services.
6. Add shared/distributed replay storage for managed hook nonce tracking in multi-replica deployments.
7. Publish shared signature vectors and a canonical payload library for the INT2 bridge so signing fixtures do not drift between runtime and control-plane consumers.
