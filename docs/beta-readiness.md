# GoShip Beta Readiness Checklist

This document defines the release gate for the GoShip `beta` label.

`beta` is **pass/fail**, not a vibe:

- `PASS`: every checklist item below is green with fresh evidence on the release candidate branch.
- `FAIL`: any single item is red, stale, or missing evidence.

## How To Use This Checklist

1. Run the evidence commands for each criterion on a fresh clone.
2. Record results in the release PR (links to logs, CI jobs, or artifacts).
3. Mark each criterion `PASS` or `FAIL` with a dated note.
4. Only announce beta when all criteria are `PASS`.

## Criteria

| # | Criterion | PASS definition | FAIL definition | Evidence commands | Docket tickets |
|---|---|---|---|---|---|
| 1 | `ship new` creates a production-credible app | Fresh app boots, has canonical layout, and passes scaffold checks without manual patching | Missing canonical files/layout, boot failures, or manual fixups required | `go test ./tools/cli/ship/internal/commands -run TestFreshApp -count=1` | [TKT-580](../.docket/tickets/TKT-580.md), [TKT-491](../.docket/tickets/TKT-491.md) |
| 2 | Core batteries install/remove cleanly | Supported batteries can be added and removed idempotently with deterministic diffs and compile-ready output | Duplicate wiring, partial uninstall, or non-deterministic outputs | `go test ./tools/cli/ship/internal/commands -run TestStarterJobsModuleRoundTripStaysBuildable -count=1` | [TKT-581](../.docket/tickets/TKT-581.md), [TKT-582](../.docket/tickets/TKT-582.md), [TKT-504](../.docket/tickets/TKT-504.md) |
| 3 | Auth golden flow passes on the generated app | Generated-app proof run covers register -> authenticated surface, logout -> login redirect, protected route redirect, and login recovery | Any broken transition in the core auth user journey | `go test ./tools/cli/ship/internal/commands -run TestFreshAppAuthFlow -count=1` | [TKT-584](../.docket/tickets/TKT-584.md) |
| 4 | Single-binary path works without Redis/Postgres | Default local runtime runs web + jobs + cache on local adapters with no Redis/Postgres dependency | Startup/runtime requires infra services for baseline workflow | `go test ./tools/cli/ship/internal/commands -run TestFreshAppNoInfraDefaultPath -count=1` | [TKT-577](../.docket/tickets/TKT-577.md), [TKT-578](../.docket/tickets/TKT-578.md), [TKT-579](../.docket/tickets/TKT-579.md) |
| 5 | `ship doctor` passes on a fresh clone | Freshly generated app and clean clone pass doctor with zero blocking findings | Doctor reports blocking findings on the documented happy path | `go test ./tools/cli/ship/internal/commands -run TestFreshApp -count=1` | [TKT-580](../.docket/tickets/TKT-580.md) |
| 6 | Getting-started guide completes in under 30 minutes | The documented install/onboarding path is executable from a fresh clone without hidden context | Guide requires hidden context, hand fixes, or the documented install path is stale | `go test ./tools/cli/ship/internal/commands -run TestGettingStartedUsesFreshCloneBuildInstallPath -count=1` | [TKT-585](../.docket/tickets/TKT-585.md) |
| 7 | Upgrade path N-1 -> N works without data loss | Upgrade readiness + apply/report flow succeeds on representative fixtures and preserves data | Upgrade reports are missing, blockers unclear, or upgrade introduces data loss | `go test ./tools/cli/ship/internal/commands -run 'Upgrade|Readiness' -count=1` | [TKT-228](../.docket/tickets/TKT-228.md), [TKT-471](../.docket/tickets/TKT-471.md), [TKT-477](../.docket/tickets/TKT-477.md), [TKT-478](../.docket/tickets/TKT-478.md) |

## Release Decision Rule

GoShip is `beta-ready` only when all seven criteria are `PASS` with evidence captured on the candidate commit.
