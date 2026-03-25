# AI Agent Guide

This guide is for code agents making changes in this repository.

## Start Here

1. Read `docs/architecture/01-architecture.md`, `docs/architecture/02-structure-and-boundaries.md`, and `docs/architecture/06-known-gaps-and-risks.md`.
2. Read root seam files before changing runtime behavior: `container.go`, `router.go`, and `schedules.go`.
3. Read scoped `CLAUDE.md` files in the directory you are editing.
4. If no scoped guide exists, fall back to this guide.

## Framework-First Runtime Seams

Canonical runtime seams are root-level and must stay synchronized.
This is framework-first runtime seam guidance and should stay consistent across docs and code:

- `container.go`: container composition seam
- `router.go`: route + middleware seam
- `schedules.go`: recurring job registration seam

Do not reintroduce deleted app-shell guidance in canonical architecture docs.

## Architectural Conventions

- HTTP handlers live in `framework/web/controllers` and enabled module route packages.
- Domain logic should prefer framework/module packages (`framework/*`, `modules/*`) over route-level DB logic.
- Rendering is done via framework controller/page/viewmodel contracts (`framework/web/ui`, `framework/web/viewmodels`).
- Templates live under framework-owned templ packages (`framework/web/*`, `framework/views/*`).

## Safe Change Workflow

1. Identify layer to change: routing, service/repository, domain/schema, tooling/docs, or UI/templates.
2. Find related tests (`rg "func Test" framework modules tools`).
3. Make the smallest coherent change first.
4. Run targeted tests, then broader tests when crossing package boundaries.
5. Update docs in the same change stream for behavior/architecture changes.

## Key Files By Concern

Runtime bootstrap:

- `cmd/web/main.go`
- `cmd/worker/main.go`
- `cmd/seed/main.go`

Runtime seams:

- `container.go`
- `router.go`
- `schedules.go`

Routing and middleware:

- `framework/web/controllers/*.go`
- `framework/web/middleware/*.go`
- `modules/*/routes/*.go`

Data and domain:

- `db/queries/*.sql`
- `db/gen/*.go`
- `framework/repos/**/*.go`
- `modules/**/*.go`

UI and rendering:

- `framework/web/ui/*.go`
- `framework/web/viewmodels/*.go`
- `framework/web/**/*.templ`
- `framework/views/**/*.templ`
- `frontend/islands/**/*`

## Common Pitfalls

- Assuming optional adapters are initialized without checking runtime plan wiring.
- Adding route handlers without registration in `router.go`.
- Editing schema/query behavior without migration + generated query updates.
- Updating templ/UI behavior without regenerating templ outputs when needed.

## Commands Commonly Used

- `go run ./tools/cli/ship/cmd/ship dev`
- `make dev`
- `make test`
- `go run ./tools/cli/ship/cmd/ship test`
- `go run ./tools/cli/ship/cmd/ship test --integration`
- `make templ-gen`
- `go run ./tools/cli/ship/cmd/ship db:generate`
- `go run ./tools/cli/ship/cmd/ship db:make your_change`
- `go run ./tools/cli/ship/cmd/ship db:migrate`
- `go run ./tools/cli/ship/cmd/ship db:status`

## Documentation Rule

When behavior or architecture changes, update at least:

- `docs/architecture/03-project-scope-analysis.md` for capability changes
- `docs/architecture/04-http-routes.md` for route surface changes
- `docs/architecture/06-known-gaps-and-risks.md` for risk/known-gap changes
- `docs/reference/01-cli.md` for CLI behavior changes
- `docs/roadmap/01-framework-plan.md` when plan/decision state changes

## Test Tagging Rule

- Integration tests must use `//go:build integration`.
- Keep default tests (`ship test`) fast/stateless when possible.
