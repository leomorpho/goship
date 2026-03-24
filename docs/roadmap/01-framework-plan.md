# GoShip Framework Plan

This is the canonical roadmap for GoShip.

If another roadmap file disagrees with this one, this file wins.

## What GoShip Is

GoShip is a convention-first Go web framework for shipping full products fast.

The target is not "a flexible starter kit."

The target is:

- Rails/Laravel-grade developer experience
- strong defaults
- batteries that install cleanly
- a fast standalone path
- a clean upgrade path to managed and multi-process operation

## What GoShip Is Not

GoShip is not:

- a dumping ground for one internal app
- a control plane
- a hosted-platform dependency
- a framework that asks every project to reinvent structure

The public repo must stay framework-first.

## End State

The desired end state is:

1. `ship new` creates a beautiful, production-credible GoShip app with almost no manual cleanup.
2. `ship dev`, `ship test`, `ship doctor`, `ship verify`, and `ship upgrade` form a trustworthy everyday loop.
3. batteries such as auth, admin, jobs, storage, notifications, profile, i18n, PWA, and billing install with deterministic wiring and removal.
4. the standalone single-node path is excellent by default.
5. the distributed/managed path is enabled through stable contracts, not hidden coupling.
6. downstream apps, including the control plane, consume GoShip like normal GoShip projects.

## Current State

GoShip is no longer early-stage concept work.

Current backlog shape from `.docket/manifest.json`:

- `263 done`
- `77 in-review`
- `41 in-progress`
- `9 backlog`

That means the framework is mostly in execution, hardening, and productization mode.

The biggest mismatch today is not "missing architecture."

The biggest mismatch is:

- stale roadmap language
- stale generator assumptions
- last-mile DX gaps
- some partially-finished operator flows

## Architectural Rules

### 1. Keep The Repo Framework-First

The public GoShip repo must not contain an internal application shell.

Canonical runtime ownership now lives in:

- repo root for runtime entrypoints
- `framework/` for reusable runtime contracts
- `modules/` for installable capabilities
- `tools/cli/ship/` for the product surface

Starter layout concerns belong in generator templates and downstream apps, not in the framework repo itself.

### 2. Capability vs Authority

GoShip owns capability.

An external control plane owns authority.

GoShip must remain fully useful as standalone software.

The control plane may rely on:

- runtime reports
- managed hook contracts
- managed override semantics
- upgrade readiness payloads
- backup/restore evidence

The control plane must not be a framework dependency.

### 3. Single-Node First, Then Promote

The default GoShip story should feel effortless:

- local single process
- SQLite-first
- minimal required services
- fast boot
- fast test loop

Then the framework must provide a clean path to:

- Postgres
- Redis or equivalent capability backends
- worker separation
- scheduler separation
- managed orchestration

### 4. Conventions Must Beat Flexibility

The framework should prefer:

- one obvious directory layout
- one obvious route composition model
- one obvious generator output style
- one obvious way to add batteries
- one obvious verify/doctor contract

Escape hatches are allowed.

But the happy path must be much cheaper than the custom path.

## Product Priorities

The roadmap should be driven by the user-facing product surface developers feel first.

### Priority 1: Golden Path DX

This is the highest priority.

Developers should be able to:

- create a project
- run it
- install batteries
- generate common artifacts
- verify the project
- understand failures quickly

Without fighting repo structure or hidden contracts.

Priority work:

- `ship dev` and local runtime loop
- `ship doctor` and `ship verify`
- route/runtime contract checks
- generator clarity
- better defaults and better error messages

### Priority 2: Deterministic Batteries

Modules must feel like first-class framework features, not repo surgery.

Priority work:

- `ship module:add`
- `ship module:remove`
- concrete wiring for `notifications`, `admin`, `jobs`, and `storage`
- deterministic config/runtime mutation
- boundary enforcement so modules do not punch through framework seams

### Priority 3: Reversible Generators

Rails/Laravel-level DX requires strong generation and destruction semantics.

Priority work:

- `ship make:*` stays convention-first
- `ship destroy` becomes real and trustworthy
- islands generation stays explicit and testable
- generated output should match current architecture, not historical repo shapes

### Priority 4: Standalone Production Credibility

The OSS framework must be honestly usable by self-managed teams.

Priority work:

- startup contract enforcement
- profile/adapter mutation safety
- DB promotion flow
- backup/restore evidence
- narrow but trustworthy browser and CLI golden suites

### Priority 5: Managed Interop

Managed operation matters, but only after the standalone path is solid.

Priority work:

- stable runtime handshake
- `runtime-contract-v1`, `runtime-handshake-v1`, and `upgrade-readiness-v1` remain the only supported contract identifiers until an explicit version bump lands with migration coverage
- `ship runtime:report --json`, `ship upgrade --json`, and `ship verify` must surface blocking mismatch diagnostics such as `unsupported_runtime_contract_version`, `unsupported_runtime_handshake_version`, and `unsupported_upgrade_readiness_version`
- stable managed hook contracts
- shared signature vectors and one canonical payload library for managed-hook signing
- managed override UX/read-only semantics
- runtime metadata for adoption/divergence
- upgrade readiness and orchestration preflight
- `staged-rollout-decision-v1` stays the control-plane input contract, including `policy_input_version`, while rollout engine and traffic shaping stay out of the framework repo
- cross-lane dependency matrix and must-finish-before contract map stay explicit in the canonical roadmap so runtime-contract-v1, upgrade-readiness-v1, promotion-state-machine-v1, backup-manifest-v1, restore_evidence.record_links, and staged-rollout-decision-v1 can be sequenced without ambiguity

Not priority:

- shipping a hosted platform inside GoShip
- fleet policy logic in the framework repo

## Production Readiness View

Beta gate definition now lives in `docs/beta-readiness.md` and is the release decision checklist for beta labeling.

### OSS GoShip

Current honest assessment:

- credible alpha
- approaching beta on the standalone path
- not yet at the promise level implied by a Rails/Laravel comparison

Main remaining gaps:

- generator reversibility
- module wiring polish
- upgrade/readiness/operator flows
- broader confidence in golden-path browser behavior
- cleanup of stale docs and template assumptions

### Control Plane Interop

Current honest assessment:

- contract-ready in many areas
- not a complete product from this repo alone

That is acceptable and intentional.

The control plane should be built as a GoShip app in its own repo and consume GoShip contracts like any other downstream app.

## Cross-Lane Dependency Matrix

The cross-lane dependency matrix and must-finish-before contract map are the canonical coordination surface for production-system work.

| Lane | Requires | Must finish before | Why |
| --- | --- | --- | --- |
| runtime contracts | `runtime-contract-v1`, `runtime-handshake-v1` | managed orchestration | Control-plane deploy logic can only consume one stable runtime contract at a time. |
| upgrade readiness | `upgrade-readiness-v1` | fleet upgrades | Managed upgrade planning needs one stable readiness payload before orchestration fans out. |
| promotion and backup | `promotion-state-machine-v1`, `backup-manifest-v1`, `restore_evidence.record_links` | disaster recovery automation | Restore and promotion evidence must be machine-readable before control-plane recovery flows are trustworthy. |
| rollout policy inputs | `staged-rollout-decision-v1`, `policy_input_version` | canary and promote decisions | Runtime facts and control-plane policy must speak one shared verdict schema before rollout automation is allowed. |

## Pagoda Intake Governance

Pagoda intake governance stays on a weekly or per-tag cadence, and each upstream change is classified as adopt, adapt, or skip in `docs/roadmap/09-pagoda-intake-log.md`.

## Canonical Repo Shape

Inside the GoShip repo, the important ownership model is:

- `container.go`, `router.go`, `schedules.go`:
  canonical runtime entrypoints
- `framework/`:
  reusable framework contracts and shared runtime behavior
- `modules/`:
  installable capabilities
- `cmd/`:
  process entrypoints
- `static/`, `styles/`, `testdata/`:
  runtime assets and fixtures
- `tools/cli/ship/`:
  generator, doctor, verify, upgrade, and operator-facing workflow surface

The root Go files are intentional.

They replaced the dead internal `app/` shell.

They should stay unless the runtime entrypoint surface is moved into a different clearly-owned package on purpose. They are not random leftovers.

## Roadmap Lanes

The roadmap should now be treated as four execution lanes.

### Lane A: Framework DX And Quality Gates

Goal:

- make the core developer loop feel sharp, strict, and pleasant

Representative backlog:

- `TKT-188`
- `TKT-194`
- `TKT-196`
- `TKT-199`
- `TKT-207`
- `TKT-381`
- `TKT-382`

Success means:

- one-command local loop
- strict startup and repo contracts
- reliable browser and CLI golden coverage
- low ambiguity when something breaks

### Lane B: Runtime Capabilities And Standalone Operation

Goal:

- make GoShip excellent as standalone software before anything hosted

Representative backlog:

- `TKT-189`
- `TKT-198`
- `TKT-200`
- `TKT-201`
- `TKT-202`
- `TKT-203`
- `TKT-228`

Success means:

- single-node local truth is excellent
- profile and adapter mutation is deterministic
- promotion/export/import/verify are executable and legible

### Lane C: Generators And Batteries

Goal:

- make the product surface feel like a real framework

Representative backlog:

- `TKT-191`
- `TKT-204`
- `TKT-205`
- `TKT-388`
- `TKT-389`
- `TKT-390`

Success means:

- generators emit the current canonical structure
- batteries wire themselves cleanly
- destruction/removal is real
- downstream apps do not accumulate garbage after framework actions

### Lane D: Runtime / Control-Plane Contract

Goal:

- support external orchestration without creating framework coupling

Representative backlog:

- `TKT-213`
- `TKT-214`
- `TKT-220`
- `TKT-230`
- `TKT-247`

Success means:

- one stable runtime handshake
- one stable managed contract surface
- one stable upgrade/readiness surface
- no hidden private-path coupling

## What Should Be De-Emphasized

The roadmap should explicitly de-emphasize:

- historical Cherie-specific framing
- old internal-app assumptions
- broad planning tickets that do not lead to executable leaves
- framework-owned marketing/demo behavior
- hosted-platform ambition ahead of OSS framework completeness

## Immediate Roadmap Rewrite Decisions

These decisions are now canonical:

1. `docs/roadmap/01-framework-plan.md` is the only canonical roadmap file.
2. Other `docs/roadmap/*` files are supporting execution notes, historical context, or task breakdowns.
3. Any roadmap/task doc that still assumes an internal `app/` layout is stale and should be updated or treated as historical.
4. GoShip should optimize for a beautiful downstream generated app, not for carrying one internally forever.
5. The control plane should be treated as a downstream GoShip app, not a hidden subdirectory in GoShip.

## Next Must-Finish Work

If we are serious about the stated end state, the highest-value next work is:

1. Align generator output and starter templates with the new runtime structure.
2. Finish battery wiring for `notifications`, `admin`, `jobs`, and `storage`.
3. Finish `ship destroy`.
4. Finish the golden browser suite and verify tiers.
5. Finish upgrade/readiness and promotion/import/verification operator flows.
6. Keep managed interop contract-only until the standalone UX feels complete.

## Relationship To Other Roadmap Files

Interpret the other roadmap files like this:

- `02-architecture-evolution.md`: supporting architecture notes, not canonical repo truth
- `03-atomic-tasks.md`: detailed execution backlog, not product strategy
- `04-pagoda-and-dx-improvements.md`: input log and ideas, not canonical priority
- `05-llm-dx-agent-friendly.md`: contributor ergonomics notes
- `06-dx-and-infrastructure.md`: execution notes for specific streams
- `07-modules-and-capabilities.md`: module-specific execution notes
- `08-ui-agent-context.md`: UI workflow and agent context
- `09-pagoda-intake-log.md`: adopt/adapt/skip record

If those files conflict with current repo shape or current product direction, update them or ignore them until rewritten.
