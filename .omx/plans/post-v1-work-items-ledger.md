# Post-V1 Work-Items Ledger

## Purpose

This ledger turns the post-v1 productivity roadmap into execution-ready work items.

Goal:
- move GoShip from a truthful v1 foundation
- to a framework that can plausibly beat Rails/Laravel for agent-led productivity

Post-v1 priorities remain:
1. richer default app
2. true downstream batteries
3. stronger generators
4. lower cognitive overhead
5. better agent-readable contracts

All work items below follow the same rules used for north-star v1:
- failing proof first
- generated-app behavior beats framework-internal convenience
- prefer one dominant happy path
- narrow before overclaiming
- do not add breadth that weakens the core product story

---

# P1 — Rich default app

## P1-01 — Full auth/account starter
**Paths**
- starter scaffold runtime
- starter auth pages/routes
- generated-app auth proof
- onboarding docs

**Changes needed**
- expand starter auth from minimal cookie flow to a complete account lifecycle
- include login, register, logout, password reset, session restore, settings, delete-account
- improve protected-route redirect semantics and default account UX

**Why**
- the default app is still too thin compared with Rails/Laravel-class app starters

**Relevant context**
- current starter auth flow is real but minimal
- the browser proof already exists and should be expanded, not replaced

**TDD ask**
- start with failing generated-app auth/browser proofs for the missing account lifecycle steps

**Acceptance criteria**
- a fresh starter app exposes a complete auth/account baseline
- auth lifecycle is proven on generated apps, not framework-only surfaces
- docs and proof lanes reflect the richer starter contract

**Verification**
- generated-app auth/browser proof lane
- generated-app command-level proof

## P1-02 — First-class validation layer
**Paths**
- framework validation/request seams
- controller/resource generators
- backend contract docs

**Changes needed**
- add request validation primitives
- add reusable field-error structures
- support HTML and JSON validation defaults
- integrate validation into generated resources/controllers

**Why**
- true app-building speed depends on validation being framework-native

**Relevant context**
- current generated app still leaves too much form/validation plumbing to the user

**TDD ask**
- start with failing validation behavior tests on generated controllers/resources

**Acceptance criteria**
- generated forms/resources validate by default
- validation errors are structured, consistent, and reusable
- generated code avoids repetitive hand-written validation glue

**Verification**
- controller/resource validation tests
- generated-app form proof

## P1-03 — Real CRUD scaffolds
**Paths**
- resource/controller/model generators
- generated pages/routes/tests

**Changes needed**
- produce useful index/show/create/update/delete scaffolds
- include forms, validation hooks, and test coverage
- support both HTML-first and API-first output paths where appropriate

**Why**
- scaffolds must produce immediate app leverage, not just placeholder files

**Relevant context**
- current generators are more truthful now, but still too thin to drive elite productivity

**TDD ask**
- start with failing generated-app CRUD scaffold proofs

**Acceptance criteria**
- generated CRUD is buildable, testable, and actually useful
- users can scaffold and then refine, rather than scaffold and rewrite everything

**Verification**
- generated-app CRUD proof lane
- generator idempotency and reversibility tests

## P1-04 — Admin/backoffice starter primitives
**Paths**
- admin resource scaffolds
- policy/auth integration
- starter-safe admin seams

**Changes needed**
- provide a minimal admin/backoffice baseline
- add admin resource listing/edit/delete patterns
- integrate policies and auth defaults

**Why**
- internal-tool velocity is one of the biggest framework multipliers

**TDD ask**
- start with failing admin flow and auth protection proofs

**Acceptance criteria**
- common backoffice flows are scaffoldable and protected by default
- admin defaults feel product-ready, not demo-only

**Verification**
- generated-app admin proof lane

## P1-05 — Mailer-ready starter
**Paths**
- mailer generator
- mail preview seams
- generated-app runtime wiring

**Changes needed**
- promote mailers from framework-only seam to starter-capable default seam
- keep preview and delivery flows coherent

**Why**
- email remains common app work and should not feel like manual integration every time

**TDD ask**
- start with failing mailer generation + preview proofs

**Acceptance criteria**
- generated mailers are useful on the default path
- preview and delivery contracts are real and documented

**Verification**
- generated mailer proof

---

# P2 — True downstream batteries

## P2-01 — Publishable first-party battery packaging
**Paths**
- `modules/*`
- release/versioning surfaces
- CLI dependency logic
- module docs

**Changes needed**
- define a real downstream-consumable packaging and versioning story for batteries
- remove dependence on repo-local `go.work` / `replace` hacks for battery consumption

**Why**
- batteries are not truly first-class until downstream apps can install them outside this repo

**Relevant context**
- current generated-app support had to be narrowed due to packaging reality

**TDD ask**
- start with failing downstream install proofs outside repo-local assumptions

**Acceptance criteria**
- first-party batteries are consumable by generated apps without repo-local workspace hacks
- docs/help/runtime report reflect that packaging reality

**Verification**
- downstream install proof targets

## P2-02 — Expand generated-app battery support
**Order**
1. storage
2. notifications
3. paidsubscriptions
4. emailsubscriptions

**Paths**
- starter scaffold seams
- `module:add/remove`
- runtime composition
- runtime report and docs

**Changes needed**
- add starter-safe mutation seams for each battery
- prove install/remove/build/report truth for each battery

**Why**
- jobs-only is too narrow for a “batteries-included” productivity framework

**Relevant context**
- do not broaden support before packaging/installability are real

**TDD ask**
- start each battery with a failing generated-app install/remove round-trip proof

**Acceptance criteria**
- each supported battery installs/removes cleanly
- generated apps remain buildable and truthful
- runtime/adoption reporting matches reality

**Verification**
- generated-app battery round-trip proofs

## P2-03 — Battery contract metadata
**Paths**
- module catalog
- `module:add` output
- structured install/remove metadata surfaces

**Changes needed**
- emit machine-readable battery install/remove ownership metadata
- expose mutation impact in a structured form for agents/tools

**Why**
- agent-led development needs inspectable mutation surfaces

**TDD ask**
- start with failing metadata contract tests

**Acceptance criteria**
- battery contracts are explicit, structured, and consumable by tools

**Verification**
- CLI contract tests

## P2-04 — Battery compatibility matrix
**Paths**
- CI workflows
- matrix scripts
- module/runtime tests

**Changes needed**
- define supported battery combinations
- prove them explicitly

**Why**
- productivity falls apart when combinations are mysterious

**TDD ask**
- start with a failing compatibility matrix target

**Acceptance criteria**
- supported battery combinations are explicit and proven

**Verification**
- battery matrix CI lane

---

# P3 — Generator engine as speed core

## P3-01 — Stronger controller/resource generation
**Paths**
- controller/resource generators
- route ownership seams
- generated tests

**Changes needed**
- add validation hooks, richer actions, HTML/API variants, and stronger defaults

**Why**
- generators must be the speed engine, not just file emitters

**TDD ask**
- start with failing generator round-trip proofs

**Acceptance criteria**
- generated controllers/resources are genuinely useful starting points

**Verification**
- generator contract tests
- generated-app proofs

## P3-02 — Model/data generation
**Paths**
- model/query/migration generators
- starter DB workflow

**Changes needed**
- produce stronger model/query outputs and migration coupling

**Why**
- data workflow must be dramatically cheaper to match elite frameworks

**TDD ask**
- start with failing end-to-end model generation proofs

**Acceptance criteria**
- common data work starts from generation rather than hand-written plumbing

**Verification**
- generated-app model/data proofs

## P3-03 — Admin/resource generation
**Paths**
- generator surfaces
- admin seams
- policy hooks

**Changes needed**
- generate admin resource list/edit/delete flows

**Why**
- backoffice speed is one of the clearest productivity wins

**TDD ask**
- start with failing generated admin-flow proofs

**Acceptance criteria**
- admin flows are scaffoldable and useful out of the box

**Verification**
- admin scaffold proof lane

## P3-04 — Policy/authorization generation
**Paths**
- policy seams
- route/resource/auth generators

**Changes needed**
- generate policy files and integration hooks

**Why**
- authorization is too repetitive when handwritten

**TDD ask**
- start with failing policy generation proofs

**Acceptance criteria**
- route/resource auth becomes scaffoldable and consistent

**Verification**
- policy generator proof

## P3-05 — Safer reversible generation
**Paths**
- generator ownership metadata
- destroy semantics
- idempotency tests

**Changes needed**
- strengthen ownership markers and destroy behavior

**Why**
- generated code must stay safely editable and removable

**TDD ask**
- start with failing destroy/idempotency tests

**Acceptance criteria**
- generated output remains reversible and trustworthy

**Verification**
- generator destroy/idempotency lane

---

# P4 — Domain/data ergonomics

## P4-01 — Unified request/response DTO workflow
**Paths**
- framework request/response seams
- controller/resource defaults
- backend contract docs

**Changes needed**
- define request DTO conventions and response DTO conventions
- integrate validation and error handling cleanly

**Why**
- too much controller glue still exists

**TDD ask**
- start with failing request/response contract tests

**Acceptance criteria**
- controller code becomes thinner and more uniform

**Verification**
- request/response contract tests

## P4-02 — Better query/repository conventions
**Paths**
- DB/query patterns
- repository/service conventions

**Changes needed**
- provide stronger conventions and less glue between data and app logic

**Why**
- product work should not require as much manual data plumbing

**TDD ask**
- start with failing data workflow proofs

**Acceptance criteria**
- common repo/service patterns are obvious and fast

**Verification**
- generated-app data workflow proof

## P4-03 — Better stateful app defaults
**Paths**
- pagination/search/filter/sort defaults
- resource/controller layer

**Changes needed**
- add common list/search/filter state patterns as framework defaults

**Why**
- this is a constant repeated app tax today

**TDD ask**
- start with failing resource behavior tests

**Acceptance criteria**
- standard stateful flows are built-in, not bespoke

**Verification**
- resource behavior proof

---

# P5 — Simplify the mental model

## P5-01 — One dominant app-builder path
**Paths**
- onboarding docs
- CLI help
- product docs
- generator/module docs

**Changes needed**
- reduce visible branching between starter / framework-author / advanced modes

**Why**
- too much framework literacy is still required for normal app work

**TDD ask**
- start with failing doc/help sync tests

**Acceptance criteria**
- most users can stay on one obvious path

**Verification**
- onboarding/help/doc contract tests

## P5-02 — Progressive disclosure
**Paths**
- docs/help/reference structure

**Changes needed**
- separate beginner, advanced, and framework-author concerns more clearly

**Why**
- advanced architecture should not burden common use

**TDD ask**
- start with failing doc-structure tests

**Acceptance criteria**
- docs reduce cognitive overhead instead of increasing it

**Verification**
- documentation contract tests

---

# P6 — Agent-first framework contracts

## P6-01 — Richer endpoint metadata
**Paths**
- `routes --json`
- generated TS contract
- backend contract docs

**Changes needed**
- add auth/policy/request/response/error metadata

**Why**
- agents need richer explicit semantics than path tables alone

**TDD ask**
- start with failing endpoint metadata contract tests

**Acceptance criteria**
- route export is a serious agent input surface

**Verification**
- endpoint metadata tests

## P6-02 — Generator ownership metadata
**Paths**
- generators
- destroy semantics
- structured artifact metadata

**Changes needed**
- emit ownership and mutation metadata for generated artifacts

**Why**
- safer agent editing depends on strong generated-code boundaries

**TDD ask**
- start with failing ownership metadata tests

**Acceptance criteria**
- agents can reason about generated code boundaries automatically

**Verification**
- generator metadata tests

## P6-03 — Battery/runtime structured metadata
**Paths**
- runtime report
- describe output
- battery metadata surfaces

**Changes needed**
- enrich machine-readable operational and installability metadata

**Why**
- agent-led workflows still rely on too much human interpretation

**TDD ask**
- start with failing runtime/battery metadata tests

**Acceptance criteria**
- operational reasoning becomes more automatic and less doc-dependent

**Verification**
- runtime/battery metadata tests

---

# P7 — Release/distribution polish

## P7-01 — Published CLI install path
**Paths**
- install docs
- release docs
- packaging/release automation

**Changes needed**
- move beyond clone-and-build as the primary install contract

**Why**
- truthful is not the same as polished

**TDD ask**
- start with a failing published install proof

**Acceptance criteria**
- install path is public, reproducible, and pleasant

**Verification**
- external install proof

## P7-02 — Cleaner release evidence bundle
**Paths**
- release-proof scripts
- artifacts
- beta checklist

**Changes needed**
- provide a cleaner artifact bundle for release review

**Why**
- release signoff should be fast and boring

**TDD ask**
- start with failing artifact-bundle checks

**Acceptance criteria**
- one clean evidence bundle exists per release

**Verification**
- release artifact proof

## P7-03 — Upgrade UX beyond Goose pin
**Paths**
- upgrade command
- fixtures
- migration docs

**Changes needed**
- expand upgrade UX beyond the current pin rewrite surface

**Why**
- upgrade should feel like a product feature, not just a maintainer tool

**TDD ask**
- start with failing broader upgrade fixture tests

**Acceptance criteria**
- upgrade UX is genuinely useful for real users

**Verification**
- upgrade matrix proof

---

# P8 — Self-hosted ops excellence

## P8-01 — Better deployment lanes
**Paths**
- deployment docs
- deployment proof lanes
- topology-aware runtime proofs

**Changes needed**
- make self-hosted deployment smoother and better proven

**Why**
- self-hosted delight is a major product differentiator

**TDD ask**
- start with failing deployment proof targets

**Acceptance criteria**
- self-hosted deploy stories are easy and truthful

**Verification**
- deployment proof lanes

## P8-02 — Promotion/recovery ergonomics
**Paths**
- db report commands
- recovery docs
- evidence/report surfaces

**Changes needed**
- improve operator ergonomics for promotion/recovery flows

**Why**
- high productivity includes faster safe ops, not just coding speed

**TDD ask**
- start with failing recovery proof tests

**Acceptance criteria**
- promotion and recovery are easier to trust and execute

**Verification**
- recovery proof scripts

## P8-03 — Managed interop as a truly proven optional surface
**Paths**
- runtime report
- managed docs
- optional hook surfaces if reintroduced

**Changes needed**
- either fully prove managed endpoint surfaces
- or keep managed scope permanently narrow and explicit

**Why**
- ambiguity here creates expensive operational confusion

**TDD ask**
- start with failing managed-surface tests

**Acceptance criteria**
- managed story is either proven or clearly minimal forever

**Verification**
- managed contract tests

## Recommended first 10 post-v1 tickets

1. P1-01 Full auth/account starter
2. P1-02 First-class validation layer
3. P1-03 Real CRUD scaffolds
4. P2-01 Publishable first-party battery packaging
5. P2-02 Expand generated-app battery support: storage
6. P3-01 Stronger controller/resource generation
7. P3-02 Model/data generation
8. P4-01 Unified request/response DTO workflow
9. P5-01 One dominant app-builder path
10. P6-01 Richer endpoint metadata
