# CLI Specification (Living)

This file is the living CLI contract for developers and agents.

Short command name:

- `ship`

Module location:

- `cli/ship` (standalone Go module)
- binary entrypoint: `cli/ship/cmd/ship`
- companion MCP module: `mcp/ship` (for LLM-facing tool access)

## Repository Placement

The CLI is in the same repository as the framework and example app.

- Repo model: monorepo with multiple Go modules.
- App/framework module: repository root.
- CLI module: `cli/ship`.
- Workspace: `go.work` includes both modules for local development.

Why this shape:

1. Single repo keeps framework, app example, and CLI evolution in sync.
2. Separate CLI module keeps dependency graph and release surface clean.
3. Developers can iterate across modules locally without publishing interim versions.

Design constraints:

1. Keep commands Rails-like and convention-first.
2. Keep v1 command set intentionally small.
3. Expand only after each command is stable and tested.

## Minimal V1 Command Set

Project lifecycle:

- `ship new <app> [--module <module-path>] [--dry-run] [--force]`
- `ship doctor` (planned)

Local runtime:

- `ship dev` (web-only default)
- `ship dev --worker`
- `ship dev --all`
- `ship check`

Testing:

- `ship test` (unit default)
- `ship test --integration`

Database:

- `ship db create`
- `ship db migrate`
- `ship db rollback`
- `ship db seed`

Generation:

- `ship templ generate [--path <dir>] [--file <file.templ>]`
- `ship generate resource <name> [--path app/goship] [--auth public|auth] [--views templ|none] [--wire] [--dry-run]`
- `ship destroy <generated-artifact>` (planned)

## Versioning Rules

1. CLI-managed tools (for example `templ`) must be pinned to project-declared versions.
2. `ship dev` and `ship test` must never auto-upgrade toolchain versions.
3. `ship doctor` reports version drift and prints explicit fix commands.
4. Only `ship upgrade` (future) may intentionally bump pinned versions.

## Implementation Mapping (Current Repo)

These commands are implemented as wrappers over existing workflows:

- `ship dev` -> `go run ./cmd/web`
- `ship dev --worker` -> `go run ./cmd/worker`
- `ship dev --all` -> starts both processes concurrently with prefixed logs (`[web]`, `[worker]`) and signal-aware shutdown
- `ship check` -> runs Go checks directly; uses package lists in `scripts/test/*.txt` when present, otherwise `go test ./...`
- `ship test` -> `go test ./...` (integration-tagged tests are excluded by default)
- `ship test --integration` -> `go test -tags=integration ./...`
- `ship db create` -> detects `docker-compose`/`docker compose` and runs `up -d cache`, then attempts `up -d mailpit` (non-fatal if mailpit fails)
- `ship db migrate` -> `atlas migrate apply --dir file://ent/migrate/migrations --url <configured>`
- `ship db rollback [amount]` -> `atlas migrate down ... [amount]`
- `ship db seed` -> `go run ./cmd/seed/main.go`
- `ship templ generate --path app` -> `templ generate -path app`, then move each `*_templ.go` into sibling `gen/` directory
- `ship new <app>` -> create minimal deterministic project scaffold in a new directory (no network calls)
- `ship generate resource <name>` -> scaffold handler (+ optional templ page), ensure route-name constant, and print route snippet for manual insertion in `app/goship/router.go`
- `ship generate resource <name> --wire` -> also insert snippet behind ship markers in `app/goship/router.go`
- `ship generate resource <name> --dry-run` -> preview all planned changes without writing files

`ship new` v1 contract:

1. Creates a local scaffold only (no external downloads or package installs).
2. Writes deterministic starter files:
`go.mod`
`app/goship/router.go` (with route marker pairs for `--wire`)
`pkg/routing/routenames/routenames.go`
`app/goship/views/templates.go`
3. Supports `--dry-run` and `--force`.

Generated project workflow:

1. `ship` is the canonical interface for dev/test/check/generate flows.
2. Generated projects must not require a Makefile to use core workflows.
3. Integration tests must use Go build tags (`//go:build integration`) instead of package list curation.

Resource generator contract (v1 minimal):

1. Does not auto-edit router.
2. Optional `--wire` mode inserts snippet behind marker pairs.
Markers:
`// ship:routes:public:start ... // ship:routes:public:end`
`// ship:routes:auth:start ... // ship:routes:auth:end`
3. Creates `app/goship/web/routes/<resource>.go`.
4. Creates `app/goship/views/web/pages/<resource>.templ` when `--views templ`.
5. Ensures `RouteName<Resource>` constant exists in `pkg/routing/routenames/routenames.go`.
6. Prints exact snippet target (`registerPublicRoutes` or `registerAuthRoutes`) when not wiring.

Generated handler behavior:

- `--views templ`: generates a controller/page-rendering handler (`controller.NewPage`, layout assignment, `RenderPage`).
- `--views none`: generates a minimal HTTP string handler for API/prototype paths.

Generator test strategy:

- Unit + integration tests for `cli/ship` run against temporary fixture projects.
- Generator tests do not depend on the live repository app tree.

Local run examples from repository root:

- `go run ./cli/ship/cmd/ship -- help`
- `go run ./cli/ship/cmd/ship -- dev`

## Ownership Boundaries

CLI owns:

- developer command interface (`ship ...`);
- orchestration of dev/test/db workflows;
- version/tooling checks and future generators.

App/framework owns:

- actual runtime behavior in `cmd/*`, `app/goship/*`, `pkg/*`, and `config/*`.

Rule:

- keep business/runtime logic out of CLI package; CLI should call stable commands/APIs.

## Deferred (Not In V1)

- `ship console`
- `ship routes`
- `ship upgrade`
- advanced generator variants
