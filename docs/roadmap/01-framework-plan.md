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
- The roadmap also needs one explicit cross-lane dependency matrix plus a must-finish-before contract map so ticket sequencing stays deterministic across runtime, docs, and control-plane work.

## Cross-Lane Dependency Matrix

This cross-lane dependency matrix is the canonical coordination contract for sequencing GoShip,
Interaction, and Control-plane work.

| Lane | Requires | Must finish before | Why |
| --- | --- | --- | --- |
| GoShip runtime contracts | `runtime-contract-v1`, `runtime-handshake-v1`, `managed-key-schema-v1` | Interaction or control-plane deploy/canary automation | Downstream orchestration must consume one stable runtime report contract instead of reverse-engineering repo state. |
| Interaction upgrade/promotion contracts | `upgrade-readiness-v1`, `promotion-state-machine-v1`, `sqlite-to-postgres-manual-v1` | Any upgrade or promotion automation beyond local CLI preflight | Upgrade/promotion policy cannot safely automate around unstable or partial runtime states. |
| Interaction recovery contracts | `backup-manifest-v1`, `restore_evidence.record_links` | Incident, rollback, and recovery audit consumers | Recovery evidence must stay linkable to external incident and deploy records without runtime storage coupling. |
| Control-plane policy and rollout tooling | `staged-rollout-decision-v1`, `policy_input_version` | Vendor-specific rollout execution or customer-facing recovery workflows | External policy may compose runtime facts, but it must not redefine the runtime-owned contract payloads. |
| Docs and verification lane | `docs/reference/01-cli.md`, `docs/architecture/09-standalone-and-managed-mode.md`, contract tests | Any new coordination ticket that claims a lane dependency | Ticket ordering stays deterministic only when the docs and tests encode the same contract map. |

## Must-Finish-Before Contract Map

- `runtime-contract-v1` and `runtime-handshake-v1` must finish before any deploy or canary policy consumes `ship runtime:report --json`.
- `upgrade-readiness-v1` must finish before upgrade automation or upgrade policy approvals rely on `ship upgrade --json`.
- `promotion-state-machine-v1` must finish before promotion orchestration attempts to continue after the config flip.
- `backup-manifest-v1` and `restore_evidence.record_links` must finish before incident/recovery tooling claims end-to-end audit linkage.
- `staged-rollout-decision-v1` must finish before rollout tooling introduces a second decision payload format.

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
2. `ship profile:set <single-binary|standard|distributed>` rewrites the `.env` runtime profile and process presets deterministically.
3. `ship module:add <name>` updates module manifest + initializer wiring.
4. `ship jobs:backend:set <backend>` updates adapter config with capability checks.
5. `ship adapter:set <db|cache|jobs|pubsub|storage|mailer> <impl>` rewrites adapter env vars deterministically and rejects invalid runtime combinations.
6. `ship db:promote` now applies the canonical SQLite-to-Postgres config mutation step from runtime metadata (with `--dry-run` / `--json` preview support) and keeps the remaining export/import hooks as explicit manual follow-up steps.
7. `ship new` should install templ tooling pinned to the project-declared version and provide an explicit update path.

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
- `ship db:promote`
- `ship db:migrate`
- `ship db:rollback`
- `ship db:seed`

Rules for versioned tooling in generated apps:

1. CLI installs tool versions pinned to the project declaration.
2. CLI does not auto-upgrade tools to latest on dev/test commands.
3. `ship doctor` reports drift (e.g., templ CLI older/newer than project version) and provides fix commands.
4. `ship upgrade` is the only command that intentionally bumps pinned tool/module versions.
5. `ship upgrade` will also surface a machine-readable upgrade readiness report and blocker schema before it mutates pinned versions.

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
   - The CLI-facing contract now exists as `ship db:export --json`, `ship db:import --json`, and `ship db:verify-import --json`; the remaining work is wiring the actual framework path behind those hooks.
5. Dedicated CI suites for module isolation and `sql-core-v1` portability so boundary regressions fail in named lanes instead of broad aggregate jobs; the module-isolation lane reports module/file context and rejects stale allowlist entries, and the SQL portability lane checks runtime metadata plus branch annotations and placeholder conventions in the canonical migration/query SQL sources.
6. Shared/distributed replay storage contract for managed hook nonce tracking so multi-replica managed mode rejects replays consistently.
7. `backup-manifest-v1` is now locked to SQLite-first metadata plus SHA-256 checksum invariants, and managed restore responses return typed restore evidence with an explicit accepted-manifest field.
8. shared signature vectors and a canonical payload library will be introduced for the INT2 bridge so runtime and control-plane signing fixtures stay aligned.
9. Managed settings will need explicit drift detection and rollback semantics so the runtime can show when intended overrides and effective state have diverged.
10. The managed-key registry will be versioned as a shared runtime/control-plane artifact so schema mapping stays authoritative instead of inferred from ad hoc key lists.
11. Signed cron entrypoint verification should reuse the same replay/timestamp contract shape so control-plane schedulers can target runtime hooks deterministically.

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

Runtime module-adoption reporting tracked under `TKT-230` / `TKT-339` / `TKT-340`:
- `ship runtime:report --json` now carries per-module adoption metadata for installed modules;
- the payload reports module identity, source, and version so orchestration tooling can consume runtime metadata without repo parsing.

Staged rollout/canary decision contract tracked under `TKT-247`:
- `staged-rollout-decision-v1` is the canonical external decision artifact for hold/canary/promote/rollback outcomes;
- the contract composes `ship runtime:report --json` readiness metadata with control-plane policy inputs instead of duplicating runtime meaning in a second payload.

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
- [x] Add `ship make:island <Name>` to generate the first canonical island module + templ mount seam; future work can reduce the remaining explicit `templ generate` / `build-js` follow-up steps.
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

## Roadmap Reset (2026-03-19)

The roadmap is no longer a second backlog.

From this point forward:

1. This document records stable direction, execution tracks, and ticketing rules.
2. Docket carries the actionable backlog.
3. Open tickets should be implementation-ready leaves, not coordination placeholders.

Work that has already landed should be reflected here as current capability or moved to the known-gaps docs, not left behind as aspirational roadmap prose.

## Active Execution Tracks

### Track 1: Framework DX

Goal:

- keep the default developer loop fast, deterministic, and easy for AI agents to modify safely.

Focus:

- one canonical command path per concern;
- stable route/container seams and repo-shape enforcement;
- generator UX, reverse scaffolding, workflow docs, and the intentionally narrow golden Playwright contract;
- CI/doctor/verify guardrails that catch drift early.

Examples of active work:

- `ship destroy` MVP;
- `ship make:island`;
- startup/HTTP/CLI contract suites;
- keep the GoShip golden Playwright suite aligned with the current scaffold contract as that contract evolves;
- standalone exportability verification.

### Track 2: Runtime Capabilities

Goal:

- make GoShip a credible standalone Rails/Laravel-style runtime before adding more orchestration complexity.

Focus:

- single-node as the canonical local truth;
- deterministic profile/adapter mutation commands;
- promotion/export/import surfaces that move from planning-only to executable;
- portable adapter contracts across cache, jobs, pubsub, and storage.

Examples of active work:

- `ship db:promote` apply-mode flow;
- `ship profile:set`;
- `ship adapter:set`;
- cache parity across memory and redis;
- shared/distributed replay store for managed hooks.

### Track 3: Installable Batteries

Goal:

- make GoShip batteries feel like first-class framework capabilities instead of partial module experiments.

Focus:

- real `ship module:add`/`module:remove` wiring for installable modules;
- battery-specific ergonomics for `storage`, `auth`, `admin`, `jobs`, and related modules;
- CLI generators that create code matching the current runtime conventions;
- module compatibility and release policy that downstream apps can trust.

Examples of active work:

- real module wiring snippets instead of TODO placeholders;
- jobs inspector surface through `core.JobsInspector`;
- admin scaffolding MVP;
- mailer and job generators;
- Cherie-oriented battery compatibility.

### Track 4: Control-Plane Interoperability

Goal:

- define the minimum runtime contracts a future external authority can rely on without turning the runtime into a control-plane client.

Focus:

- signed managed-hook contract hardening;
- upgrade/deploy/promotion preflight contracts;
- managed override registry/versioning/drift semantics;
- runtime metadata and module-adoption surfaces for orchestration tooling.

Examples of active work:

- orchestration preflight via `ship verify`;
- upgrade readiness report and blocker schema;
- distributed replay defense and key rotation;
- managed override registry versioning;
- promotion lifecycle and incident/recovery contracts.

## Ticketing Rules

Roadmap execution is biased toward AI pickup, not serialized phase completion.

Rules:

1. Epics may aggregate a theme, but their children must be leaf tickets.
2. A leaf ticket must fit in one focused session and own one deliverable.
3. Leaf tickets must include specific, observable acceptance criteria.
4. Leaf tickets must include exact likely paths and concrete verification commands.
5. Code, tests, and docs belong in the same ticket unless a true shared artifact forces a split.
6. Red-spec and implementation tickets are optional decomposition, not the default pattern.
7. Hard blockers are allowed only for real prerequisites:
   - a shared contract artifact must exist first;
   - two tickets would otherwise collide on the same narrow write scope;
   - shipping in parallel would create unsafe runtime behavior.
8. `docket start` should surface leaf implementation work before any epic or coordination ticket.

A ticket is not ready for AI execution if it only says:

- “coordinate”
- “define child topology”
- “keep evidence current”
- “track scope”

Those are epic responsibilities, not implementation tickets.

## Current Backlog Policy

Use these rules when revisiting Docket:

1. Close stale tickets whose capabilities have already landed in code, docs, and CI.
2. Repurpose broad open tickets into atomic leaves when their IDs are still useful.
3. Prefer reparenting existing granular tickets under the current epics over creating new umbrella phases.
4. Avoid open tickets with child tickets unless the parent is an epic.
5. Avoid keeping both a broad coordination ticket and its real implementation leaves open at the same time.

## What Is Already Canonical

The following are no longer roadmap aspirations:

1. `ship runtime:report --json` is implemented and documented.
2. extension zones and protected contract zones are documented and doctor-enforced.
3. Cherie compatibility smoke has a named CI lane.
4. doc-sync, dead-route, generator, alpha-surface, and SQL portability/module-isolation lanes exist.
5. `ship module:add` / `ship module:remove` deterministic workspace wiring exists as the baseline install/remove contract.

Any remaining backlog around these areas should describe the next missing layer, not restate the capability already shipped.

## Immediate Direction

Near-term execution should stay concentrated on:

1. making the planning-only promotion/export/import/upgrade surfaces executable;
2. replacing module-wiring TODO placeholders with real battery wiring;
3. tightening contract suites around startup, HTTP, CLI, and managed-mode interoperability;
4. proving standalone exportability and downstream Cherie compatibility continuously;
5. publishing `llm.txt`, MCP scope, and contributor runbooks only after the runtime and CLI contracts they depend on stabilize.

## Definition of Success (v1)

1. A developer can run `ship new myapp` and ship a working app quickly.
2. The same app code can run in single-node or distributed mode by config.
3. Optional batteries are added via CLI without copy-paste or hand wiring.
4. LLMs can reliably reason over framework structure using stable docs and runtime/CLI contracts.
5. Cherie can upgrade to current GoShip with a documented, repeatable process.
