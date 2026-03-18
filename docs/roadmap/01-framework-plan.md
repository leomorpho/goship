# GoShip Framework Plan

This document tracks the plan to evolve GoShip from a starter kit into a Rails-like Go framework with strong developer ergonomics.

## Terminology Note

Voice-to-text aliases used in discussion:

- `shiri` => `Cherie`
- `jerry` => `Cherie`
- `sherry` => `Cherie`

When planning or tracking work, treat all of the above as `Cherie`.

## Vision

Build a highly productive, convention-first Go framework where developers can:

1. ship fast with sensible defaults;
2. install batteries as versioned modules;
3. choose deployment/runtime mode without rewriting app code.

## Documentation Source Of Truth

Implementation and architecture documentation lives in `docs/`.

Primary files for ongoing refactor work:

1. `docs/architecture/02-structure-and-boundaries.md` (canonical placement rules)
2. `docs/architecture/01-architecture.md` (runtime/system behavior)
3. `docs/guides/01-ai-agent-guide.md` (agent execution conventions)
4. `docs/reference/01-cli.md` (living `ship` CLI contract)

Documentation execution priority:

1. Docs-first is preferred over MCP/tooling-first in the near term.
2. `ship-mcp` remains a future extension point, not a primary workflow dependency today.
3. Build task-focused how-to guides that make common changes fast and repeatable.

Primary framing:

- GoShip aims to be a Ruby on Rails alternative in Go, with comparable batteries-included productivity and developer ergonomics.
- Product aspiration: be a deeply loved framework by developers by putting developer joy, speed, and clarity first.
- Rails inspiration applies to the entire framework experience (not only environment settings): app structure, conventions, generators, batteries, defaults, testing, and deployment workflows.

### Rails-Inspired Framework Pillars

1. Convention over configuration:
- opinionated defaults for layout, modules, and naming;
- explicit escape hatches when teams need control.
2. Generators-first workflow:
- create apps/resources/jobs/modules with minimal manual wiring.
3. Batteries-included, modular delivery:
- auth, billing, notifications, storage, admin, jobs as installable modules.
4. Stable abstractions over pluggable implementations:
- developers code to GoShip interfaces; adapters are swap-friendly.
5. Excellent local DX:
- fast feedback, minimal setup, stateless tests by default.
6. Production-ready path:
- clear migration from local single-process to distributed process topology.

Rails inspiration for configuration and "what runs":

1. Rails uses layered configuration:
- `config/application.rb` for global app config;
- `config/environments/*.rb` for per-environment overrides;
- `config/initializers/*.rb` for subsystem wiring;
- Gemfile/Bundler groups for dependency activation.
2. GoShip equivalent target:
- global framework/module manifest + adapter selection;
- per-environment overrides;
- startup wiring that follows enabled modules/adapters.

### Execution Topology and File Approach

To decide what runs in parallel (web/worker/scheduler) and with which adapters, use layered config files:

1. `config/application.yaml`:
- global defaults;
- module enablement;
- adapter defaults (`db`, `cache`, `jobs`, `pubsub`, `storage`).
2. `config/environments/{local,dev,test,prod}.yaml`:
- per-environment overrides;
- profile selection (`server-db`, `single-node`, `distributed`).
3. `config/processes.yaml`:
- process topology matrix:
  - `web` (bool)
  - `worker` (bool)
  - `scheduler` (bool)
  - `co_located` (bool)
- examples:
  - local default: web=true, worker=true, scheduler=true (single process with goroutines if backend supports it)
  - prod distributed web: web=true, worker=false, scheduler=false
  - prod worker: web=false, worker=true, scheduler=true
4. `config/initializers/*.go` (generated/wired by CLI):
- runtime boot wiring based on enabled modules/adapters/processes.

CLI responsibilities:

1. `ship new` writes initial config set with sane defaults.
2. `ship profile:set <profile>` updates environment/process presets.
3. `ship module:add <name>` updates module manifest + initializer wiring.
4. `ship jobs:backend:set <backend>` updates adapter config with capability checks.
5. `ship new` should install templ tooling pinned to the project-declared version and provide an explicit update path.

### CLI Surface (Rails-Inspired)

Primary command groups to match Rails ergonomics while staying Go-native:

1. Project lifecycle:
- `ship new <app>`
- `ship upgrade`
- `ship doctor`
2. Runtime/developer workflow:
- `ship dev` (web-only default)
- `ship dev --worker`
- `ship dev --all`
- `ship test` (unit default)
- `ship test --integration`

Canonical dev command contract:

- no `shipdev` alias;
- no positional dev mode arguments (`ship dev web|worker|all`);
- only explicit flags (`--web`, `--worker`, `--all`) for mode selection.
3. Code generation:
- `ship make:resource`
- `ship make:model`
- `ship make:controller`
- `ship make:scaffold`
- `ship destroy:<artifact>`
4. Modules/adapters:
- `ship module:add <name>`
- `ship module:remove <name>`
- `ship adapter:set <db|cache|jobs|pubsub|storage|mailer> <impl>`
5. Data/schema:
- `ship db:make <name>`
- `ship db:make <name> --soft-delete --table <table>`
- `ship db:migrate`
- `ship db:rollback`
- `ship db:seed`

Rules for versioned tooling in generated apps:

1. CLI installs tool versions pinned to the project declaration.
2. CLI does not auto-upgrade tools to latest on dev/test commands.
3. `ship doctor` reports drift (e.g., templ CLI older/newer than project version) and provides fix commands.
4. `ship upgrade` is the only command that intentionally bumps pinned tool/module versions.

## Core Product Goals

1. Rails-like productivity in Go.
2. LLM-first developer experience.
3. Convention over configuration.
4. Modular infrastructure adapters.
5. Strong defaults with optional escape hatches.

## Current Repository Shape (Post-Refactor)

1. `app/` contains app-specific web handlers and templ views.
2. `framework/` is the framework/infrastructure layer.
3. `cmd/` contains process entrypoints.
4. `ship new` templates are embedded from `tools/cli/ship/internal/templates/starter` (no separate runtime `starter/` app tree in repo root).

Note: app composition/runtime wiring has moved to `app/foundation`. `framework/repos` remains intentionally centralized for now and will be split into app-specific vs reusable framework modules in a dedicated follow-up pass.

## Architecture Style (Pragmatic)

GoShip will use a pragmatic blend:

1. Rails-style developer experience at the framework surface.
2. Selective clean-architecture boundaries only where they add real value.

What this means:

1. Do not implement clean architecture verbatim across every layer.
2. Use interfaces at infrastructure seams so backend technology is swappable.
3. Keep core app/domain logic straightforward and low-ceremony.

Primary infrastructure seams to abstract:

1. `Store` (database)
2. `Cache`
3. `Jobs`
4. `PubSub`
5. `BlobStorage`
6. `Mailer`
7. `AI`
8. `DomainEvents`

Non-goal:

- Avoid abstracting everything "just in case"; abstractions must improve portability or testability.

## Confirmed Decisions

1. Use Bob + Goose as the canonical ORM/migrations path.
2. Use a monorepo with multiple Go modules plus `go.work` for maintainers.
3. Ship installable/versioned modules (auth, billing, notifications, storage, admin).
4. Keep one blessed default stack, but support both single-node and distributed runtime modes via adapters.
5. Near-term default is single-binary-first (SQLite first), with expansion to separate worker, cache, and database services through adapters.
6. Redis is optional capability, not a hard requirement.
7. GoShip must remain fully standalone; any hosted control plane consumes stable runtime hooks and managed overrides but does not become a framework dependency.
8. Feature flags baseline is framework-owned (`modules/flags` + `container.Flags`) with admin toggle access at `/auth/admin/flags`.
9. The prior `app/contracts` + `ship api:spec` approach was removed during app-minimalization cleanup; request DTO ownership is now local to controllers/modules.
10. Test-data factory baseline now lives in `framework/factory` with `ship make:factory` scaffold support.
11. Typed HTTP integration helpers now live in `framework/testutil` (`NewTestServer`, `PostForm` CSRF automation, `AsUser`, fluent response assertions).
12. I18n baseline now lives in `modules/i18n` with canonical TOML locale files under `locales/` (temporary YAML dual-read migration support), runtime toggle support (`PAGODA_I18N_ENABLED` / `PAGODA_I18N_DEFAULT_LANGUAGE`), strict-mode doctor enforcement (`PAGODA_I18N_STRICT_MODE` + `.i18n-allowlist`), profile preference persistence (`profiles.preferred_language`), and CLI support for `make:locale`, `i18n:init`, `i18n:scan`, `i18n:instrument`, `i18n:migrate`, `i18n:normalize`, `i18n:missing`, and `i18n:unused`.
13. `ship new` now asks (interactive) whether to enable i18n in the starter app, supports explicit `--i18n|--no-i18n` for non-interactive runs, and prints an explicit “enable later + migrate later” path.
14. I18n operator DX is now documented in `docs/guides/10-i18n-llm-migration-workflow.md`, including canonical migration loop, deterministic diagnostics schema, issue-ID contract, and strict rollout (`off -> warn -> error`) guidance.
15. `app/controller` was removed; app page ownership is `app/web/ui.Page` with reusable framework-owned base fields/behavior extracted into `framework/web/page`.
16. Legacy `app/subscriptions` was removed; plan catalog construction now lives in app runtime composition (`app/foundation/subscription_catalog.go`), and paidsubscriptions integration branches through catalog/service predicates instead of fixed free/pro key literals.
17. Module source isolation temporary exceptions were reduced to a short allowlist focused on remaining notifications/paidsubscriptions bridge files; new exceptions should be treated as regressions by default.
18. G1-03 hardening now promotes structural doctor checks that define canonical repo shape (`DX005` unpaired markers, `DX027` raw controller form parsing) to blocking errors and removes the temporary `DX020` module-isolation allowlist.

## Upstream/Downstream Relationship

GoShip is the framework upstream.
Cherie is a downstream product built on top of GoShip.

Framework work must include a sync path so Cherie stays current without fragile manual cherry-picking.

## Candidate Capabilities To Pull From Cherie

Based on Cherie docs and current implementation, these are strong candidates to upstream into GoShip modules:

1. Realtime baseline that is fully wired (SSE endpoint + unread counts + notification center patterns).
2. Mature notification permissions model (type + platform + grant/revoke lifecycle).
3. Background job patterns for daily/periodic workflows (retention, maintenance, notification orchestration).
4. Referral system primitives (link generation, attribution, reward application).
5. Gamification primitives (points/progression hooks) as optional module.
6. Multi-app branding strategy from one codebase (app profile/brand config).
7. Security hardening patterns:
- route-level authorization checks for resource interaction;
- friend/relationship ownership checks where relevant;
- explicit forbidden/not-found behavior for unauthorized access.
8. Production operations runbooks:
- deploy profile separation;
- cache invalidation/ops guidance;
- migration caveats and guardrails.

## Pagoda Upstream Intake Plan

Long-term policy:

1. Treat Pagoda as an upstream source of framework/runtime improvements.
2. Regularly evaluate and selectively port changes into GoShip.
3. Do not adopt Pagoda UI/component layer choices that conflict with GoShip direction.

Current known upstream shifts in Pagoda (to evaluate and/or port):

1. Default move from Postgres+Redis to SQLite-centric operation (now aligned with GoShip's single-binary-first direction, subject to GoShip's own adapter boundaries).
2. Migration from Asynq to Backlite for DB-backed task queues.
3. In-process task runner startup in web process, with graceful task shutdown in container.
4. Use of in-memory cache as default for simpler local development.
5. Admin/task runtime integration improvements.

Non-goals for direct adoption:

1. Go-based HTML component stack from Pagoda (`gomponents`) as a hard dependency.
2. Any upstream UI architecture changes that reduce GoShip's Templ+HTMX ergonomics.
3. Forcing hosted-control-plane assumptions into GoShip's runtime architecture.

## Managed and Self-Managed Operation

GoShip should support both:

- self-managed operation: the app owns its own settings, backup controls, and deployment choices.
- externally managed operation: the app keeps the same runtime capability, but settings authority may be delegated to an external control plane.

Boundary rules:

1. framework capability belongs in GoShip;
2. fleet authority belongs outside GoShip;
3. provider deployment logic must not become a required part of app runtime;
4. managed overrides must be explicit, allowlisted, and inspectable.

Current managed settings UX contract:

1. managed-capable settings expose explicit local access states (`editable`, `read-only`, `externally-managed`);
2. settings surface (`/welcome/preferences`) shows effective value, source layer, and authority metadata;
3. admin surface (`/auth/admin/managed-settings`) shows the same access-state model for operators.

### SQLite-To-Postgres Promotion Contract (v1)

Promotion target:

- Source: SQLite (`embedded` mode)
- Target: Postgres (`standalone` mode)
- Workflow identifier: `sqlite-to-postgres-manual-v1`

Required runtime metadata contract:

- DB mode and normalized DB driver
- migration tracking table
- migration dialect
- migration portability profile (`sql-core-v1`)
- compatible target drivers
- active promotion path (when source is SQLite)

First supported promotion workflow:

1. Freeze writes in the source runtime.
2. Capture runtime metadata + migration baseline.
3. Export SQLite data with framework-owned export hooks.
4. Provision Postgres and apply canonical migrations.
5. Import exported data with framework-owned import hooks.
6. Run verification hooks (row counts, migration baseline, integrity checks).
7. Switch runtime config to Postgres and resume writes.

Portability constraints for framework/module authors:

1. Default to SQL that is portable across SQLite and Postgres.
2. Use explicit dialect branches for engine-specific SQL.
3. Keep migration files deterministic and idempotent for replay/import workflows.
4. Avoid SQLite-specific assumptions in reusable module contracts.

Minimum framework tooling/hooks to expose:

1. Runtime metadata report contract (read-only).
2. Data export hook with typed manifest (version + dialect + checksums).
3. Data import hook with manifest validation.
4. Post-import verification hook callable from CLI/control-plane adapters.
5. Dedicated CI suites for module isolation and `sql-core-v1` portability so boundary regressions fail in named lanes instead of broad aggregate jobs.
6. Shared/distributed replay storage contract for managed hook nonce tracking so multi-replica managed mode rejects replays consistently.

## Docket Tracking

Framework follow-up for this boundary is tracked in:

- `TKT-110` managed-mode config authority (done)
- `TKT-111` backup capability contract and S3-compatible providers (done)
- `TKT-112` managed-mode settings lock/read-only admin surfaces (done)
- `TKT-113` signed managed hooks for backup, restore, and runtime status (done)
- `TKT-114` SQLite-first promotion path to Postgres (done)

## Documentation Quality Initiative

Primary goal:

1. Deliver documentation quality at least equal to Pagoda's onboarding clarity, and better on practical implementation guides.

Near-term deliverable:

1. Build out `docs/guides/03-how-to-playbook.md` into concrete how-to guides for common engineering tasks.

Initial guide set:

1. Add endpoint
2. Add page/view
3. Add model + migration
4. Add service/repo
5. Add background job
6. Add adapter
7. Add tests

## Pagoda Intake TODOs

- [x] Create a recurring upstream review cadence (weekly or per-tag) for Pagoda.
- [x] Add a "Pagoda intake log" mapping upstream commit/tag -> GoShip decision (`adopt`, `adapt`, `skip`).
- [ ] Evaluate Backlite-style DB-backed jobs as a GoShip jobs adapter candidate.
- [ ] Port container lifecycle hardening patterns where applicable (startup/shutdown ordering and timeouts).
- [ ] Port testability improvements that reduce Docker dependence.
- [ ] Keep UI/component layer decisions independent from runtime/service layer intake.
- [ ] Prefer LLM-assisted feature re-implementation over direct commit cherry-picks due codebase divergence.

Governance cadence:

- Review Pagoda upstream changes weekly or per-tag, whichever happens first.
- Record each reviewed item in `docs/roadmap/09-pagoda-intake-log.md` with an explicit `adopt`, `adapt`, or `skip` decision and any follow-up ticket IDs.

## Cherie Sync Policy

1. Every GoShip framework milestone must include a Cherie compatibility check.
2. Breaking changes require:
- migration notes;
- codemods or scripted upgrade steps where possible;
- direct hard-cut wording in canonical docs and CLI surfaces.
3. Maintain a living "GoShip -> Cherie adoption board" with statuses:
- `not started`
- `in progress`
- `adopted`
- `blocked`
4. Do not merge major framework refactors without validating Cherie boot, auth flow, and realtime flow.

## Runtime Modes

### 1) Single-Binary Mode (Primary Near-Term Default)

- DB: SQLite
- Cache: in-memory
- PubSub: in-process
- Jobs: in-process scheduler/worker or DB-backed queue

### 2) Server-DB Mode (Expansion Profile)

- DB: external DB server (Postgres first; MySQL later through adapter boundary)
- Cache: in-memory by default
- Jobs: pluggable (`inproc` for simplicity, durable backend for reliability)
- Redis: optional, not required

### 3) Distributed Mode

- DB: Postgres
- Cache: adapter-driven (Redis optional)
- PubSub: adapter-driven
- Jobs: adapter-driven (DB-backed queue or external queue service)

## Worker Queue Abstraction Strategy

Goal:

- one stable app-facing jobs API with multiple backend implementations.

Design principles:

1. Define a minimal stable core contract in `goship/jobs`:
- `Register(name, handler)`
- `Enqueue(name, payload, opts...)`
- `StartWorker(ctx)`
- `StopWorker(ctx)`
- `StartScheduler(ctx)` (if supported)
2. Use capability declarations per backend (delayed jobs, retries, cron, priority, dead-letter, UI).
3. Validate feature usage against backend capabilities at startup.
4. Keep backend-specific settings in adapter config, not spread in app code.
5. Keep handlers/payload contracts backend-agnostic.
6. Introduce capability contracts so unsupported backend features fail fast at startup.

Planned adapters:

1. `jobs/inproc` (best local DX)
2. `jobs/dbqueue` (DB-backed durable queue, no Redis required)
3. `jobs/asynq` (optional Redis-backed adapter)
4. future cloud adapters (Pub/Sub / Cloud Tasks bridges)

## Backend-Agnostic Framework Rule

Like Rails/Django, GoShip should not force one database/cache choice.

Policy:

1. Framework APIs remain backend-agnostic.
2. Backend selection happens in config + runtime wiring.
3. Application/business code must not depend directly on concrete infra clients.

## Controller and Domain Layer Rules

Rails/Laravel-inspired ergonomics in GoShip should use clear layer responsibilities:

1. Controllers (route handlers) are HTTP adapters, not business logic containers.
2. Controllers should only orchestrate request/response concerns and call service/use-case code.
3. Business logic should live in app domain/service packages.
4. Domain/service packages should depend on small interfaces (store ports) so behavior is testable via mocks/fakes.
5. Do not force repository pattern everywhere; use explicit interfaces where they improve clarity/testability.

## Installable Module Extraction Rule

A package is installable as an official `ship` module only if:

1. It has no hard dependency on app route/view packages (`app/web/controllers`, templ views).
2. It has a stable, documented config surface.
3. It is wired through stable interfaces/adapters.
4. It has dedicated tests and module-level docs.
5. It can be enabled/installed through `ship` with deterministic wiring.

Initial candidate official modules:

- storage
- mailer
- notifications
- jobs adapters
- subscriptions/billing

## Routing Organization (Rails-Inspired, Pragmatic)

Target:

1. Keep one orchestration router entrypoint.
2. Register routes by domain slices (auth, public, docs, billing, notifications, etc.).
3. Keep domain registration functions small and convention-driven.
4. Avoid over-engineered plugin systems in early stages.

Current direction:

1. `BuildRouter` handles shared middleware and runtime feature gating.
2. Domain registration is split into focused files:
- public
- docs
- auth
- external
- realtime
3. App composition happens via a single route composition function, preparing for multi-app mounting later.

## Required Core Interfaces

Create stable contracts in `core` so app code is backend-agnostic:

1. `Store` (database/repository boundary)
2. `Cache`
3. `PubSub`
4. `Jobs`
5. `SessionStore`
6. `BlobStore`
7. `Auth`
8. `Billing`
9. `Notifications`

## Rails-Like Capabilities to Implement First

1. `ship new <app>` CLI with a 2-minute happy path.
2. Generators:
- `ship make:model`
- `ship make:scaffold`
- `ship db:make`
- `ship make:job`
- `ship make:mailer`
- `ship module:add <name>`

Job-generator contract tracked under `TKT-250` / `TKT-313` / `TKT-314`:
- `ship make:job <Name>` is now a first-class generator in help, docs, and CLI dispatch;
- the scaffold now emits `app/jobs/<name>.go` and `app/jobs/<name>_test.go` around `core.Jobs` / `core.JobHandler` registration helpers instead of introducing new backend-specific processor patterns.

Mailer-generator contract tracked under `TKT-251` / `TKT-315` / `TKT-316`:
- `ship make:mailer <Name>` is now a first-class generator in help, docs, and CLI dispatch;
- the scaffold now targets templ email views and wires the existing `/dev/mail/*` preview surface instead of introducing a parallel preview path.

OSS workflow-docs contract tracked under `TKT-252` / `TKT-317` / `TKT-318`:
- canonical task-oriented guides now exist for add endpoint, add module, and add background job flows;
- the docs index and how-to playbook now surface those guides as the canonical contributor workflow set.

Shared-infra adoption reporting tracked under `TKT-238` / `TKT-355` / `TKT-356`:
- `ship describe` now exposes a non-blocking shared-infra adoption summary;
- the summary reports shared module adoption alongside app-owned controller/job/command counts to guide upstreaming decisions.

Pagoda-intake governance tracked under `TKT-253` / `TKT-357` / `TKT-358`:
- the executable spec now pins a recurring Pagoda intake cadence plus an adopt/adapt/skip decision log;
- the docs set is expected to expose the cadence and a canonical intake table for follow-up actions.
3. ActiveStorage-like file attachments:
- attach files to entities;
- support local + S3 backends;
- simple URL + variant APIs.
4. Admin scaffolding from Bob query/model metadata.
5. Background jobs with retries/scheduling.

## Frontend Strategy (HTMX-First, Svelte as Islands)

Svelte should remain optional and isolated. Current pattern in Cherie/GoShip bundles all Svelte entrypoints into one global `svelte_bundle.js`, which increases payload and causes coupling.

Target approach:

1. Keep HTMX + Templ as default.
2. Load Svelte only for pages/components that need it.
3. Replace single global Svelte bundle with per-island chunks (no single global Svelte build artifact loaded site-wide).
4. Use auto-discovery to find islands and generate a manifest + runtime registry automatically.
5. Use dynamic imports so `renderSvelteComponent(name, ...)` loads code on demand.
6. Avoid globally injecting `svelte_bundle.js` for all pages.

Important DX constraint:

- No manual island wiring by default.
- Island wiring should be generated by CLI/scaffolding.

### Auto-Discovery + Auto-Wire Model

Desired developer flow:

1. Create a Svelte component either:
- next to its usage (colocated), or
- in a central reusable island library.
2. Reference it in Templ with a helper/component tag and props.
3. Build step auto-discovers island files and generates:
- island manifest (`component -> js/css asset`);
- runtime registry for lazy mounting;
- optional typed helper stubs.
4. Templ helper injects only needed island script(s) for that page/partial.

No manual edits should be required for:

- central JS registry files;
- script tag wiring in templates;
- per-component import maps.

## Svelte/Islands TODOs

- [ ] Replace single `svelte_bundle.js` build with multiple outputs (code-split entries).
- [ ] Add an island manifest mapping `componentName -> asset path`.
- [ ] Add template helper to include only required island scripts per page.
- [ ] Support HTMX swap lifecycle: mount/unmount Svelte instances safely after partial swaps.
- [ ] Benchmark before/after page payload and interaction latency.
- [ ] Document "when to use Svelte vs Alpine vs vanilla vs pure HTMX" as framework guidance.
- [ ] Add `ship make:island <Name>` to generate component + entrypoint + registration with zero manual edits.
- [ ] Support colocated islands and central-library islands with the same discovery pipeline.
- [ ] Add a watch mode that re-generates island manifest/registry automatically during development.

## Frontend Alternatives (No-Compile Paths)

Question: can we load raw `.svelte` components directly in browser without build?

Answer:

- Not as a production default. Svelte is compile-based.

Viable alternatives for no-build interactivity:

1. HTMX + Alpine (preferred default for small/medium interactions).
2. Vanilla Web Components for reusable widgets.
3. Lightweight runtime libraries (e.g. Petite-Vue) where appropriate.

Decision:

- Keep Svelte optional for advanced islands.
- Keep HTMX-first and no-build-friendly by default.
- Ensure CLI removes manual compile-pipeline pain where Svelte is used.

## Unified Styling Policy

Goal:

- one unified styling system across GoShip modules and generated apps.

Preferred direction:

1. Use a single design system source of truth (tokens, component variants, spacing, colors, typography).
2. If Svelte is used, keep style parity with the same design tokens/components (e.g. shadcn + shadcn-svelte style equivalence).
3. Avoid fragmented ad-hoc component styling across stacks.

LLM/agent styling guardrails:

1. LLMs may change HTML/Templ/Svelte structure and behavior logic.
2. LLMs should not freely rewrite Tailwind styling.
3. Styling changes must be centralized in designated style system files with explicit documentation comments.
4. Generated/component docs should clearly mark style contract boundaries as "do not mutate directly" unless requested.
5. Framework prompts/checklists should enforce: "change behavior first, preserve style tokens/classes unless style task is explicitly requested."

## Testing Strategy (Developer Ergonomics First)

Test workflow should be fast and local-first without requiring Docker for most feedback loops.

Principles:

1. Maximize unit tests and table-driven tests for business logic.
2. Push side effects behind interfaces to allow in-memory fakes.
3. Keep integration tests focused and limited (happy-path + critical failure cases).
4. Keep end-to-end tests minimal and scenario-driven.
5. Avoid making Docker a prerequisite for routine test runs.

Target pyramid:

1. Unit tests (majority): pure Go, table tests, no network, no containers.
2. Integration tests (few): DB/repo boundaries and adapter contracts.
3. E2E tests (very few): key user journeys only.

### Pre-Commit Test Policy

1. Every commit must pass `lefthook` pre-commit tests.
2. Pre-commit runs a fast, stateless Go unit suite only.
3. Docker/integration suites run separately (manual or CI), not as a local pre-commit default.
4. As packages are refactored, they should be moved into the pre-commit suite.

### Agent-Driven Documentation and Downstream Sync Policy

For every implementation change and commit:

1. Update developer + LLM-oriented docs in the same change stream.
2. Keep docs split into focused markdown files by area/topic rather than one giant file.
3. Reflect behavior changes in framework docs and LLM reference docs.
4. Add/update downstream impact notes for Cherie when GoShip changes affect integration.
5. Treat documentation sync and Cherie sync as required agent checks.

## Commit Standard

Use Conventional Commits for all framework work:

`type(scope): imperative summary`

Allowed types:

- `feat`
- `fix`
- `refactor`
- `test`
- `docs`
- `chore`
- `ci`

Examples:

- `fix(services): make container shutdown nil-safe`
- `test(services): add nil-safe shutdown coverage`
- `docs(plan): define first rework execution workflow`

## Module Plan

Proposed module boundaries:

1. `packages/core`
2. `packages/auth`
3. `packages/billing`
4. `packages/notifications`
5. `packages/storage`
6. `packages/admin`
7. `tools/cli/ship`

## Priority Roadmap

### Phase 0: Stabilize Current Base

1. Fix container initialization mismatch and shutdown safety.
2. Resolve realtime/notification route wiring drift.
3. Clean and align current docs with actual runtime behavior.
4. Refresh stale e2e coverage for critical flows.

## First Rework Execution Plan

This section is the active implementation tracker for the first rework pass.
Rule: execute exactly one task at a time, validate with tests, then move to the next task.

### Quality Gates (for every task)

1. Add or update tests with the change (prefer table-driven tests).
2. Run targeted tests for touched package(s).
3. Keep tests mostly stateless (no Docker for default test path).
4. No task is marked complete without test evidence.

### Coverage Targets

1. Global target: 90%+ over time (not required in one PR).
2. Reworked packages should trend toward 90%+ before moving on.
3. Complex pure-logic packages should aim for near-100% branch coverage with table tests.

### Active Task Queue (First Rework)

1. `R0.1` Container shutdown safety + reliable container unit test baseline.
Status: `completed`
Done when:
- `Container.Shutdown()` is nil-safe for optional services.
- container unit tests compile and pass without external services.
Test evidence:
- `go test ./app/foundation -run 'Test(NewContainer|ContainerShutdownNilSafe)$'`

2. `R0.2` Container initialization policy by runtime mode (`single-node` vs `distributed`), with explicit config contract.
Status: `completed`
Done when:
- runtime/process/adapters config scaffold exists;
- runtime plan resolver exists with table tests;
- no startup behavior change yet (scaffold only).
Test evidence:
- `go test ./config ./framework/runtimeplan`

3. `R0.3` Router consistency pass (realtime + notifications wired consistently with initialized dependencies).
Status: `completed`
Done when:
- router enables cache middleware only when cache dependency is available;
- realtime routes are wired only when notifier+pubsub dependencies are available;
- runtime plan is resolved at router build with safe fallback on invalid plan configuration.
Test evidence:
- `go test ./framework/runtimeplan`

Follow-up hardening tracked under `TKT-195` / `TKT-257` / `TKT-258`:
- invalid runtime-plan resolution now fails startup instead of warning + fallback;
- resolved realtime capability mismatches (for example notifier/pubsub startup gaps) now fail startup instead of silently disabling route surface;
- adapter-backed dependency mismatches (for example `pubsub=redis` without its required runtime dependency) now fail at startup.
- notification-center route inventory and canonical route-name cleanup are tracked under `TKT-197` / `TKT-261` / `TKT-262`.

Default-topology hardening tracked under `TKT-198` / `TKT-263` / `TKT-264`:
- local and dev defaults now normalize to `single-node`;
- the canonical default local topology is colocated web/worker/scheduler with SQLite + Otter + Backlite + inproc pubsub semantics.

Dev-loop convergence tracked under `TKT-199` / `TKT-265` / `TKT-266`:
- `ship dev` still resolves default mode from jobs-adapter heuristics instead of the explicit runtime profile;
- executable red specs now pin the intended contract: `single-node` should stay on the web/app-on loop by default, while `distributed` should opt into the full multiprocess loop.

Cache-contract parity tracked under `TKT-200` / `TKT-267` / `TKT-268`:
- in-memory page caching is now wired through the same cache seam as Redis;
- one shared contract suite now covers grouped payloads, tag invalidation, TTL expiry, and raw byte/prefix operations across `memory` and `redis`;
- the shared cache seam normalizes positive TTLs to second precision so adapter choice does not change expiry semantics.

Route-composition hardening tracked under `TKT-232` / `TKT-293` / `TKT-294`:
- `app/router.go` remains the canonical route composition root;
- `BuildRouter` now rejects a nil container explicitly before route/module wiring;
- static route registration now flows through one canonical `appweb.RegisterStaticRoutes(c)` path, including PWA assets.

Docs hard-reset tracking under `TKT-233` / `TKT-295` / `TKT-296`:
- doctor now flags selected migration-bridge wording in the canonical architecture, CLI, and roadmap docs set;
- canonical docs have been rewritten to describe the current hard-cut model directly instead of relying on migration-era framing.

Local-runtime alignment tracked under `TKT-234` / `TKT-297` / `TKT-298`:
- config defaults now point to `single-node`, while compose-backed accessories remain optional Redis + Mailpit only;
- Makefile help and contributor docs now describe the same local runtime story as config and compose;
- the default local contract is app-on single-node first, with compose reserved for optional accessory services.

Runtime-report contract tracked under `TKT-235` / `TKT-299` / `TKT-300`:
- the runtime metadata/config primitives now feed `ship runtime:report --json`;
- `ship runtime:report --json` is now the canonical machine-readable runtime capability surface for operators and agents.

CLI-path reset tracked under `TKT-249` / `TKT-311` / `TKT-312`:
- the executable spec now pins one canonical top-level quality path per concern;
- `ship check` has been removed from help, docs, dispatch, and agent-policy surfaces;
- `ship test` now owns the fast package-list quality loop, while `ship verify --profile fast|standard|strict` covers tiered repository verification.
- `verify_strict` plus `cherie_compatibility_smoke` now define the required Cherie sync baseline in CI.

4. `R0.4` Testing harness improvements so default `make test` is Docker-free and fast.
Status: `completed`
Done when:
- default `make test` executes a Docker-free unit package set;
- integration/infra-heavy tests run via separate command;
- cache-dependent unit tests do not fail when cache service is disabled in runtime profile.
Test evidence:
- `bash tools/scripts/test-unit.sh`

5. `R0.5` Establish package-level coverage baselines and close highest-value test gaps.
Status: `in_progress`

6. `R1.1` Resource route registration refactor (prep for generator-driven resources).
Status: `completed`
Done when:
- canonical router entrypoint remains `app/router.go`;
- route declarations are centralized in `app/router.go`;
- handler implementations remain in `app/web/controllers/*.go`;
- realtime registration is feature-gated directly in the canonical router.
Test evidence:
- `go test ./cmd/web ./app/web/ui ./framework/runtimeplan`
- `go test -c ./app/web/controllers` (compile check in restricted env)

7. `R1.2` Minimal resource generator foundation.
Status: `completed`
Done when:
- `ship make:resource <name>` scaffolds route handler (+ optional templ page);
- generator prints router insertion snippet instead of auto-editing `app/router.go`;
- generator logic is table-driven tested in `tools/cli/ship`.
Test evidence:
- `go test ./tools/cli/ship`

8. `R1.3` Optional safe router wiring mode for generator.
Status: `completed`
Done when:
- `ship make:resource ... --wire` inserts generated snippet into `app/router.go`;
- insertion only occurs behind explicit marker pairs (`public` or `auth`);
- operation is idempotent and tested (no duplicate insertion).
Test evidence:
- `go test ./tools/cli/ship`

9. `R1.4` Route name automation and dry-run previews for resource generation.
Status: `completed`
Done when:
- generator ensures `RouteName<Resource>` constant is present in `app/web/routenames/routenames.go`;
- `ship make:resource ... --dry-run` previews file/router/routename changes without writing;
- wiring/import/constant insertion paths are idempotent and tested.
Test evidence:
- `go test ./tools/cli/ship`

10. `R1.5` Canonical generator output and idempotency matrix.
Status: `completed`
Done when:
- `ship make:*` commands share one output contract built around `Created:`, `Updated:`, `Preview:`, and `Next:` sections;
- dry-run generator output stays in the same report shape with explicit dry-run labeling;
- generators package includes a consolidated idempotency matrix test for duplicate scaffold attempts.
Test evidence:
- `go test ./tools/cli/ship/internal/generators`

### Phase 1: Core Abstractions

1. Define `core` interfaces for DB/cache/pubsub/jobs/storage.
2. Implement adapters:
- SQLite + Postgres
- memory-cache + Redis-cache
- inproc-pubsub + Redis-pubsub
- inproc-jobs + Asynq
3. Add runtime mode config (`single-node`, `distributed`).

Current progress:

1. Adapter registry + capability validation are active at container startup.
2. Container exposes `CoreCache`, `CoreJobs`, and `CorePubSub`.
3. First application call sites migrated to interface seams:
- notifications fan-out enqueue in `app/jobs/notifications.go` now uses `core.Jobs`.
- notifier publish/subscribe path in `modules/notifications/notifier.go` now uses `core.PubSub`.
4. Domain testability improved:
- notifications task processor now depends on a small planned-notification interface with table-driven unit tests.

### Phase 2: Monorepo and Module Packaging

1. Restructure into multi-module layout.
2. Add `go.work` for local development across modules.
3. Establish semver tagging and module release process.
4. Define how Cherie consumes modules during local dev (`go.work`) vs released versions (tags).

### Phase 3: CLI and Generators

1. Build `goship` CLI.
2. Implement app bootstrap and generator commands.
3. Add idempotent install/wire commands for optional modules.

### Phase 4: Batteries and DX

1. Deliver auth, storage, notifications, billing, admin modules.
2. Implement ActiveStorage-like attachment primitives.
3. Improve diagnostics, error pages, and test templates.

### Phase 5: LLM-First Tooling

1. Add `llm.txt` as machine-readable framework reference.
2. Add an MCP server exposing commands, module contracts, and examples.
3. Generate concise human docs from the same source of truth.

## TODO Checklist

## Immediate

- [x] Decide and document exact package naming convention (`github.com/leomorpho/goship/*`).
- [x] Choose CLI implementation approach (stdlib `flag` + explicit command dispatch in `tools/cli/ship`).
- [x] Draft `core` interface contracts in a design doc and first package (`framework/core`).
- [x] Define runtime config schema for adapter selection and add startup adapter validation registry.
- [ ] Specify module compatibility/version policy.
- [ ] Create a developer-facing README + LLM-facing README/`llm.txt` split with one source of truth.
- [ ] Define MCP server scope for GoShip (commands, module APIs, recipes, migration help).
- [ ] Create a `CHERIE_SYNC.md` runbook (upgrade process + rollback + validation checklist).
- [ ] Create a baseline compatibility test suite for Cherie critical paths.
- [ ] Add a dedicated CI smoke lane that names and enforces the Cherie boot/auth/realtime baseline separately from the generic GoShip smoke test.
- [ ] Publish one extension-zone manifest that distinguishes safe customization areas from protected framework contract seams and keep `ship doctor` aligned to it.
- [ ] Add a run-anywhere verification gate that proves generated apps remain standalone-exportable with no control-plane dependency assumptions.
- [ ] Define testing standards doc: what must be unit-testable and where table tests are required.
- [ ] Add doc-sync guardrails in pre-commit/CI for framework-impacting changes.
- [ ] Add Cherie-sync guardrails in pre-commit/CI (or mandatory checklist gate).

## Near-Term

- [ ] Build first two DB adapters.
- [ ] Build first cache pair: `memory` + `redis`.
- [ ] Build first pubsub pair: `inproc` + `redis`.
- [ ] Build first jobs pair: `inproc` + `asynq`.
- [ ] Prototype attachment API with local and S3 storage.
- [ ] Refactor high-logic route/service code into testable units with interface boundaries.
- [ ] Add in-memory test doubles for cache/pubsub/jobs/storage adapters.
- [ ] Ensure default `make test` runs without Docker.

## Mid-Term

- [x] Release minimal `ship new` CLI command (v1 local scaffold, no network bootstrap).
- [x] Release `ship make:model` and `ship db:make`.
- [ ] Release `auth` and `storage` modules.
- [ ] Release `admin` scaffolding MVP.
- [ ] Add golden-path example apps for both runtime modes.
- [ ] Move Cherie onto released GoShip modules incrementally (module by module).
- [ ] Upstream selected Cherie capabilities into optional GoShip modules (notifications/referrals/gamification/security helpers).
- [ ] Keep Docker-based integration suite as optional/CI-focused (`make test-integration`), not default local path.
- [x] Keep starter i18n scaffolding focused on `en`/`fr`; treat broader locale sets as follow-up work outside `ship new`.

## Open Questions

1. How strict should conventions be before allowing customization hooks?
2. Which features are mandatory in v1 versus module-only?
3. What is the minimum stable API surface for `core` v1.0.0?
4. How should we guarantee cross-module compatibility at release time?

## Definition of Success (v1)

1. A developer can run `ship new myapp` and ship a working app quickly.
2. The same app code can run in single-node or distributed mode by config.
3. Optional batteries are added via CLI without copy-paste.
4. LLMs can reliably reason over framework structure using `llm.txt` + MCP.
5. Cherie can upgrade to current GoShip with a documented, repeatable process.
