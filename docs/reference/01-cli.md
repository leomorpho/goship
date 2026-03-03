# CLI Specification (Living)

This file is the living CLI contract for developers and agents.

Short command name:

- `ship`

Module location:

- `cli/ship` (standalone Go module)
- binary entrypoint: `cli/ship/cmd/ship`

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

Local run examples from repository root:

- `go run ./cli/ship/cmd/ship -- help`
- `go run ./cli/ship/cmd/ship -- dev`

## Deferred (Not In V1)

- `ship console`
- `ship routes`
- `ship upgrade`
- advanced generator variants
