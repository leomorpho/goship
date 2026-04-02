# PRD — Post-V1 Unified Generation / Default App / Batteries

## Objective
Execute the post-v1 program in three ordered programs:
1. Unified Generation Substrate
2. Productive Default App
3. Real Downstream Batteries

## Product thesis
GoShip should deliver Rails/Laravel-class productivity for agent-led development by making generation the primary speed engine, making the starter app a genuinely useful product baseline, and keeping optional product capabilities installable as first-party modules.

## Program 1 — Unified Generation Substrate
### Goals
- Replace generator fragmentation with one shared capability model.
- Make resource/controller/model generation truthful across starter and framework-workspace backends.
- Systematize ownership, rerun, destroy, and drift behavior.

### Acceptance criteria
- Resource/controller/model generation have explicit shared contracts.
- Starter-safe controller generation exists or starter-safe controller capability is provided through the unified resource/controller substrate.
- Model generation is no longer merely a query-stub helper.
- Generator rerun/destroy behavior is explicit, tested, and stable.
- Starter `make:scaffold` remains closed until the reopen gate is satisfied.

## Program 2 — Productive Default App
### Goals
- Make the default generated app materially richer without deepening starter-specific hacks.
- Provide admin/backoffice, policy, and mailer foundations on top of Program 1.

### Acceptance criteria
- Admin/backoffice contract is scaffoldable and starter-safe.
- Policy/authorization generation exists as a first-class contract.
- Mailer-ready starter seams are truthful and test-backed.
- Default-app promise is explicit and documentation-aligned.

## Program 3 — Real Downstream Batteries
### Goals
- Make first-party batteries genuinely downstream-installable.
- Expand starter/generated-app battery reality only after packaging truth is real.

### Acceptance criteria
- Battery packaging/version/install contract is explicit and executable.
- Storage is the first additional real downstream battery beyond jobs.
- Battery metadata is machine-readable.
- Supported battery combinations are proven and documented.
- Batteries remain module-owned, not leaked into core.

## Non-goals
- Reopening starter `make:scaffold` early.
- Expanding starter-only runtime branches as a substitute for generator substrate work.
- Treating the notifications monolith as the long-term architecture target.
- Broadening battery claims before packaging/install proof exists.

## Execution order
1. Program 1
2. Program 2
3. Program 3

## Immediate first tranche
- Stronger controller/resource generation
- Stronger model/data generation
- Safer reversible generation
