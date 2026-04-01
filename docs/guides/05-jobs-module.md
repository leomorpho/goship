# Jobs Module Guide

This guide defines the install/wiring contract for `modules/jobs`.

Last updated: 2026-03-16

## Goal

- Provide one backend-agnostic jobs seam (`framework/core.Jobs`) with strict runtime backend selection.
- Support exactly one backend at a time:
  - `redis` (Asynq driver)
  - `sql` (DB-backed queue)

## Runtime Contract

`modules/jobs` exposes:

- `Module.Jobs() core.Jobs`
- `Module.Inspector() core.JobsInspector`

The module is composed in `cmd/*`, not in `app/*`:

- [cmd/web/main.go](/Users/leoaudibert/Workspace/pagoda-based/goship/cmd/web/main.go)
- [cmd/worker/main.go](/Users/leoaudibert/Workspace/pagoda-based/goship/cmd/worker/main.go)

Container seam fields:

- [container.go](/Users/leoaudibert/Workspace/pagoda-based/goship/app/foundation/container.go)
  - `CoreJobs core.Jobs`
  - `CoreJobsInspector core.JobsInspector`
  - `Scheduler *cron.Cron`

## Scheduled Jobs Convention

App-owned periodic scheduling lives in `app/schedules/schedules.go` and is wired by
`app/foundation/container.go`.

Rules:

- Register schedules only through `Register(s *cron.Cron, jobsProvider JobsProvider)`.
- Keep schedule callbacks thin: they only enqueue jobs through `core.Jobs`.
- Do not run business logic inline in cron callbacks.
- Keep generated/custom entries inside:
  - `// ship:schedules:start`
  - `// ship:schedules:end`

CLI support:

- `ship make:job <Name>` scaffolds `app/jobs/<name>.go` and `app/jobs/<name>_test.go` with `core.Jobs` / `core.JobHandler` registration helpers; this currently targets the framework workspace and rejects the minimal starter scaffold.
- `ship make:schedule <Name> --cron "<expr>"` inserts a schedule entry at the marker block; this currently targets the framework workspace and rejects the minimal starter scaffold.

## Backend Selection Rules

`modules/jobs/config.go` enforces strict XOR config:

- `BackendSQL` requires `DB` and forbids redis settings.
- `BackendRedis` requires `Redis.Addr` and forbids `DB`.

Validation tests:

- [config_test.go](/Users/leoaudibert/Workspace/pagoda-based/goship/modules/jobs/config_test.go)

## SQL Backend Baseline

Implemented in:

- [client.go](/Users/leoaudibert/Workspace/pagoda-based/goship/modules/jobs/drivers/sql/client.go)
- [core_jobs_sql.go](/Users/leoaudibert/Workspace/pagoda-based/goship/modules/jobs/core_jobs_sql.go)
- [core_jobs_inspector_sql.go](/Users/leoaudibert/Workspace/pagoda-based/goship/modules/jobs/core_jobs_inspector_sql.go)

Current behavior:

- Enqueue persists to `goship_jobs`.
- Worker loop supports claim -> handle -> done/retry/failed.
- Retry uses bounded quadratic backoff.
- Inspector supports normalized list/get via `core.JobRecord`.
- Scheduler is a no-op for now (`Cron=false` capability on SQL backend).

## Redis Backend Baseline

Implemented in:

- [client.go](/Users/leoaudibert/Workspace/pagoda-based/goship/modules/jobs/drivers/redis/client.go)
- [core_jobs_redis.go](/Users/leoaudibert/Workspace/pagoda-based/goship/modules/jobs/core_jobs_redis.go)

Current behavior:

- Enqueue + scheduler delegates to Asynq.
- Worker runtime is still Asynq server wiring in `cmd/worker`.
- Inspector exists as a seam but currently returns "not implemented".
- The admin queue monitor still mounts, but it reports an explicit unavailable state until the Redis inspector grows list/get support.

## Migration Notes

- Legacy `app/foundation/tasks.go` ownership was removed.
- Job infrastructure now lives in `modules/jobs`.
- App code should depend on `core.Jobs` (and `core.JobsInspector` if read access is required), not Asynq types.

## Non-Negotiable Rules

- No Asynq imports outside `modules/jobs/drivers/redis`.
- No direct job infra construction inside `app/*`.
- Module wiring happens in `cmd/*` and writes to container core seams.
- New admin/job UI code must read through `core.JobsInspector`, not backend-specific APIs.
