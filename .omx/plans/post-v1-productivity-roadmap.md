# Post-V1 Productivity Roadmap

## Objective

Move GoShip from a truthful, proven north-star v1 foundation to a framework that can plausibly beat Rails/Laravel for agent-led productivity.

The target is not just correctness or honesty.
The target is:
- richer default app value
- real downstream batteries
- stronger generators
- lower cognitive overhead
- more agent-readable machine contracts

## Current Product Judgment

GoShip is now a credible, test-backed framework foundation.
It is not yet a top-tier productivity framework.

Main remaining gaps:
1. the default app is still too thin
2. batteries are still too weak downstream
3. generators are not yet the main speed engine
4. the conceptual surface is still too fragmented
5. agent-readable contracts are improving but not yet pervasive

## Post-V1 Milestones

### P1 — Rich default app
Goal: make `ship new` feel like a real app platform, not just a clean starter.

Key workstreams:
- full auth/account starter
- first-class validation layer
- real CRUD scaffolds
- admin/backoffice starter primitives
- mailer-ready starter

### P2 — True downstream batteries
Goal: make first-party batteries genuinely installable and useful outside this repo.

Key workstreams:
- publishable first-party battery packaging
- expand generated-app battery support
- battery contract metadata
- battery compatibility matrix

### P3 — Generator engine as speed core
Goal: make generators the main productivity multiplier.

Key workstreams:
- stronger controller/resource generation
- model/data generation
- admin/resource generation
- policy/authorization generation
- safer reversible generation

### P4 — Domain/data ergonomics
Goal: reduce glue code across DB, domain, route, and UI layers.

Key workstreams:
- unified request/response DTO workflow
- better query/repository conventions
- better stateful app defaults

### P5 — Simplify the mental model
Goal: make there be one dominant app-builder path.

Key workstreams:
- one dominant app-builder path
- progressive disclosure for advanced/framework-author concerns

### P6 — Agent-first framework contracts
Goal: make GoShip more machine-readable than classic frameworks.

Key workstreams:
- richer endpoint metadata
- generator ownership metadata
- battery/runtime structured metadata

### P7 — Release/distribution polish
Goal: make adoption feel polished, not just truthful.

Key workstreams:
- published CLI install path
- cleaner release evidence bundle
- upgrade UX beyond Goose pin rewrites

### P8 — Self-hosted ops excellence
Goal: make self-hosted production operation easy, not merely possible.

Key workstreams:
- better deployment lanes
- promotion/recovery ergonomics
- managed interop as a truly proven optional surface

## First 10 Tickets To Start

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

## Hard Prioritization Rule

Only prioritize work that increases one of these:
- default app leverage
- downstream battery power
- generator power
- lower cognitive overhead
- agent-readable machine contracts

## Recommendation

If GoShip wants to beat Rails/Laravel for productivity, the immediate post-v1 bet should be:
1. richer default app
2. stronger generators
3. real downstream batteries

Those three together are the shortest path to dramatically higher app-building speed.
