# P1–P3 Groomed Todo Plan

## Purpose

This plan replaces time-based framing with a groomed execution stack that can be turned directly into Ralph action lists.

Use this document to:
- choose the next high-leverage slice
- understand dependencies
- understand what must be proven first
- break each item into Ralph-ready work packets

## Prioritization rule

Always prefer items that most improve:
- default app leverage
- generator leverage
- safe reversibility
- downstream battery reality
- machine-readable contracts

## Tier 1 — scaffolds and generation first

### TODO 1 — P1-03 real CRUD scaffolds
**Outcome**
- a fresh app can scaffold real list/show/create/update/delete flows
- forms, validation, routes, and tests come out useful by default

**Why first**
- this is the shortest path to “ship features fast”

**Depends on**
- P1-02 ✅

**Ralph packet ideas**
- packet A: failing generated-app CRUD scaffold proof
- packet B: scaffold runtime/pages/routes/tests for one canonical resource
- packet C: idempotency and destroy proof

### TODO 2 — P3-01 stronger controller/resource generation
**Outcome**
- resource/controller generation becomes a real speed engine

**Why now**
- CRUD scaffolds and generator quality should evolve together

**Depends on**
- P1-02 ✅
- strongly benefits from TODO 1

**Ralph packet ideas**
- packet A: failing generator round-trip proof
- packet B: richer controller/resource output with validation hooks
- packet C: HTML/API variant proof

### TODO 3 — P3-02 model/data generation
**Outcome**
- model/query/migration generation feels coherent instead of stub-like

**Why now**
- data workflow is still too manual

**Depends on**
- no hard blocker, but should align with TODO 1 and TODO 2

**Ralph packet ideas**
- packet A: failing end-to-end model/data generation proof
- packet B: model/query output upgrade
- packet C: migration coupling and proof

### TODO 4 — P3-05 safer reversible generation
**Outcome**
- generated code is safer to re-run, evolve, and destroy

**Why now**
- generator breadth without reversibility will create long-term drag

**Depends on**
- best done alongside TODO 1–3

**Ralph packet ideas**
- packet A: failing destroy/idempotency proof
- packet B: ownership marker tightening
- packet C: destroy/drift behavior cleanup

## Tier 2 — make the starter feel like more of a real app

### TODO 5 — P1-04 admin/backoffice starter primitives
**Outcome**
- starter gets real backoffice leverage

**Why next**
- internal tools are one of the highest-ROI framework wins

**Depends on**
- benefits from TODO 1

**Ralph packet ideas**
- packet A: failing admin starter proof
- packet B: starter-safe admin routes/pages
- packet C: auth/policy protection proof

### TODO 6 — P3-03 admin/resource generation
**Outcome**
- admin resource flows become scaffoldable

**Why next**
- starter admin primitives should be generator-backed, not hand-wired

**Depends on**
- TODO 5
- benefits from TODO 2

**Ralph packet ideas**
- packet A: failing admin generation proof
- packet B: admin scaffold output
- packet C: generated admin tests/proof

### TODO 7 — P3-04 policy/authorization generation
**Outcome**
- route/resource auth becomes scaffoldable and less repetitive

**Why next**
- auth and admin speed depend on policy generation

**Depends on**
- TODO 2

**Ralph packet ideas**
- packet A: failing policy generation proof
- packet B: policy file generation + hook wiring
- packet C: route/resource protection proof

### TODO 8 — P1-05 mailer-ready starter
**Outcome**
- email features stop feeling like a special integration path

**Why next**
- common product work needs native seams

**Depends on**
- can run in parallel with TODO 5–7

**Ralph packet ideas**
- packet A: failing mailer generation/preview proof
- packet B: starter mail preview/runtime seam
- packet C: doc/proof alignment

## Tier 3 — make batteries real downstream

### TODO 9 — P2-01 publishable first-party battery packaging
**Outcome**
- installable batteries work outside repo-local workspace tricks

**Why after Tier 1/2**
- breadth matters less than product leverage until the base app experience improves

**Depends on**
- none hard

**Ralph packet ideas**
- packet A: failing downstream install proof
- packet B: packaging/versioning surface
- packet C: docs/runtime truth alignment

### TODO 10 — P2-02a storage battery support
**Outcome**
- one additional real battery beyond jobs works on generated apps

**Why first battery**
- storage is a common need and broad leverage multiplier

**Depends on**
- TODO 9

**Ralph packet ideas**
- packet A: failing storage add/remove/build/report proof
- packet B: starter-safe mutation seams
- packet C: runtime report + docs alignment

### TODO 11 — P2-03 battery contract metadata
**Outcome**
- battery mutation becomes inspectable for agents/tools

**Why now**
- downstream batteries should be machine-readable as they become real

**Depends on**
- TODO 9
- ideally alongside TODO 10

**Ralph packet ideas**
- packet A: failing battery metadata contract tests
- packet B: structured add/remove metadata output
- packet C: proof and docs alignment

### TODO 12 — P2-04 battery compatibility matrix
**Outcome**
- supported battery combinations are explicit and proven

**Why now**
- combinations become risky once multiple batteries are real

**Depends on**
- TODO 10
- ideally after TODO 11 starts landing

**Ralph packet ideas**
- packet A: failing matrix target
- packet B: supported-combo CI lane
- packet C: documentation of supported combinations

## Tier 4 — broader battery rollout

### TODO 13 — P2-02b notifications battery support
**Depends on**
- TODO 9
- ideally TODO 11 and TODO 12

### TODO 14 — P2-02c paid subscriptions battery support
**Depends on**
- TODO 9
- ideally TODO 11 and TODO 12

### TODO 15 — P2-02d email subscriptions battery support
**Depends on**
- TODO 9
- ideally TODO 11 and TODO 12

These three should each be split the same way:
- packet A: failing install/remove/build/report proof
- packet B: starter/generated-app mutation seams
- packet C: metadata/docs/runtime truth

## Parallelizable groups

### Group A — scaffold engine
- TODO 1
- TODO 2
- TODO 3
- TODO 4

### Group B — starter expansion
- TODO 5
- TODO 6
- TODO 7
- TODO 8

### Group C — battery productization
- TODO 9
- TODO 10
- TODO 11
- TODO 12
- later TODO 13–15

## Best next Ralph-ready stack

If choosing the next action list right now, the strongest sequence is:
1. TODO 1 — P1-03 real CRUD scaffolds
2. TODO 2 — P3-01 stronger controller/resource generation
3. TODO 3 — P3-02 model/data generation
4. TODO 4 — P3-05 safer reversible generation

That is the highest-leverage block.

## Non-goals while executing this plan

Do not let execution drift into:
- release/distribution polish
- managed-platform expansion
- low-value breadth
- broad cleanup disconnected from app leverage
- module breadth before downstream installability is real
