# Post-V1 Wave 1 Sequencing

## Objective

Turn the post-v1 roadmap into the narrowest high-leverage first execution wave.

This wave is intentionally biased toward the things most likely to move GoShip from a truthful framework foundation to a genuinely high-productivity app platform:
- richer default app leverage
- stronger generator leverage
- real downstream battery viability
- lower app-builder cognitive overhead
- richer machine-readable contracts for agents

## Wave 1 Rules

- prefer depth on the default generated-app path over breadth across optional surfaces
- keep generated-app truth ahead of framework-internal convenience
- use TDD-first proof lanes for every ticket
- do not widen supported surfaces until installability/runtime truth is real
- avoid new conceptual branches unless they remove more complexity than they add

## Sequence

### Wave 1A — Default app leverage first

#### 1. P1-01 Full auth/account starter
**Why first**
- the current starter is still too skeletal to feel Rails/Laravel-grade
- richer auth immediately raises the minimum app value of `ship new`

**Blocked by**
- nothing substantial

**Enables**
- better CRUD/admin defaults
- richer validation and form work
- stronger starter “real app” story

**Proof first**
- failing generated-app browser proofs for password reset, settings, delete-account, session restore

#### 2. P1-02 First-class validation layer
**Why second**
- CRUD and stronger generators are much less valuable without framework-native validation

**Blocked by**
- none, but should follow P1-01 so starter auth can use the same primitives

**Enables**
- real CRUD scaffolds
- stronger controller/resource generators
- better HTML + JSON parity

**Proof first**
- failing generated-app validation behavior tests on starter forms/controllers

#### 3. P1-03 Real CRUD scaffolds
**Why third**
- this is the first big “days not weeks” multiplier after auth + validation

**Blocked by**
- P1-02

**Enables**
- admin generation
- better model/data generation
- stronger default app leverage story

**Proof first**
- failing end-to-end scaffold proof on a fresh generated app

### Wave 1B — Make batteries real downstream

#### 4. P2-01 Publishable first-party battery packaging
**Why now**
- generated-app batteries are still structurally capped until downstream packaging is real

**Blocked by**
- none

**Enables**
- meaningful expansion of starter battery support
- honest package/versioning story

**Proof first**
- failing temp-dir downstream install proof without repo-local `go.work` / `replace`

#### 5. P2-02 Expand generated-app battery support: storage
**Why storage first**
- it is a common product need and a better leverage multiplier than ornamental breadth

**Blocked by**
- P2-01

**Enables**
- media-heavy apps
- stronger real-world starter viability
- better install/remove mutation seams for other batteries

**Proof first**
- failing add/remove/build/report round-trip on a fresh generated app

### Wave 1C — Generator engine and contract expansion

#### 6. P3-01 Stronger controller/resource generation
**Why here**
- once validation and CRUD exist, generators can stop being thin wrappers and become real force multipliers

**Blocked by**
- P1-02
- materially improved by P1-03

**Enables**
- policy/admin generation
- thinner manual app wiring

**Proof first**
- failing generator round-trip and generated-app scaffold proofs

#### 7. P3-02 Model/data generation
**Why now**
- this is the shortest path to making normal app/domain work much cheaper

**Blocked by**
- none hard, but should follow P1-03 / P3-01 for cohesion

**Enables**
- better CRUD/admin/data ergonomics
- more coherent domain workflow

**Proof first**
- failing end-to-end model/data generation proof on a fresh app

#### 8. P6-01 Richer endpoint metadata
**Why in wave 1**
- agents become dramatically more useful when route export is a real build input, not just an inspection aid

**Blocked by**
- none

**Enables**
- better generated frontend contracts
- stronger agent automation
- clearer auth/policy/validation surface metadata

**Proof first**
- failing route metadata contract tests for auth/request/response/error semantics

### Wave 1D — Simplify the app-builder story

#### 9. P5-01 One dominant app-builder path
**Why after leverage work**
- simplification is most credible once the dominant path is actually more valuable

**Blocked by**
- benefits from P1/P3 progress

**Enables**
- better docs/onboarding/help
- lower cognitive overhead

**Proof first**
- failing doc/help contract tests that detect branching/conflicting default guidance

#### 10. P4-01 Unified request/response DTO workflow
**Why last in wave 1**
- strong payoff, but best shaped after initial auth/validation/CRUD improvements expose the real seams

**Blocked by**
- benefits from P1-02 and P3-01

**Enables**
- thinner controller code
- more coherent API + HTML flow
- better contract generation

**Proof first**
- failing request/response DTO contract tests on starter-generated resources

## Recommended ownership model

### Lane A — Default app leverage
- P1-01
- P1-02
- P1-03

### Lane B — Batteries/installability
- P2-01
- P2-02 (storage first)

### Lane C — Generator/core ergonomics
- P3-01
- P3-02
- P4-01

### Lane D — Contract + simplification
- P6-01
- P5-01

## Exit criteria for Wave 1

Wave 1 is done when all of the following are true:
- the starter app feels materially richer out of the box
- generated CRUD and validation are useful without heavy rewriting
- at least one additional real downstream battery works outside repo-local hacks
- route export is meaningfully richer for agent consumption
- default docs/help clearly present one dominant app-builder path

## Non-goals for Wave 1

Do not spend Wave 1 on:
- broad managed-platform expansion
- ornamental battery breadth before packaging/installability is real
- release/distribution polish ahead of product leverage
- abstract internal architecture cleanup that does not materially improve default app speed

## Brutal decision rule

If a candidate change does not noticeably improve one of these, it should probably not be in Wave 1:
- how much app value `ship new` provides on day one
- how safely and usefully agents can extend the app
- how much generator output reduces manual app wiring
- how easily generated apps consume first-party capabilities downstream
