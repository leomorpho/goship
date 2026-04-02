# PRD — P1-03 Real CRUD Scaffolds

## Objective

Upgrade the starter generation surface from thin route/page/file emission into real CRUD scaffolds that save meaningful app-building time.

## Problem

Today the starter path proves only that generation is not broken.
It does not prove that generation is actually high leverage.

Current state:
- `make:resource` can create a starter-safe route/page shell
- `make:model` creates a query scaffold and prints next steps
- `make:scaffold` is rejected on the starter path
- starter smoke tests only prove buildability and destroy safety

That is truthful, but still far below Rails/Laravel productivity.

## Product Goal

A fresh generated app should be able to scaffold a canonical CRUD slice that includes:
- routes
- pages/views
- create/edit form surfaces
- validation hooks
- starter-safe persistence workflow
- tests

Users should be able to scaffold and refine, not scaffold and rewrite everything.

## Scope

### In scope
- a starter-safe CRUD scaffold path on generated apps
- useful index/show/create/update/delete surface for one canonical resource shape
- validation integration using the new starter validation seam
- generated tests/proof for the CRUD path
- buildability, idempotency, and destroy safety for the upgraded scaffold output
- docs/help alignment if the starter-supported generator surface changes

### Out of scope
- full admin/backoffice generation
- broad policy generation
- fully generic API/HTML parity across every surface
- complete downstream battery integration
- large framework-workspace generator widening unrelated to starter CRUD

## Current Truth Baseline

Today the starter path supports:
- `make:resource`
- `make:model`
- `destroy resource:<name>`

But the emitted output is still too skeletal to count as real CRUD leverage.

`make:scaffold` remains framework-workspace-only for now.

## Target CRUD Contract

### Minimum first useful scaffold
The first credible starter CRUD scaffold should produce:
- list/index page
- create/new page/form
- show page
- edit/update page/form
- delete action path
- route wiring
- route-name constants
- starter-safe page rendering hooks
- starter-safe validation integration
- generated tests proving the scaffold works

### Product standard
The first scaffolded resource should feel like:
- a real starting point for app work
not
- a placeholder someone must largely replace

## Delivery strategy

### Slice A — define the first canonical CRUD target
1. choose one starter-safe resource shape
2. define the route/page/test contract for that target
3. add failing generated-app proof

### Slice B — upgrade generator output
1. widen `make:resource` and/or coordinate with `make:model`
2. add starter-safe form pages and route handlers
3. integrate the P1-02 validation seam
4. keep generated output readable and reversible

### Slice C — persistence/data path coordination
1. make starter CRUD more data-real than the current stub story
2. if needed, coordinate tightly with a narrow slice of P3-02 instead of overbuilding in P1-03

### Slice D — scaffold orchestration decision
1. decide whether starter-safe `make:scaffold` should reopen now
2. if not, keep starter support narrow but make the supported path genuinely valuable
3. if yes, reopen only after proof exists

## Constraints

- generated-app truth outranks broad generator surface claims
- do not reopen starter `make:scaffold` unless the starter path is truly ready
- keep reversibility and destroy safety as first-class requirements
- do not trade clarity for abstraction density

## Acceptance Criteria

- the starter path has a genuinely useful CRUD scaffold flow
- generated CRUD output includes forms, validation hooks, routes, and tests
- generated output remains buildable and destroy-safe
- docs/help truthfully describe the supported starter CRUD surface
- scaffold output clearly reduces manual app wiring relative to today

## Sequencing recommendation

Implement in this order:
1. failing CRUD proof
2. first useful scaffold output
3. validation integration
4. build/destroy/idempotency proof
5. decide whether starter `make:scaffold` can be reopened or must remain narrowed
