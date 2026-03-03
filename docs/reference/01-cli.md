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

- `ship new <app>` (planned)
- `ship doctor` (planned)

Local runtime:

- `ship dev` (web-only default)
- `ship dev --worker`
- `ship dev --all`

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
- `ship generate <resource|model|job|module>` (planned)
- `ship destroy <generated-artifact>` (planned)

## Versioning Rules

1. CLI-managed tools (for example `templ`) must be pinned to project-declared versions.
2. `ship dev` and `ship test` must never auto-upgrade toolchain versions.
3. `ship doctor` reports version drift and prints explicit fix commands.
4. Only `ship upgrade` (future) may intentionally bump pinned versions.

## Implementation Mapping (Current Repo)

These commands are implemented as wrappers over existing workflows:

- `ship dev` -> `make dev`
- `ship dev --worker` -> `make dev-worker`
- `ship dev --all` -> `make dev-full`
- `ship test` -> `make test`
- `ship test --integration` -> `make test-integration`
- `ship db create` -> `make up`
- `ship db migrate` -> `make migrate`
- `ship db rollback [amount]` -> `atlas migrate down ... [amount]`
- `ship db seed` -> `make seed`
- `ship templ generate --path app` -> `templ generate -path app`, then move each `*_templ.go` into sibling `gen/` directory

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
