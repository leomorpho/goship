# DX + LLM Reliability Phases

Track the multi-phase execution plan for maximizing developer ergonomics and agent reliability.

Last updated: 2026-03-04

## Status Legend

- `done`: implemented and verified
- `in_progress`: currently being implemented
- `next`: queued next, not started
- `later`: explicitly deferred

## Phases

1. `P1` `done` - Establish cognitive model and guardrails docs.
2. `P2` `done` - Implement `ship doctor` baseline checks (layout, markers, package conventions).
3. `P3` `done` - Align `ship new` scaffold to canonical app layout so fresh apps pass doctor.
4. `P4` `done` - Remove legacy `ship check` route compile special-case; rely on explicit package lists.
5. `P5` `done` - Expand enforceable guardrails in `ship doctor` and quality gates.
6. `P6` `done` - Strengthen generator deterministic behavior and wire-safety.
7. `P7` `done` - Add richer docs-to-code sync checks and contributor-facing runbooks.
8. `P8` `later` - Add project upgrade workflows (`ship upgrade`) after command surfaces stabilize.

## Implemented in this phase stream

### P1

- Added cognitive model architecture doc:
  - `docs/architecture/08-cognitive-model.md`

### P2

- Added `ship doctor` command and tests:
  - `cli/ship/doctor.go`
  - `cli/ship/doctor_test.go`
- Wired doctor command/help:
  - `cli/ship/cli.go`

### P3

- Updated `ship new` scaffold to canonical structure:
  - `apps/goship/app/*` (replaces legacy `domains/*`)
  - `apps/goship/foundation/container.go`
  - `apps/goship/web/{controllers,middleware,ui,viewmodels}`
  - `apps/goship/jobs/jobs.go`
  - baseline docs (`docs/00-index.md`, architecture stubs)
- Updated integration tests to validate fresh scaffold + `ship doctor`.

### P4

- Removed legacy special-case compile path from `ship check`.
- Explicit package list now controls compile checks.

## P5 Scope (Current)

1. Added doctor checks for root binary artifact hygiene and `.gitignore` coverage.
2. Added file-length budget enforcement for human-authored `.go` files.
3. Excluded generated paths (`ent/`, `**/gen`) and retained low-noise fix hints.

## P6 Scope (Next)

1. Ensure all generators are deterministic and path-safe.
2. Expand `--wire` safety checks (idempotency, markers, import insertion stability).
3. Add integration tests for generator workflows on fresh scaffold.

Progress note:

- Added router marker-order validation in `ship doctor` (`DX011`) to catch broken `--wire` blocks early.
- Added integration coverage for multi-run `--wire` stability across resource/controller generators.
- Added integration guard that duplicate generation attempts fail without mutating router/route-name wiring.

## P7 Scope

1. Add practical docs coverage checks for core command behavior.
2. Ensure CLI docs + `LLM.txt` stay aligned by default workflow.
3. Expand how-to guides for high-frequency tasks.

Progress note:

- Added `ship doctor` docs-token coverage checks to ensure core CLI commands remain documented in `docs/reference/01-cli.md`.
- Added `ship doctor` required-section coverage checks for `docs/reference/01-cli.md`.
- Replaced `docs/guides/03-how-to-playbook.md` backlog with concrete high-frequency how-to runbooks.
