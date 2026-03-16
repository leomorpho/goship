# CLI Specification (Living)

This file is the living CLI contract for developers and agents.

Short command name:

- `ship`

Module location:

- `tools/cli/ship` (standalone Go module)
- binary entrypoint: `tools/cli/ship/cmd/ship`
- companion MCP module: `tools/mcp/ship` (for LLM-facing tool access)

## Repository Placement

The CLI is in the same repository as the framework and canonical single app runtime.

- Repo model: monorepo with multiple Go modules.
- App/framework module: repository root.
- CLI module: `tools/cli/ship`.
- Workspace: `go.work` includes both modules for local development.

Why this shape:

1. Single repo keeps framework, app runtime, and CLI evolution in sync.
2. Separate CLI module keeps dependency graph and release surface clean.
3. Developers can iterate across modules locally without publishing interim versions.

Design constraints:

1. Keep commands Rails-like and convention-first.
2. Keep v1 command set intentionally small.
3. Expand only after each command is stable and tested.

## Minimal V1 Command Set

Project lifecycle:

- `ship new <app> [--module <module-path>] [--dry-run] [--force] [--i18n|--no-i18n]`
- `ship doctor [--json]`
- `ship config:validate [--json]`
- `ship routes [--json]`
- `ship api:spec [--out <path>] [--serve]`
- `ship i18n:missing`
- `ship i18n:unused`
- `ship agent:setup`
- `ship agent:check`
- `ship agent:status`
- `ship agent:start --task "<description>" [--id TASK-001]`
- `ship agent:finish --id TASK-001 --message "feat(scope): summary" [--pr]`
- `ship upgrade --to <version> [--dry-run]`

Local runtime:

- `ship dev` (auto default: web-only for single-binary adapters; full mode when jobs adapter is `asynq`)
- `ship dev --worker`
- `ship dev --all`
- `ship check`
- `ship infra:up` (or `ship infra` for help)
- `ship infra:down`
- `ship run:command <name> [-- <args...>]`

Testing:

- `ship test` (unit default)
- `ship test --integration`

Database:

- `ship db:create [--dry-run]` (or `ship db` for help)
- `ship db:generate [--config <path>] [--dry-run]`
- `ship db:make <migration_name> [--soft-delete --table <table>]` (or `ship db` for help)
- `ship db:migrate`
- `ship db:status`
- `ship db:reset [--seed] [--force] [--yes] [--dry-run]`
- `ship db:drop [--force] [--yes] [--dry-run]`
- `ship db:rollback`
- `ship db:seed`

Generation:

- `ship templ generate [--path <dir>] [--file <file.templ>]`
- `ship make:resource <name> [--path app] [--auth public|auth] [--views templ|none] [--domain <name>] [--wire] [--dry-run]` (or `ship make` for help)
- `ship make:model <Name> [fields...] [--force]`
- `ship make:factory <Name>`
- `ship make:locale <code>`
- `ship make:controller <Name|NameController> [--actions index,show,create,update,destroy] [--auth public|auth] [--domain <name>] [--wire]`
- `ship make:schedule <Name> --cron "<expr>"`
- `ship make:command <Name>`
- `ship make:scaffold <Name> [fields...] [--path app] [--views templ|none] [--auth public|auth] [--api] [--migrate] [--dry-run] [--force]`
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
4. `ship doctor --json` emits `{"ok","issues":[{"type","file","detail","severity"}]}` for agent tooling.
5. `ship routes --json` emits a JSON array of route objects (`method`, `path`, `auth`, `handler`, `file`).
6. Only `ship upgrade` may intentionally bump pinned versions.

## Implementation Mapping (Current Repo)

These commands are implemented as wrappers over existing workflows:

- `ship dev` -> auto mode:
  - jobs adapter `asynq` => full mode (`ship dev --all`)
  - other jobs adapters => `air -c .air.toml` (web with live reload + templ pre-generation)
- in interactive terminals, prints the local URL and prompts to open it in a browser (`[Y/n]`, Enter = yes); browser launch waits until the URL is reachable
- `ship dev --worker` -> `go run ./cmd/worker`
- `ship dev --all` -> starts web (`air -c .air.toml`) and worker concurrently with prefixed logs (`[web]`, `[worker]`) and signal-aware shutdown
- `ship run:command <name> [-- <args...>]` -> `go run ./cmd/cli/main.go <name> <args...>`
- `ship check` -> `go test ./...` (compile + unit checks, no integration-tagged tests)
- `ship test` -> `go test ./...` (integration-tagged tests are excluded by default)
- `ship test --integration` -> `go test -tags=integration ./...`
- `ship infra:up` -> detects `docker-compose`/`docker compose` and runs `up -d cache`, then attempts `up -d mailpit` (non-fatal if mailpit fails)
- `ship infra:down` -> detects `docker-compose`/`docker compose` and runs `down`
- `ship db:create` -> validates that target database URL is reachable (`goose status`)
- `ship db:generate [--config <path>] [--dry-run]` -> runs Bob generation via `bobgen-sql -c <config>` (default: core `db/bobgen.yaml`, then enabled module configs in deterministic sorted order from `config/modules.yaml`)
- `ship db:migrate` -> `goose up` for core migrations, then enabled module migrations in deterministic sorted order
- `ship db:status` -> `goose status` for core migrations, then enabled module migrations in deterministic sorted order; output is sectioned by scope (`== core migrations ==`, `== module <name> migrations ==`)
- `ship db:reset [--seed] [--force] [--yes] [--dry-run]` -> prints plan, runs `goose reset`, then `goose up`; optional seed
- `ship db:drop [--force] [--yes] [--dry-run]` -> prints plan, runs `goose reset` (reverts all applied migrations; does not physically drop the database)
- `ship db:make <migration_name>` -> `goose create <migration_name> sql`
- `ship db:make <migration_name> --soft-delete --table <table>` -> writes a migration that adds `deleted_at DATETIME` plus `idx_<table>_deleted_at`
- `ship db:rollback [amount]` -> `goose down` (default) or `goose down-to <amount>`
- Goose is managed by `ship`: it uses `goose` from `PATH` when present, otherwise falls back to `go run github.com/pressly/goose/v3/cmd/goose@v3.26.0`.
- `ship db:seed` -> `go run ./cmd/seed/main.go`

DB URL resolution precedence for db commands:

1. `DATABASE_URL`
2. `.env` / shell `PAGODA_DATABASE_*` variables via `config.GetConfig()`

If `PAGODA_DATABASE_URL` is set, CLI fails with an explicit error and asks to use `DATABASE_URL`.
If config resolves to embedded DB mode, DB commands fail with an explicit error.
- `ship db:reset`/`ship db:drop` refuse non-local DB URLs unless `--force` is provided.
- In `APP_ENV=production|prod`, `ship db:reset`/`ship db:drop` require both `--force` and `--yes`.
- Supported DB schemes for the Goose flow are currently limited to: `postgres`, `mysql`, and `sqlite`/`sqlite3`.
- Embedded SQLite connections are opened with lock-safety defaults: `journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout=5000`, `foreign_keys=ON`, `cache_size=-64000`, `temp_store=MEMORY`, and a single pooled connection (`SetMaxOpenConns(1)`).
- For `db:migrate`, `db:status`, `db:reset`, and `db:drop`, `ship` runs core migrations first, then enabled module migrations in deterministic sorted module order from `config/modules.yaml`.

Safety matrix:

| Command | Local DB | Non-local DB | Production |
|---|---|---|---|
| `db:reset` | requires `--yes` (or `--dry-run`) | requires `--force` + `--yes` (or `--dry-run`) | requires `--force` + `--yes` |
| `db:drop` | requires `--yes` (or `--dry-run`) | requires `--force` + `--yes` (or `--dry-run`) | requires `--force` + `--yes` |
| `db:create` | safe; supports `--dry-run` | safe; supports `--dry-run` | safe; supports `--dry-run` |
- `ship templ generate --path app` -> `templ generate -path app`, then move each `*_templ.go` into sibling `gen/` directory
- `ship new <app>` -> create minimal deterministic project scaffold in a new directory from CLI-embedded starter templates (no network calls)
  - interactive terminals prompt for i18n starter enablement unless `--i18n`/`--no-i18n` is provided
  - when enabled, starter locale files are scaffolded at `locales/en.yaml` and `locales/fr.yaml`
  - when disabled, CLI prints a follow-up hint that i18n can be enabled/migrated later with `ship i18n:*` + doctor-driven loops
- `ship agent:setup` -> generate per-agent allowlist artifacts from `tools/agent-policy/allowed-commands.yaml`
- `ship agent:check` -> fail if generated artifacts drift from canonical allowlist (for pre-commit/CI parity)
- `ship agent:status` -> show best-effort local Codex/Claude/Gemini install status vs repo policy
- `ship agent:start --task ... [--id ...]` -> create `.worktrees/<id>` on branch `agent/<id>`, generate `TASK.md` with task text + `ship describe` JSON + discovered `CLAUDE.md` context
- `ship agent:finish --id ... --message ... [--pr]` -> run `ship verify` in worktree, `git add -A`, validate conventional commit message, commit, optional `git push -u origin agent/<id>` + `gh pr create`, then `git worktree remove`
- `ship doctor --json` -> machine-readable doctor result on stdout; exit code 0 when there are no errors, 1 when any error is reported
- `ship config:validate` -> prints the known `PAGODA_*` config variables with type/default metadata and fails when any required variable is missing
- `ship config:validate --json` -> prints the same contract as JSON for agent tooling
- `ship routes` -> prints a route inventory table from `app/router.go` AST parsing (`METHOD PATH AUTH HANDLER`)
- `ship routes --json` -> prints the same route inventory as a JSON array
- `ship api:spec` -> parses `app/contracts/*.go` `// Route:` contracts and prints OpenAPI 3.0 JSON to stdout
- `ship api:spec --out <path>` -> writes generated OpenAPI JSON to the given file path
- `ship api:spec --serve` -> serves Swagger UI + generated spec at `http://127.0.0.1:<port>/api/docs` until interrupted
- `ship i18n:missing` -> compares `locales/en.yaml` keys with other locale files and lists missing/empty translations
- `ship i18n:unused` -> lists locale keys not referenced by `I18n.T(...)`/`i18n.T(...)` calls in `.go`/`.templ` sources
- `ship make:resource <name>` -> scaffold handler (+ optional templ page), ensure route-name constant, and print route snippet for manual insertion in `app/router.go`
- `ship make:resource <name> --domain <name>` -> generate domain-aware constructor slot (`domainService any`) and route wiring using `nil` placeholder
- `ship make:resource <name> --wire` -> also insert snippet behind ship markers in `app/router.go`
- `ship make:resource <name> --dry-run` -> preview all planned changes without writing files
- `ship make:model <Name>` -> scaffold a model query file at `db/queries/<model>.sql` with Bob-friendly named-query placeholders
- `ship make:model <Name> [fields...]` -> include typed field comments in the query scaffold and print next DB steps (`db:make`, `db:migrate`, `db:generate`)
- `ship make:factory <Name>` -> scaffold `tests/factories/<name>_factory.go` with a typed `Record` struct + `factory.New(...)` baseline
- `ship make:locale <code>` -> scaffold `locales/<code>.yaml` from `locales/en.yaml` with matching keys and empty values
- `ship make:controller <Name>` -> generate controller/handler scaffold in `app/web/controllers`
- `ship make:controller <Name> --domain <name>` -> generate domain-aware constructor slot (`domainService any`) and route wiring using `nil` placeholder
- `ship make:controller <Name> --actions ... --wire` -> wire generated routes into `app/router.go` markers
- `ship make:schedule <Name> --cron "<expr>"` -> insert a named cron entry into `app/schedules/schedules.go` between `ship:schedules` markers
- `ship make:command <Name>` -> scaffold `app/commands/<name>.go` and register it in `cmd/cli/main.go` at `ship:commands` markers
- `ship make:scaffold <Name> ...` -> orchestration command that composes `make:model`, `db:make`, `make:controller --domain <plural_model> --wire`, and optionally `make:resource --domain <plural_model>` / `db:migrate`
- `ship make:module <Name>` -> generate isolated module scaffold in `modules/<name>` with its own `go.mod`, module-facing types/contracts, and service tests
- `ship upgrade --to <version>` -> upgrades the pinned Goose CLI go-run fallback version (`gooseGoRunRef` in `tools/cli/ship/internal/cli/cli.go`)
- `ship upgrade --dry-run` -> prints planned pin change without writing files
- current scope: Goose pin only (expandable later)

Doctor checks (current):

- validates canonical app/layout directories under `app`
- validates required files (router, container, routenames, core docs)
- flags forbidden legacy paths from pre-refactor layout
- validates router marker pairs used by `--wire` generators
- validates router marker ordering (`start` before `end`) for `public` and `auth` sections
- validates package naming conventions in `web/ui` and `web/viewmodels`
- warns when exported templ functions in `app/views/` or enabled module `views/` directories are missing a `// Renders:` comment
- warns when component templates in `app/views/web/components/` or enabled module component dirs miss/mismatch `data-component="<kebab-name>"` on the root element (excluding `*_layout.templ`)
- flags unexpected root build artifacts (`web`, `worker`, `seed`, `ship`, `ship-mcp`)
- validates `.gitignore` includes root binary artifact ignore entries
- enforces a line budget for non-generated human-authored `.go` files (target <= 500 lines)
- validates CLI reference docs include core command tokens (`ship new`, `ship doctor`, `ship make:*`, `ship db:migrate`, `ship test --integration`)
- validates required config env vars declared in `config.Config`
- validates agent allowlist artifacts are in sync with `tools/agent-policy/allowed-commands.yaml`
- validates enabled modules in `config/modules.yaml` include `db/migrate/migrations` and `db/bobgen.yaml`
- validates cross-boundary import rules (controller `QueryProfile()` ban, jobs SQL coupling ban, notifications pubsub framework-core coupling ban, module source isolation ban for direct `github.com/leomorpho/goship/*` imports except explicit allowlist paths)
- warns when `/api/` routes appear to render HTML directly instead of using the standard JSON API helpers
- warns when SQL queries in `db/queries/` reference soft-delete tables without an explicit `deleted_at` filter (`DX028`)

Field syntax for `make:model`:

- `name:type` (for example: `title:string`, `published_at:time`, `is_live:bool`)
- supported types: `string`, `text`, `int`, `bool`, `time`, `float`, `email`, `url`
- use `--force` to overwrite an existing schema file

`ship new` v1 contract:

1. Creates a local scaffold only (no external downloads or package installs).
2. Writes deterministic starter files:
`go.mod`
`config/modules.yaml` (workspace-level module enablement)
`app/router.go` (with route marker pairs for `--wire`)
`app/web/routenames/routenames.go`
`app/views/templates.go`
`app/foundation/container.go`
`app/*` (domain skeletons)
`app/web/{controllers,middleware,ui,viewmodels}`
`app/jobs/jobs.go`
`db/{migrate/migrations,queries,gen,bobgen.yaml}`
`docs/00-index.md` and baseline architecture docs
3. Supports `--dry-run` and `--force`.
4. Supports `--i18n` and `--no-i18n` (otherwise prompts in interactive terminals).
5. If i18n is enabled during scaffold, writes starter locale files:
`locales/en.yaml`
`locales/fr.yaml`

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
3. Creates `app/web/controllers/<resource>.go`.
4. Creates `app/views/web/pages/<resource>.templ` when `--views templ`.
5. Ensures `RouteName<Resource>` constant exists in `app/web/routenames/routenames.go`.
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

- actual runtime behavior in `cmd/*`, `app/*`, `framework/*`, and `config/*`.

Rule:

- keep business/runtime logic out of CLI package; CLI should call stable commands/APIs.

## Deferred (Not In V1)

- `ship console`
- advanced generator variants
- `ship make:flag` (feature-flag seed/helper command)
