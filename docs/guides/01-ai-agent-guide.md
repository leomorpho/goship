# AI Agent Guide

This guide is for code agents making changes in this repository.

## Start Here

1. Read `docs/architecture/03-project-scope-analysis.md` and `docs/architecture/06-known-gaps-and-risks.md`.
2. Inspect route wiring in `app/goship/router.go` before editing handlers.
3. Inspect `app/goship/services/container.go` before assuming a dependency is initialized.

## Architectural Conventions

- HTTP handlers live in `app/goship/web/routes`.
- Domain logic should prefer repository packages (`pkg/repos/...`) over route-level DB logic.
- Rendering is typically done via `controller.Page` + templ components.
- Enums/constants are centralized in `pkg/domain`.

## Safe Change Workflow

1. Identify layer to change:
- routing
- repository/service
- domain/schema
- template/frontend

2. Check for related tests:
- `rg "func Test" pkg/...`
- route tests in `app/goship/web/routes/*_test.go`

3. Implement minimal, local change first.
4. Run targeted tests, then broader tests if needed.
5. Update docs in `docs/` when behavior or architecture changes.

## Key Files By Concern

Runtime bootstrap:

- `cmd/web/main.go`
- `cmd/worker/main.go`
- `cmd/seed/main.go`

Dependency wiring:

- `app/goship/services/container.go`
- `app/goship/services/auth.go`
- `app/goship/services/tasks.go`

Routing and middleware:

- `app/goship/router.go`
- `pkg/middleware/*.go`

Data and domain:

- `app/goship/db/schema/*.go`
- `pkg/repos/**/*.go`
- `pkg/domain/*.go`

UI and rendering:

- `pkg/controller/*.go`
- `app/goship/views/**/*.templ`
- `javascript/**/*`

## Common Pitfalls

- Assuming cache/notifier/task clients are initialized in the container.
- Implementing a route but not registering it in `router.go`.
- Adding schema logic without checking migration/generation workflow.
- Updating frontend behavior without checking templ + JS integration points.

## Commands Commonly Used

- `make dev` (default local dev: infra + web)
- `make dev-full` (web + worker + JS/CSS watchers)
- `make test` (Go tests)
- `make test-integration`
- `go run ./cli/ship/cmd/ship test`
- `go run ./cli/ship/cmd/ship test --integration`
- `make build-js`
- `make build-css`
- `make templ-gen`
- `make ent-gen`
- `make makemigrations name=YourChange`

## Documentation Rule

When code behavior changes, update at least:

- `docs/architecture/03-project-scope-analysis.md` if capability changed
- `docs/architecture/04-http-routes.md` if route surface changed
- `docs/architecture/06-known-gaps-and-risks.md` if a risk was added/removed

## Test Tagging Rule

- Integration tests must use `//go:build integration`.
- Keep default tests (`ship test`) stateless and fast by tagging infra-dependent tests.
