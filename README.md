# GoShip

Convention-first, Rails/Laravel-inspired Go framework for shipping production-ready web apps fast.

Last updated: 2026-03-25

[![Test](https://github.com/leomorpho/GoShip/actions/workflows/test.yml/badge.svg)](https://github.com/leomorpho/GoShip/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## What GoShip Is

GoShip is a framework-first repository with one canonical runtime seam set and a batteries-included CLI.

- convention-first project structure and generators (`ship make:*`, `ship destroy`)
- server-first web stack (Go + Echo + Templ + HTMX)
- supported first-party batteries for the current v1 generated-app path: jobs
- standalone-first runtime with an explicit path to distributed/managed operation

## Product Status

GoShip is in late alpha / beta-hardening mode.

- The standalone path is credible today.
- The release gate for a public beta label is explicit in [`docs/beta-readiness.md`](docs/beta-readiness.md).
- Managed/control-plane operation is contract-driven, but external authority logic stays out of this repo.

## Start Here

Requirements:

- Go
- Make
- Node.js (for frontend/e2e workflows)
- Docker (optional for standard infra-backed mode)

Quick start:

```bash
cp .env.example .env

# fastest local loop (single binary)
make run

# canonical dev loop
make dev
# equivalent direct CLI command
go run ./tools/cli/ship/cmd/ship dev
```

Common commands:

- `make run`: single-binary SQLite + Otter + Backlite runtime
- `make dev`: canonical local dev wrapper (`ship dev`)
- `go run ./tools/cli/ship/cmd/ship dev`: direct invocation of the canonical dev loop
- `make test`: Docker-free unit package set
- `go run ./tools/cli/ship/cmd/ship test`: canonical CLI test surface
- `make templ-gen`: templ generation
- `go run ./tools/cli/ship/cmd/ship doctor`: project health checks
- `go run ./tools/cli/ship/cmd/ship verify`: runtime contract checks

## `ship new` Default Story

The default `ship new` output is a minimal starter app, not the full module-capable framework workspace.

- starter route surface: landing, auth, home, and profile
- installable batteries: `ship module:add` is not supported in a fresh starter app
- canonical first-boot sequence: `ship db:migrate`, then `ship dev`

Move to the full framework workspace shape if you need installable batteries or framework-authoring surfaces.

## Runtime Seams (Canonical)

Framework runtime ownership is explicit under `app/`: `app/container.go`, `app/router.go`, `app/schedules.go`.

- `app/container.go`: runtime container composition seam
- `app/router.go`: HTTP route + middleware composition seam
- `app/schedules.go`: recurring schedule registration seam

This keeps wiring deterministic for humans, generators, and LLM agents.

## Default Path

GoShip must remain fully usable as standalone software.

- Default path: single-binary, SQLite-first, no required control plane
- Upgrade path: profile/adapter promotion to Postgres/Redis/worker separation
- Managed path: external control plane consumes stable runtime contracts (reports, hooks, upgrade readiness), without becoming a runtime dependency

See [`docs/architecture/09-standalone-and-managed-mode.md`](docs/architecture/09-standalone-and-managed-mode.md).

## Frontend Story

GoShip is server-first, with optional islands when interactivity needs client runtime support.

- Templ + HTMX is the default and recommended path
- Vite pipeline supports vanilla JS plus per-island React/Vue/Svelte mounts
- Islands are explicit and route-local, not SPA-first by default

Blessed split-frontend contract:

- contract id: `api-only-same-origin-sveltekit-v1`
- supported custom frontend scope: `SvelteKit-first`
- browser boundary: `same-origin auth/session` with `cookie/CSRF` preserved

See [`docs/guides/08-building-an-api.md`](docs/guides/08-building-an-api.md) and [`examples/sveltekit-api-only/README.md`](examples/sveltekit-api-only/README.md).

## Repository Shape

GoShip is framework-first:

- `app/container.go`, `app/router.go`, `app/schedules.go`: canonical runtime seams
- `framework/`: reusable framework runtime contracts and implementations
- `modules/`: installable capabilities
- `cmd/`: runtime process entrypoints
- `tools/cli/ship/`: canonical CLI product surface
- `docs/`: architecture, guides, reference, roadmap, and release-gate docs

## Documentation

Use these entry docs first:

- [`docs/00-index.md`](docs/00-index.md)
- [`docs/architecture/01-architecture.md`](docs/architecture/01-architecture.md)
- [`docs/architecture/09-standalone-and-managed-mode.md`](docs/architecture/09-standalone-and-managed-mode.md)
- [`docs/guides/01-ai-agent-guide.md`](docs/guides/01-ai-agent-guide.md)
- [`docs/reference/01-cli.md`](docs/reference/01-cli.md)
- [`docs/roadmap/01-framework-plan.md`](docs/roadmap/01-framework-plan.md)
- [`docs/beta-readiness.md`](docs/beta-readiness.md)
