# Core Interfaces

This document defines the first stable adapter contracts for GoShip's backend-agnostic runtime.

## Purpose

GoShip should let app code stay stable while infrastructure choices vary by environment:

- database (`postgres`, `mysql`, `sqlite`)
- cache (`memory`, `redis`)
- pubsub (`inproc`, `redis`, cloud)
- jobs (`inproc`, `dbqueue`, `asynq`, cloud)
- blob storage (`local`, `s3-compatible`)
- mailer (provider-specific adapters)

To enforce that, app and module code should depend on core interfaces instead of concrete clients.

## Canonical Package

- `pkg/core/interfaces.go`
- `pkg/core/adapters/registry.go`
- `pkg/core/adapters/resolve.go`

This package is the current source of truth for adapter seam contracts.

## Current Contracts (v0)

1. `Store`
- `Ping(ctx)` for health checks
- `WithTx(ctx, fn)` for transaction boundaries

2. `Cache`
- `Get`, `Set`, `Delete`, `InvalidatePrefix`, `Close`

3. `PubSub`
- `Publish`
- `Subscribe` with `MessageHandler`
- `Close`

4. `Jobs`
- `Register`, `Enqueue`
- `StartWorker`, `StartScheduler`, `Stop`
- `Capabilities`
- `EnqueueOptions` supports `timeout`, `run_at`, `max_retries`, and `retention`

5. `BlobStorage`
- `Put`, `Delete`, `PresignGet`

6. `Mailer`
- `Send(MailMessage)`

## Capability Validation

Jobs backends expose `JobCapabilities` (delayed, retries, cron, priority, dead-letter, dashboard).

- `ValidateJobCapabilities(required, available)` fails fast at startup when runtime config requires unsupported features.
- This is the first implementation of the "capability contract" rule from the framework plan.

## Adapter Registry and Startup Guardrails

The runtime now has a canonical adapter registry:

- known adapter names by seam (`db`, `cache`, `jobs`, `pubsub`)
- jobs capability map per backend
- derived requirements from process/runtime config (`pkg/core/adapters/requirements.go`)

Container startup validates:

1. selected adapters are known;
2. selected jobs backend satisfies derived capability requirements.

Current startup behavior:

- invalid selection or capability mismatch fails fast at startup.
- resolved adapter metadata is stored in `services.Container.Adapters` for downstream wiring.
- `services.Container` initializes backend-agnostic seams:
  - `CoreCache` (`core.Cache`) via `services.CoreCacheAdapter`
  - `CoreJobs` (`core.Jobs`) via `services.CoreJobsAdapter`
  - `CorePubSub` (`core.PubSub`) via `services.CorePubSubAdapter`

First migrated call site:

- `pkg/tasks/notifications.go` now enqueues follow-up jobs through `core.Jobs` instead of `*services.TaskClient`.

## Scope Boundaries

These interfaces are runtime seams, not domain/repository APIs.

- Do not force app repos to become generic CRUD wrappers.
- Keep domain modeling in Ent + app/framework repos.
- Use `pkg/core` only where backend swapability or startup validation is required.

## Migration Plan

1. Keep existing concrete packages running (`pkg/services`, `pkg/repos/*`).
2. Add adapters that satisfy `pkg/core` contracts.
3. Move container wiring to resolve adapters via config and capabilities.
4. Gradually convert call sites from concrete clients to interfaces.

## Follow-Ups

1. Add adapter registry/factory by runtime config (`adapter:set` direction).
2. Add core interface conformance tests per adapter package.
3. Define module-level contracts for auth/billing/notifications once module extraction starts.
