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
- `ship i18n:init [--force]`
- `ship i18n:scan [--format json] [--paths <path1,path2,...>] [--limit <n>]`
- `ship i18n:instrument [--apply] [--paths <path1,path2,...>] [--limit <n>]`
- `ship i18n:migrate [--force]`
- `ship i18n:normalize`
- `ship i18n:compile`
- `ship i18n:ci`
- `ship i18n:missing`
- `ship i18n:unused`
- `ship agent:setup`
- `ship agent:check`
- `ship agent:status`
- `ship agent:start --task "<description>" [--id TASK-001]`
- `ship agent:finish --id TASK-001 --message "feat(scope): summary" [--pr]`
- `ship upgrade --to <version> [--contract-version <schema>] [--dry-run] [--json]`
- `ship upgrade apply --to <version> [--contract-version <schema>] [--dry-run] [--json]`
- `ship upgrade` will surface an upgrade readiness report and blocker schema for orchestration preflight before future automation mutates pinned versions.

Local runtime:

- `ship dev` (auto default: web-only for single-binary adapters; full mode when jobs adapter is `asynq`)
- `ship dev --web`
- `ship dev --worker`
- `ship dev --all`
- `ship infra:up` (or `ship infra` for help)
- `ship infra:down`
- `ship run:command <name> [-- <args...>]`

Testing:

- `ship test` (unit default)
- `ship test --integration`

Database:

- `ship db:create [--dry-run]` (or `ship db` for help)
- `ship db:generate [--config <path>] [--dry-run]`
- `ship db:export [--json]`
- `ship db:import [--json]`
- `ship db:make <migration_name> [--soft-delete --table <table>]` (or `ship db` for help)
- `ship db:migrate`
- `ship db:status`
- `ship db:verify-import [--json]`
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
- `ship make:job <Name>`
- `ship make:mailer <Name>`
- `ship make:schedule <Name> --cron "<expr>"`
- `ship make:command <Name>`
- `ship make:scaffold <Name> [fields...] [--path app] [--views templ|none] [--auth public|auth] [--api] [--migrate] [--dry-run] [--force]`
- `ship make:module <Name> [--path modules] [--module-base github.com/leomorpho/goship-modules] [--dry-run] [--force]`
- `ship destroy resource:<name>`

Command grammar policy:

1. Canonical commands are namespaced via colon (`db:*`, `make:*`, `infra:*`).
2. Bare namespace commands (`ship db`, `ship make`, `ship infra`) show scoped help.
3. Space-form subcommands (for example `ship db migrate`) are intentionally rejected with a hint to use colon form.
4. Root help is intentionally concise and grouped by command area; each direct top-level command includes a one-line purpose description for human/LLM discoverability.
5. Scoped help (`ship db --help`, `ship make --help`, etc.) should also use one-line purpose descriptions per listed subcommand.

## Versioning Rules

1. CLI-managed tools (for example `templ`) must be pinned to project-declared versions.
2. `ship dev` and `ship test` must never auto-upgrade toolchain versions.
3. `ship doctor` reports version drift and prints explicit fix commands.
4. `ship doctor --json` emits `{"ok","issues":[{"type","file","detail","severity"}]}` for agent tooling.
5. `ship routes --json` emits a JSON array of route objects (`method`, `path`, `auth`, `handler`, `file`).
6. Installable standalone batteries use `v0.0.0` plus a local `replace` and `go.work use` entry during in-repo development.
7. Released consumers such as Cherie should consume tagged module versions instead of committing local-path replaces downstream.
8. Only `ship upgrade` may intentionally bump pinned versions.

## Implementation Mapping (Current Repo)

These commands are implemented as wrappers over existing workflows:

- `ship dev` -> canonical app-on loop:
  - runtime profile `single-node` or `server-db` => web mode (`air -c .air.toml`)
  - runtime profile `distributed` => full mode (`ship dev --all`)
- `ship dev` now runs a generated-app scaffold fast-path check before launching processes; when required scaffold paths/markers are broken it exits early with root-cause doctor diagnostics and a corrective `ship doctor --json` next step
- in interactive terminals, prints the local URL and prompts to open it in a browser (`[Y/n]`, Enter = yes); browser launch waits until the URL is reachable
- `ship profile:set <single-binary|standard|distributed>` -> rewrites the project `.env` runtime profile and process preset values so the canonical topology can be selected deterministically
- `ship dev --worker` -> `go run ./cmd/worker`
- `ship dev --all` -> starts web (`air -c .air.toml`) and worker concurrently with prefixed logs (`[web]`, `[worker]`) and signal-aware shutdown
- `ship run:command <name> [-- <args...>]` -> `go run ./cmd/cli/main.go <name> <args...>`
- `ship test` -> canonical fast quality loop: runs the curated unit package list from `scripts/test/unit-packages.txt`, then compile-only checks for `scripts/test/compile-packages.txt`; falls back to `go test ./...` when the package lists are absent
- `ship test --integration` -> `go test -tags=integration ./...`
- `ship profile:set <single-binary|standard|distributed>` -> rewrites the project `.env` runtime profile and process preset values so the canonical topology can be selected deterministically
- `ship adapter:set <db|cache|jobs|pubsub|storage|mailer>=<impl>...` -> rewrites canonical adapter env vars in the project `.env` and rejects invalid runtime selections before they drift into an unsupported plan
- `ship module:add <name>` -> updates `config/modules.yaml`, app marker snippets, root `go.mod` `require`/`replace` directives, and `go.work` `use` entries for standalone batteries with local `go.mod` files
- `ship module:add <name>` now prints an explicit install contract summary grouped by `routes`, `config`, `assets`, `jobs`, `templates`, `migrations`, and `tests` so install/remove impact is visible before review
- `ship module:remove <name>` -> removes those managed entries (including module-owned `.env.example` snippets) when safe; fails with exact blocker file paths when the repo still imports the module outside managed wiring points
- `ship verify` -> rejects standalone-battery drift when root `go.mod` dependencies on installable modules are not the canonical local-dev shape (`v0.0.0` + local `replace` + matching `go.work use`), enforces the canonical GoShip framework repo layout when run in this repo (including required root runtime seams `container.go`, `router.go`, and `schedules.go`, plus failure on a forbidden top-level `app/` shell), validates the extension-zone manifest for those protected seams, and enforces the canonical no-compatibility/no-deprecation wording invariant across the operator-facing docs set
- `ship verify` now runs a generated-app scaffold fast-path gate before `templ generate`/`go build`; when scaffold root-cause checks fail (`DX001`, `DX002`, `DX005`, `DX011`) verify stops immediately and points to `ship doctor --json`
- `ship verify` includes an orchestration contract-mismatch preflight step before deploy/upgrade/promote flows so unsupported runtime combinations fail before orchestration starts, and the preflight stays aligned with the managed-settings access contract used by the runtime report; `--runtime-contract-version` and `--upgrade-contract-version` reject unsupported contract identifiers before the preflight runs
- `ship infra:up` -> detects `docker-compose`/`docker compose` and runs `up -d cache`, then attempts `up -d mailpit` (non-fatal if mailpit fails)
- `ship infra:down` -> detects `docker-compose`/`docker compose` and runs `down`
- `make test-module-isolation` -> dedicated CI lane for installable-module root import isolation; reports offending module/file context and rejects stale allowlist entries
- `make test-sql-portability` -> dedicated CI lane for `sql-core-v1` runtime metadata plus migration/query portability; diagnostics call out missing branch handling and placeholder drift
- `make test-generator-contracts` -> dedicated CI lane for shared generator-report snapshot and idempotency-matrix drift
- `make test-generator-idempotency` -> focused local rerun for the duplicate-generation matrix without snapshot refresh
- `make test-alpha-contracts` -> dedicated CI lane for the frozen `v0.1.0-alpha` root-help and route-inventory contract
- `make test-bootstrap-budget` -> dedicated CI lane for the canonical starter bootstrap budget (`ship new` + starter `go run ./cmd/web`)
- `npm --prefix tests/e2e run test:golden` -> canonical Playwright golden-flow lane for current GoShip scaffold behavior (`/up`, landing/register/login entrypoints, anonymous auth redirects, `/demo/islands` runtime mount contract)
- `npm --prefix tests/e2e run test:cherie-smoke` -> dedicated CI lane for the Cherie boot/auth/realtime compatibility baseline
- `ship verify --profile strict` -> strict verify tier used as the precondition for the required Cherie sync gate
- `ship db:create` -> validates that target database URL is reachable (`goose status`)
- `ship db:generate [--config <path>] [--dry-run]` -> runs Bob generation via `bobgen-sql -c <config>` (default: core `db/bobgen.yaml`, then enabled module configs in deterministic sorted order from `config/modules.yaml`)
- `ship db:export [--json]` -> reports the SQLite export manifest checksum contract from current runtime metadata; `--json` emits a structured export report with the typed backup manifest payload, suggested next commands, and planning note for agents/tooling
- `ship db:import [--json]` -> reports the manual SQLite export/import plan from current runtime metadata and suggests the follow-up post-import verification command; `--json` emits machine-readable plan output for agents/tooling
- `ship db:promote [--dry-run] [--json]` -> builds the canonical SQLite-to-Postgres config mutation plan from current runtime metadata; default mode rewrites the local `.env` to the standard profile plus `db=postgres cache=redis jobs=asynq`, `--dry-run` previews the exact mutation set without writing files, and `--json` emits the same mutation payload plus `promotion-state-machine-v1` metadata for agents/tooling; export/import/verification steps remain manual follow-up commands
- `ship db:promote` rejects partial or unsafe states such as `config-mutated-awaiting-import`, so the CLI cannot re-run the config flip after the runtime has already crossed into a post-mutation state
- `ship db:migrate` -> `goose up` for core migrations, then enabled module migrations in deterministic sorted order
- `ship db:status` -> `goose status` for core migrations, then enabled module migrations in deterministic sorted order; output is sectioned by scope (`== core migrations ==`, `== module <name> migrations ==`)
- `ship db:verify-import [--json]` -> reports the post-import verification checks from current runtime metadata; `--json` emits machine-readable verification output for agents/tooling
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
  - when enabled, starter locale files are scaffolded at `locales/en.toml` and `locales/fr.toml`
  - validates scaffold template layout before rendering and returns explicit diagnostics when required starter files are missing
  - when disabled, CLI prints a follow-up hint that i18n can be enabled/migrated later with `ship i18n:*` + doctor-driven loops
- `ship agent:setup` -> generate per-agent allowlist artifacts from `tools/agent-policy/allowed-commands.yaml`
- `ship agent:check` -> fail if generated artifacts drift from canonical allowlist (for pre-commit/CI parity)
- `ship agent:status` -> show best-effort local Codex/Claude/Gemini install status vs repo policy
- `ship agent:start --task ... [--id ...]` -> create `.worktrees/<id>` on branch `agent/<id>`, generate `TASK.md` with task text + `ship describe` JSON + discovered `CLAUDE.md` context
- `ship agent:finish --id ... --message ... [--pr]` -> run `ship verify` in worktree, `git add -A`, validate conventional commit message, commit, optional `git push -u origin agent/<id>` + `gh pr create`, then `git worktree remove`
- `ship doctor --json` -> machine-readable doctor result on stdout; exit code 0 when there are no errors, 1 when any error is reported
- `ship runtime:report --json` -> machine-readable runtime capability report covering active profile, adapters, process plan, source-aware `process_topology` (including web/worker realtime roles when enabled), web features, DB runtime metadata, managed-key sources, per-module adoption metadata, and a versioned handshake envelope; downstream staged rollout and canary orchestration must compose those runtime facts plus the approved `policy_input_version` into `staged-rollout-decision-v1` instead of inventing a second runtime-specific decision payload
- `ship runtime:report --json` also emits the divergence-classification contract (`divergence-classification-v1` + `divergence-escalation-v1`) so downstream tooling can distinguish extension-zone drift, protected-contract drift, and repeated local divergence that should trigger upstream review or recovery action
- `ship describe --pretty` now includes a shared-infra adoption summary (`shared_modules`, `shared_module_ids`, `custom_app_controllers`, `custom_app_jobs`, `custom_app_commands`) as a non-blocking upstreaming metric
- `ship config:validate` -> prints the known `PAGODA_*` config variables with type/default metadata and fails when any required variable is missing
- `ship config:validate --json` -> prints the same contract as JSON for agent tooling
- `ship routes` -> prints a route inventory table from `app/router.go` AST parsing (`METHOD PATH AUTH HANDLER`)
- `ship routes --json` -> prints the same route inventory as a JSON array
- `ship api:spec` -> removed from the CLI surface; invocations return unknown command/help output
- `ship i18n:init [--force]` -> scaffolds `locales/en.toml` and `locales/fr.toml` for existing apps, preserving existing files unless `--force` is passed; prints deterministic next-step migration loop commands
- `ship i18n:scan [--format json] [--paths <path1,path2,...>] [--limit <n>]` -> scans Go/templ/islands sources (including `.js`, `.ts`, `.jsx`, `.tsx`, `.svelte`, `.vue` under `frontend/islands/`) for hardcoded user-facing literals and emits deterministic JSON diagnostics (`id`, `kind`, `severity`, `file`, `line`, `column`, `message`, `suggested_key`, `confidence`)
- `ship i18n:instrument [--apply] [--paths <path1,path2,...>] [--limit <n>]` -> builds a deterministic rewrite plan for high-confidence findings; default mode is dry-run JSON report, `--apply` rewrites safe Go controller `*.String(..., "literal")` response sites (for example `c.String`, `ctx.String`) to i18n calls and appends missing baseline keys to `locales/en.toml` (falls back to legacy YAML when present)
- `ship i18n:migrate [--force]` -> converts `locales/*.yaml`/`*.yml` catalogs to canonical `locales/*.toml` catalogs (preserves existing TOML unless `--force`)
- `ship i18n:normalize` -> rewrites TOML locale catalogs into deterministic canonical ordering for stable diffs/CI
- `ship i18n:compile` -> generates typed i18n key artifacts from baseline English catalogs for Go (`app/i18nkeys/keys_gen.go`) and islands TypeScript (`frontend/islands/i18n-keys.ts`)
- `ship i18n:ci` -> deterministic strict i18n CI profile command; enforces scanner findings + strict-mode doctor `DX029` findings and exits non-zero on violations
- `ship i18n:missing` -> compares baseline English locale keys with other locale files and lists missing/empty translations
- `ship i18n:unused` -> lists locale keys not referenced by `I18n.T(...)`/`i18n.T(...)` calls in `.go`/`.templ` sources
- Full i18n operator policy (coverage matrix, enforcement mapping, strict rollout, JSON contracts): `docs/guides/10-i18n-llm-migration-workflow.md`
- `ship make:resource <name>` -> scaffold handler (+ optional templ page), ensure route-name constant, and print route snippet for manual insertion in `app/router.go`
- `ship make:resource <name> --domain <name>` -> generate domain-aware constructor slot (`domainService any`) and route wiring using `nil` placeholder
- `ship make:resource <name> --wire` -> also insert snippet behind ship markers in `app/router.go`
- `ship make:resource <name> --dry-run` -> preview all planned changes without writing files
- `ship make:resource` path ownership is canonical: `--path` must resolve to `app`; values that escape or diverge from `app` fail fast
- `ship make:model <Name>` -> scaffold a model query file at `db/queries/<model>.sql` with Bob-friendly named-query placeholders
- `ship make:model <Name> [fields...]` -> include typed field comments in the query scaffold and print next DB steps (`db:make`, `db:migrate`, `db:generate`)
- `ship make:factory <Name>` -> scaffold `tests/factories/<name>_factory.go` with a typed `Record` struct + `factory.New(...)` baseline
- `ship make:locale <code>` -> scaffold `locales/<code>.toml` from `locales/en.toml` (or legacy `en.yaml`) with matching keys and empty values
- `ship make:controller <Name>` -> generate controller/handler scaffold in `app/web/controllers`
- `ship make:controller <Name> --domain <name>` -> generate domain-aware constructor slot (`domainService any`) and route wiring using `nil` placeholder
- `ship make:controller <Name> --actions ... --wire` -> wire generated routes into `app/router.go` markers
- `ship make:controller` path ownership is canonical: `--path` must resolve to `app`; values that escape or diverge from `app` fail fast
- `ship make:island <Name>` -> Generate a frontend island scaffold: the canonical pair `frontend/islands/<Name>.js` with an exported `mount(el, props)` seam and `app/views/web/components/<name>_island.templ` with the matching `data-island` / `data-props` mount target; follow-up remains explicit: run `ship templ generate --file app/views/web/components/<name>_island.templ`, run `make build-js`, then render `@components.<Name>Island(...)` from the page/component that should host the island
- `ship make:job <Name>` -> Generate a background job scaffold at `app/jobs/<name>.go` plus `app/jobs/<name>_test.go` around `core.Jobs` / `core.JobHandler` registration helpers
- `ship make:mailer <Name>` -> Generate a mailer scaffold at `app/views/emails/<name>.templ` and wire a `/dev/mail/<name>` preview into the existing mail preview controller and route surface
- `ship make:schedule <Name> --cron "<expr>"` -> insert a named cron entry into `schedules.go` between `ship:schedules` markers
- `ship make:command <Name>` -> scaffold `app/commands/<name>.go` and register it in `cmd/cli/main.go` at `ship:commands` markers
- `ship make:scaffold <Name> ...` -> orchestration command that composes `make:model`, `db:make`, `make:controller --domain <plural_model> --wire`, and optionally `make:resource --domain <plural_model>` / `db:migrate`
- `ship make:module <Name>` -> generate isolated module scaffold in `modules/<name>` with its own `go.mod`, module-facing types/contracts, and service tests
- generated `modules/<name>/module.go` now includes a battery install-contract scaffold (`Contract()`) with explicit sections for `routes`, `config`, `assets`, `jobs`, `templates`, and `migrations`
- `ship make:module <Name> --dry-run` now includes explain output for each emitted scaffold file (`- file: <path> -> owner: <battery-contract-owner>`) so generator ownership is explicit during review
- `ship make:module` path ownership is canonical: `--path` must resolve to `modules`; values that escape or diverge from `modules` fail fast
- `ship destroy resource:<name>` -> remove generator-managed resource scaffold targets in deterministic order (router marker block, route-name constant, templ page, optional scaffold test, controller); paths without recognized ownership signals are skipped with explicit reasons and the command exits non-zero when nothing is safely removable
- current first-class installable batteries include `notifications`, `paidsubscriptions`, `emailsubscriptions`, `realtime`, `pwa`, `jobs`, and `storage`; `realtime` captures the canonical runtime startup and route contract seams, `pwa` captures installable route/static asset/browser seams, and `storage` exposes the canonical `core.BlobStorage` seam as a standalone battery package under `modules/storage`
- `ship upgrade --to <version>` -> runs readiness preflight and prints the deterministic rewrite plan without writing files
- `ship upgrade apply --to <version>` -> applies the deterministic rewrite plan to the pinned Goose CLI go-run fallback version (`gooseGoRunRef` in `tools/cli/ship/internal/cli/cli.go`) and canonicalizes legacy Goose command paths when present
- upgrade apply writes include post-write verification and automatic rollback to the previous file content when verification fails in failure paths
- `ship upgrade --contract-version <schema>` / `ship upgrade apply --contract-version <schema>` -> require the supported upgrade readiness contract version
- `ship upgrade --dry-run` / `ship upgrade apply --dry-run` -> print planned pin changes without writing files
- `ship upgrade --json` -> emits the machine-readable `upgrade-readiness-v1` preflight contract (`schema_version`, `blocker_classification`, `target_version`, `ready`, `rollback_target`, `canary`, `verification`, `plan`, `result`, `blockers`, `manual_follow_ups`, `remediation_hints`, `planned_changes`) without writing files; `plan.safe_steps` computes deterministic minor-boundary bridge hops (when needed) and maps each hop to a concrete `ship upgrade apply --to <version>` command, `result` reports whether a pin change is planned, blocker entries carry `classification`, and `manual_follow_ups` plus `verification.command` remain concrete so weaker agents can trust the plan before mutating the pinned Goose reference
- when canonical generated conventions drift (for example missing `gooseGoRunRef` in stale scaffolds), `ship upgrade` returns a blocked readiness report (`result.outcome=blocked`, `blockers[0].id=upgrade.convention_drift`) instead of applying rewrites
- current scope: Goose pin only (expandable later)

Doctor checks (current):

- validates canonical app/layout directories under `app`
- validates required files (router, container, routenames, core docs)
- flags forbidden legacy paths from pre-refactor layout
- validates router marker pairs used by `--wire` generators
- validates router marker ordering (`start` before `end`) for `public` and `auth` sections
- validates package naming conventions in `web/ui` and `web/viewmodels`
- warns when exported templ functions in `app/views/` or enabled module `views/` directories are missing a `// Renders:` comment; `DX023` scans the contiguous pre-function comment block and tolerates `// Renders:`/`// Route(s):` ordering differences plus optional blank lines
- warns when component templates in `app/views/web/components/` or enabled module component dirs miss/mismatch `data-component="<kebab-name>"` on the root element (excluding `*_layout.templ`)
- flags unexpected root build artifacts (`web`, `worker`, `seed`, `ship`, `ship-mcp`)
- allows top-level `tmp/` as intentional local dev build output (`air` default in this repo)
- validates `.gitignore` includes root binary artifact ignore entries
- enforces a line budget for non-generated human-authored `.go` files (target <= 500 lines)
- validates CLI reference docs include core command tokens (`ship new`, `ship doctor`, `ship make:*`, `ship db:migrate`, `ship test --integration`)
- validates the extension-zone manifest keeps one canonical list of extension zones and protected contract seams (`DX031`)
- validates required config env vars declared in `config.Config`
- validates agent allowlist artifacts are in sync with `tools/agent-policy/allowed-commands.yaml`
- validates enabled modules in `config/modules.yaml` include `db/migrate/migrations` and `db/bobgen.yaml`
- validates cross-boundary import rules (controller `QueryProfile()` ban, jobs SQL coupling ban, notifications pubsub framework-core coupling ban, module source isolation ban for direct `github.com/leomorpho/goship/*` imports with no runtime allowlist escape hatch)
- treats unpaired generator markers (`DX005`) and raw controller form parsing (`DX027`) as blocking structural errors
- rejects canonical-doc transition/deprecation wording with file:line diagnostics and optional historical-reference allowlist support via `docs/policies/02-transition-wording-allowlist.txt` (`DX030`)
- warns when `/api/` routes appear to render HTML directly instead of using the standard JSON API helpers
- warns when SQL queries in `db/queries/` reference soft-delete tables without an explicit `deleted_at` filter (`DX028`)
- i18n strict-mode enforcement (`PAGODA_I18N_STRICT_MODE=off|warn|error`) for hardcoded literals in controllers/views/islands and plural/select locale completeness for `I18n.TC(...)`/`I18n.TS(...)`, with `.i18n-allowlist` support for intentional exceptions (stable `I18N-S-*` selectors preferred; legacy `path:line` still accepted) (`DX029`)

Managed hook replay contract:

- `framework/security.ManagedHookVerifier` exposes a pluggable `NonceStore` seam via `WithNonceStore(...)`
- default behavior remains process-local in-memory replay protection until app/runtime wiring provides a shared backend
- managed hook key rotation keeps `PAGODA_MANAGED_HOOKS_SECRET` as the active key and accepts `PAGODA_MANAGED_HOOKS_PREVIOUS_SECRET` during the rotation window
- the rotation window does not relax replay protection; nonce and timestamp validation still apply to both secrets

run-anywhere verification gate:

- `ship verify --profile fast|standard|strict` selects the verification tier (`standard` default; `strict` requires `nilaway`; `fast` skips nilaway and `go test`)
- `ship verify` includes a `standalone exportability gate` step
- `ship verify` includes a `hard-cut wording invariant` step that re-checks canonical docs for transition/deprecation drift after `ship doctor --json`
- the gate rejects control-plane dependency drift in standalone runtime/starter surfaces

Field syntax for `make:model`:

- `name:type` (for example: `title:string`, `published_at:time`, `is_live:bool`)
- supported types: `string`, `text`, `int`, `bool`, `time`, `float`, `email`, `url`
- use `--force` to overwrite an existing schema file

Generator output contract:

- `ship make:*` commands now emit shared `Created:`, `Updated:`, `Preview:`, and `Next:` sections so generator results stay deterministic for humans and agent tooling
- dry-run generator flows keep the same section layout and mark the report as `(dry-run)` instead of switching to a different prose format
- the generator contract lane snapshots the report shape for model, job, command, and dry-run resource flows so per-generator drift is explicit

`ship new` v1 contract:

1. Creates a local scaffold only (no external downloads or package installs).
2. Writes deterministic starter files:
`.env`
`go.mod`
`config/modules.yaml` (workspace-level module enablement)
`app/router.go` (with route marker pairs for `--wire`)
`app/web/routenames/routenames.go`
`app/views/templates.go`
`container.go`
`app/*` (domain skeletons)
`app/web/{controllers,middleware,ui,viewmodels}`
`app/jobs/jobs.go`
`app/views/web/pages/{landing,home_feed,profile}.templ`
`cmd/web/main.go`
`cmd/worker/main.go`
`db/migrate/migrations/`
`db/{queries,gen,bobgen.yaml}`
`styles/`
`static/`
`docs/00-index.md` and baseline architecture docs
`tools/agent-policy/allowed-commands.yaml`
`tools/agent-policy/generated/`
3. Supports `--dry-run` and `--force`.
4. Fresh scaffolds are expected to pass the canonical confidence loop: `ship db:migrate`, `go run ./cmd/web`, and `ship verify --profile fast`.
4. Supports `--i18n` and `--no-i18n` (otherwise prompts in interactive terminals).
5. If i18n is enabled during scaffold, writes starter locale files:
`locales/en.toml`
`locales/fr.toml`
6. Fails with explicit scaffold layout diagnostics if embedded starter template root/files are missing or unreadable.

Generated project workflow:

1. `ship` is the canonical interface for dev/test/verify/generate flows.
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

- `--views templ`: generates a controller/page-rendering handler (`ui.NewPage`, layout assignment, `RenderPage`).
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

- actual runtime behavior in `cmd/*`, `container.go`, `router.go`, `schedules.go`, `framework/*`, and `config/*`.

Rule:

- keep business/runtime logic out of CLI package; CLI should call stable commands/APIs.

## Deferred (Not In V1)

- `ship console`
- advanced generator variants
- `ship make:flag` (feature-flag seed/helper command)
