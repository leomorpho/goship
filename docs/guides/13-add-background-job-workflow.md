# Add Background Job Workflow

Canonical contributor workflow for adding a background job in GoShip.

## Goal

Add a job scaffold, register it through `core.Jobs`, and keep worker/runtime wiring aligned with the jobs module seam.

## Use This Workflow When

- You need a new app-owned background job.
- You want the canonical `app/jobs` scaffold and baseline test file.
- You need to validate worker/runtime wiring or scheduling changes.

## Preferred Path

Generate the job scaffold:

```bash
go run ./tools/cli/ship/cmd/ship make:job BackfillUserStats
```

If the job also needs a schedule entry:

```bash
go run ./tools/cli/ship/cmd/ship make:schedule BackfillUserStats --cron "0 0 * * *"
```

## What This Should Change

- `app/jobs/<name>.go`
- `app/jobs/<name>_test.go`
- optional runtime registration where the app registers jobs against `core.Jobs`
- optional schedule wiring under `app/schedules/schedules.go`

## Verification

```bash
go test ./app/jobs ./tools/cli/ship/internal/generators -count=1
go run ./tools/cli/ship/cmd/ship doctor
```

If runtime registration changed:

```bash
go test ./cmd/web ./cmd/worker -count=1
```

## Common Failure Modes

1. Backend-specific drift: keep job registration on `core.Jobs` / `core.JobHandler`, not direct driver types.
2. Missing runtime registration: generated job files are scaffolds, so register them where the app boots workers.
3. Schedule callbacks doing too much: keep cron wiring thin and enqueue through the jobs seam.

## Related References

- `docs/reference/01-cli.md`
- `docs/architecture/07-core-interfaces.md`
- `docs/guides/05-jobs-module.md`
- `docs/guides/02-development-workflows.md`
