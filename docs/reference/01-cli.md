# CLI Specification (Living)

This file is the living CLI contract for developers and agents.

Short command name:

- `ship`

Module location:

- `tools/cli/ship` (standalone Go module)
- binary entrypoint: `tools/cli/ship/cmd/ship`
- companion MCP module: `tools/mcp/ship` (for LLM-facing tool access)

## Repository Placement

The CLI is in the same repository as the framework and example app.

- Repo model: monorepo with multiple Go modules.
- App/framework module: repository root.
- CLI module: `tools/cli/ship`.
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
- `ship doctor`
- `ship agent:setup`
- `ship agent:check`
- `ship agent:status`
- `ship upgrade --to <version> [--dry-run]`

Local runtime:

- `ship dev` (web-only default)
- `ship dev --worker`
- `ship dev --all`
- `ship check`
- `ship infra:up` (or `ship infra` for help)
- `ship infra:down`

Testing:

- `ship test` (unit default)
- `ship test --integration`

Database:

- `ship db:create [--dry-run]` (or `ship db` for help)
- `ship db:make <migration_name>` (or `ship db` for help)
- `ship db:migrate`
- `ship db:status`
- `ship db:reset [--seed] [--force] [--yes] [--dry-run]`
- `ship db:drop [--force] [--yes] [--dry-run]`
- `ship db:rollback`
- `ship db:seed`

Generation:

- `ship templ generate [--path <dir>] [--file <file.templ>]`
- `ship make:resource <name> [--path apps/site] [--auth public|auth] [--views templ|none] [--domain <name>] [--wire] [--dry-run]` (or `ship make` for help)
- `ship make:model <Name> [fields...] [--force]`
- `ship make:controller <Name|NameController> [--actions index,show,create,update,destroy] [--auth public|auth] [--domain <name>] [--wire]`
- `ship make:scaffold <Name> [fields...] [--path apps/site] [--views templ|none] [--auth public|auth] [--api] [--migrate] [--dry-run] [--force]`
- `ship make:module <Name> [--path modules] [--module-base github.com/leomorpho/goship-modules] [--dry-run] [--force]`
- `ship destroy <generated-artifact>` (planned)

Command grammar policy:

1. Canonical commands are namespaced via colon (`db:*`, `make:*`, `infra:*`).
2. Bare namespace commands (`ship db`, `ship make`, `ship infra`) show scoped help.
3. Space-form subcommands (for example `ship db migrate`) are intentionally rejected with a hint to use colon form.

## Versioning Rules

1. CLI-managed tools (for example `templ`) must be pinned to project-declared versions.
2. `ship dev` and `ship test` must never auto-upgrade toolchain versions.
3. `ship doctor` reports version drift and prints explicit fix commands.
4. Only `ship upgrade` may intentionally bump pinned versions.

## Implementation Mapping (Current Repo)

These commands are implemented as wrappers over existing workflows:

- `ship dev` -> `go run ./apps/cmd/web`
- `ship dev --worker` -> `go run ./apps/cmd/worker`
- `ship dev --all` -> starts both processes concurrently with prefixed logs (`[web]`, `[worker]`) and signal-aware shutdown
- `ship check` -> `go test ./...` (compile + unit checks, no integration-tagged tests)
- `ship test` -> `go test ./...` (integration-tagged tests are excluded by default)
- `ship test --integration` -> `go test -tags=integration ./...`
- `ship infra:up` -> detects `docker-compose`/`docker compose` and runs `up -d cache`, then attempts `up -d mailpit` (non-fatal if mailpit fails)
- `ship infra:down` -> detects `docker-compose`/`docker compose` and runs `down`
- `ship db:create` -> validates that target database URL is reachable (`atlas schema inspect --url <resolved>`)
- `ship db:migrate` -> `atlas migrate apply --dir file://apps/db/migrate/migrations --url <resolved>`
- `ship db:status` -> `atlas migrate status --dir file://apps/db/migrate/migrations --url <resolved>`
- `ship db:reset [--seed] [--force] [--yes] [--dry-run]` -> prints plan, runs `atlas schema clean --auto-approve`, then `atlas migrate apply`; optional seed
- `ship db:drop [--force] [--yes] [--dry-run]` -> prints plan, runs `atlas schema clean --auto-approve`
- `ship db:make <migration_name>` -> `atlas migrate diff <migration_name> --dir file://apps/db/migrate/migrations --to ent://apps/db/schema --dev-url sqlite://file?mode=memory&_fk=1`
- `ship db:rollback [amount]` -> `atlas migrate down ... [amount]`
- Atlas is managed by `ship`: it uses `atlas` from `PATH` when present, otherwise auto-installs pinned `ariga.io/atlas/cmd/atlas@v0.27.1` to `.cache/tools/bin/atlas`, and finally falls back to `go run` for zero-friction operation.
- `ship db:seed` -> `go run ./apps/cmd/seed/main.go`

DB URL resolution precedence for db commands:

1. `DATABASE_URL`
2. `config/application.yaml` + `config/environments/<APP_ENV|app.environment>.yaml`

If `PAGODA_DATABASE_URL` is set, CLI fails with an explicit error and asks to use `DATABASE_URL`.
If config resolves to embedded DB mode, `ship db:migrate`/`db:rollback` fail with an explicit error.
- `ship db:reset`/`ship db:drop` refuse non-local DB URLs unless `--force` is provided.
- In `APP_ENV=production|prod`, `ship db:reset`/`ship db:drop` require both `--force` and `--yes`.

Safety matrix:

| Command | Local DB | Non-local DB | Production |
|---|---|---|---|
| `db:reset` | requires `--yes` (or `--dry-run`) | requires `--force` + `--yes` (or `--dry-run`) | requires `--force` + `--yes` |
| `db:drop` | requires `--yes` (or `--dry-run`) | requires `--force` + `--yes` (or `--dry-run`) | requires `--force` + `--yes` |
| `db:create` | safe; supports `--dry-run` | safe; supports `--dry-run` | safe; supports `--dry-run` |
- `ship templ generate --path app` -> `templ generate -path app`, then move each `*_templ.go` into sibling `gen/` directory
- `ship new <app>` -> create minimal deterministic project scaffold in a new directory (no network calls)
- `ship agent:setup` -> generate per-agent allowlist artifacts from `tools/agent-policy/allowed-commands.yaml`
- `ship agent:check` -> fail if generated artifacts drift from canonical allowlist (for pre-commit/CI parity)
- `ship agent:status` -> show best-effort local Codex/Claude/Gemini install status vs repo policy
- `ship make:resource <name>` -> scaffold handler (+ optional templ page), ensure route-name constant, and print route snippet for manual insertion in `apps/site/router.go`
- `ship make:resource <name> --domain <name>` -> generate domain-aware constructor slot (`domainService any`) and route wiring using `nil` placeholder
- `ship make:resource <name> --wire` -> also insert snippet behind ship markers in `apps/site/router.go`
- `ship make:resource <name> --dry-run` -> preview all planned changes without writing files
- `ship make:model <Name>` -> run Ent schema scaffolding (`ent new`) then ORM codegen (`ent generate`)
- `ship make:model <Name> [fields...]` -> write `apps/db/schema/<model>.go` with typed fields, then run ORM codegen (`ent generate`)
- `ship make:controller <Name>` -> generate controller/handler scaffold in `apps/site/web/controllers`
- `ship make:controller <Name> --domain <name>` -> generate domain-aware constructor slot (`domainService any`) and route wiring using `nil` placeholder
- `ship make:controller <Name> --actions ... --wire` -> wire generated routes into `apps/site/router.go` markers
- `ship make:scaffold <Name> ...` -> orchestration command that composes `make:model`, `db:make`, `make:controller --domain <plural_model> --wire`, and optionally `make:resource --domain <plural_model>` / `db:migrate`
- `ship make:module <Name>` -> generate isolated module scaffold in `modules/<name>` with its own `go.mod`, module-facing types/contracts, and service tests
- `ship upgrade --to <version>` -> updates `atlasGoRunRef` pin in `tools/cli/ship/cli.go`
- `ship upgrade --dry-run` -> prints planned pin change without writing files
- current scope: Atlas pin only (expandable later)

Doctor checks (current):

- validates canonical app/layout directories under `apps/site`
- validates required files (router, container, routenames, core docs)
- flags forbidden legacy paths from pre-refactor layout
- validates router marker pairs used by `--wire` generators
- validates router marker ordering (`start` before `end`) for `public` and `auth` sections
- validates package naming conventions in `web/ui` and `web/viewmodels`
- flags unexpected root build artifacts (`web`, `worker`, `seed`, `ship`, `ship-mcp`)
- validates `.gitignore` includes root binary artifact ignore entries
- enforces a line budget for non-generated human-authored `.go` files (target <= 500 lines)
- validates CLI reference docs include core command tokens (`ship new`, `ship doctor`, `ship make:*`, `ship db:migrate`, `ship test --integration`)
- validates agent allowlist artifacts are in sync with `tools/agent-policy/allowed-commands.yaml`

Field syntax for `make:model`:

- `name:type` (for example: `title:string`, `published_at:time`, `is_live:bool`)
- supported types: `string`, `text`, `int`, `bool`, `time`, `float`, `email`, `url`
- use `--force` to overwrite an existing schema file

`ship new` v1 contract:

1. Creates a local scaffold only (no external downloads or package installs).
2. Writes deterministic starter files:
`go.mod`
`config/modules.yaml` (workspace-level module enablement)
`apps/site/router.go` (with route marker pairs for `--wire`)
`apps/site/web/routenames/routenames.go`
`apps/site/views/templates.go`
`apps/site/foundation/container.go`
`apps/site/app/*` (domain skeletons)
`apps/site/web/{controllers,middleware,ui,viewmodels}`
`apps/site/jobs/jobs.go`
`apps/db/{schema,migrate/migrations}`
`docs/00-index.md` and baseline architecture docs
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
3. Creates `apps/site/web/controllers/<resource>.go`.
4. Creates `apps/site/views/web/pages/<resource>.templ` when `--views templ`.
5. Ensures `RouteName<Resource>` constant exists in `apps/site/web/routenames/routenames.go`.
6. Prints exact snippet target (`registerPublicRoutes` or `registerAuthRoutes`) when not wiring.

Generated handler behavior:

- `--views templ`: generates a controller/page-rendering handler (`controller.NewPage`, layout assignment, `RenderPage`).
- `--views none`: generates a minimal HTTP string handler for API/prototype paths.

## Generator test strategy

- Unit + integration tests for `tools/cli/ship` run against temporary fixture projects.
- Generator tests do not depend on the live repository app tree.
- `--wire` generator paths are covered for multi-run stability (no duplicate imports/snippets).
- Duplicate generation attempts are covered to ensure failure does not mutate router or route-name wiring.

Local run examples from repository root:

- `go run ./tools/cli/ship/cmd/ship -- help`
- `go run ./tools/cli/ship/cmd/ship -- dev`

## Ownership Boundaries

CLI owns:

- developer command interface (`ship ...`);
- orchestration of dev/test/db workflows;
- version/tooling checks and future generators.

App/framework owns:

- actual runtime behavior in `apps/cmd/*`, `apps/site/*`, `framework/*`, and `config/*`.

Rule:

- keep business/runtime logic out of CLI package; CLI should call stable commands/APIs.

## Deferred (Not In V1)

- `ship console`
- `ship routes`
- advanced generator variants
