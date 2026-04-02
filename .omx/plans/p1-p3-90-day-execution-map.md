# P1–P3 90-Day Execution Map

## Objective

Turn the P1–P3 post-v1 work into a realistic 90-day execution map focused on the highest-leverage moves toward Rails/Laravel-class productivity.

This map assumes the goal is not broad feature accumulation.
The goal is to maximize:
- default app leverage
- generator leverage
- downstream battery reality
- lower cognitive overhead
- reusable machine-readable contracts for agents

## Strategic reading

The next 90 days should not optimize for breadth.
They should optimize for the shortest path to:
- a starter app that feels much more complete
- scaffolds that actually save days of work
- batteries that are real outside repo-local tricks

## Tiers

### Tier 1 — must land first
These items most directly affect app-building speed:
1. `P1-03` Real CRUD scaffolds
2. `P3-01` Stronger controller/resource generation
3. `P3-02` Model/data generation
4. `P3-05` Safer reversible generation

### Tier 2 — next leverage layer
These build on Tier 1 and widen default-app usefulness:
5. `P1-04` Admin/backoffice starter primitives
6. `P3-03` Admin/resource generation
7. `P3-04` Policy/authorization generation
8. `P1-05` Mailer-ready starter

### Tier 3 — battery reality
These turn installable capability into a real downstream story:
9. `P2-01` Publishable first-party battery packaging
10. `P2-02a` Storage battery support
11. `P2-03` Battery contract metadata
12. `P2-04` Battery compatibility matrix

### Tier 4 — broader battery rollout
These come after packaging/installability truth exists:
13. `P2-02b` Notifications battery support
14. `P2-02c` Paid subscriptions battery support
15. `P2-02d` Email subscriptions battery support

## Dependency map

### Hard dependencies
- `P1-03` depends on `P1-02` ✅ complete
- `P3-01` depends on `P1-02`, and benefits strongly from `P1-03`
- `P3-02` depends loosely on `P1-03` / `P3-01` for coherent output shape
- `P3-05` should happen before generator surface grows too wide
- `P3-03` depends on `P1-04` and/or `P3-01`
- `P3-04` depends on stronger resource/controller generation (`P3-01`)
- `P2-02*` depends on `P2-01`
- `P2-03` should happen no later than early `P2-02`
- `P2-04` depends on at least one or two real downstream batteries being supportable

### Soft dependencies
- `P1-05` benefits from stronger generation patterns but can move in parallel if capacity exists
- `P1-04` benefits from `P1-03` because admin UX should not diverge wildly from CRUD UX

## Parallel groups

### Group A — default app + scaffolds
- `P1-03`
- `P3-01`
- `P3-02`
- `P3-05`

### Group B — starter expansion
- `P1-04`
- `P1-05`
- later `P3-03`
- later `P3-04`

### Group C — battery productization
- `P2-01`
- `P2-03`
- `P2-02a`
- `P2-04`
- later `P2-02b/c/d`

## 90-day map

## Days 1–30
### Theme
Make scaffolds and generation materially more valuable.

### Primary targets
1. `P1-03` Real CRUD scaffolds
2. `P3-01` Stronger controller/resource generation
3. `P3-02` Model/data generation
4. `P3-05` Safer reversible generation

### Desired outcome by day 30
- a fresh app can generate a real CRUD feature
- generated resources include useful forms, validation hooks, routes, and tests
- model/data generation feels coherent, not like a stub emitter
- generated output remains safe to re-run and destroy

### Why this first
This is the shortest path to “ship features in days” instead of “hand-wire everything.”

## Days 31–60
### Theme
Expand the starter into a more complete app platform.

### Primary targets
5. `P1-04` Admin/backoffice starter primitives
6. `P3-03` Admin/resource generation
7. `P3-04` Policy/authorization generation
8. `P1-05` Mailer-ready starter

### Desired outcome by day 60
- admin/backoffice work is scaffoldable
- authorization is less repetitive and more generator-driven
- mailers feel native on the starter path
- the starter app is noticeably more app-like and less starter-like

### Why this second
Once core CRUD/generation is useful, the next differentiator is how quickly teams can build internal tools and product workflows.

## Days 61–90
### Theme
Make batteries real downstream capabilities.

### Primary targets
9. `P2-01` Publishable first-party battery packaging
10. `P2-02a` Storage battery support
11. `P2-03` Battery contract metadata
12. `P2-04` Battery compatibility matrix

### Stretch targets if ahead
13. `P2-02b` Notifications battery support
14. `P2-02c` Paid subscriptions battery support
15. `P2-02d` Email subscriptions battery support

### Desired outcome by day 90
- first-party battery packaging is real outside repo-local tricks
- at least one more real downstream battery works cleanly
- battery add/remove is machine-readable and contract-backed
- supported combinations are explicit instead of mysterious

### Why this third
Battery breadth matters, but only after app-authoring leverage is already improving.
Otherwise the repo gets broader before it gets faster.

## Recommended team lanes

### Lane 1 — Scaffold/core generation
- `P1-03`
- `P3-01`
- `P3-02`
- `P3-05`

### Lane 2 — Starter app expansion
- `P1-04`
- `P1-05`
- `P3-03`
- `P3-04`

### Lane 3 — Battery productization
- `P2-01`
- `P2-03`
- `P2-02a`
- `P2-04`
- later `P2-02b/c/d`

## Non-goals for this 90-day window

Do not let this window get diluted by:
- broad release/distribution polish
- managed-platform expansion
- large architectural cleanup not tied to app leverage
- ornamental module breadth before downstream installability is real
- documentation churn that is not directly attached to product-truth changes

## Success criteria at day 90

By the end of this window, GoShip should be able to say:
- the starter app gives much more real product leverage out of the box
- CRUD/resource generation is genuinely useful
- generated output is safer and more reversible
- admin/authorization/mailer flows are closer to first-class
- at least one additional battery beyond jobs is truly downstream-real

## Brutal prioritization rule

If an item does not noticeably increase one of these, it should probably lose priority in this window:
- how much app value `ship new` provides on day one
- how much generator output reduces manual app wiring
- how safely generated code can be evolved/removed
- how real the downstream battery story is outside this repo
