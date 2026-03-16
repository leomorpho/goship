# Development Workflows
<!-- FRONTEND_SYNC: Landing capability explorer in app/views/web/pages/landing_page.templ links here for Database and Migrations and Testing. Keep both landing copy and this doc aligned. -->

## Configuration

Copy `.env.example` to `.env` and fill in the values your environment needs. All configuration is managed via environment variables. The application does not use YAML files for secrets or environment-specific overrides; the `.env` file is the single source of truth for local development.

## Local Startup

Primary commands:

- `make dev`: starts auto development mode (single-binary default; full multiprocess when jobs adapter is `asynq`)
- `make run`: single-binary web process with SQLite + Otter + Backlite
- `go run ./tools/cli/ship/cmd/ship dev`: CLI equivalent of `make dev`

Recommended modes:

- Unified dev mode:
  `make dev` (or `ship dev`)
  Runs in auto mode by convention: web-only for single-binary adapters, full mode when jobs adapter is `asynq`.
  Web mode runs via `air -c .air.toml` so Go rebuilds and templ generation happen automatically on edits.
  In interactive terminals, `ship dev` prints the local URL and prompts to open it in your browser (`[Y/n]`, Enter = yes). Browser open is deferred until the URL is reachable.
- Full multiprocess mode:
  `make dev-full` (or `ship dev --all`)
  Starts the web server (via `air`), worker, Vite (js), and Tailwind CSS watchers in a single multiplexed stream. Templ generation runs in `air` pre-build commands via `ship templ generate --path app` and `ship templ generate --path modules`. Requires `overmind` or `goreman`.
- Single-binary mode:
  `cp .env.example .env && make run`
  Uses embedded SQLite, in-memory Otter cache, and Backlite jobs. No Docker required.

Legacy aliases still exist (`make init`, `make watch`) but they are no longer the preferred path.

Email template previews (development only):

- Visit `/dev/mail` to see all browser preview links.
- Direct routes: `/dev/mail/welcome`, `/dev/mail/password-reset`, `/dev/mail/verify-email`.

`dev-full` process group runs:

- `watch-js`
- `watch-go`
- `watch-css`
- `watch-go-worker`

## Single Binary Mode

For the fastest development experience with zero dependencies:

1. Set the following in your `.env`:
   - `PAGODA_DB_DRIVER=sqlite`
   - `PAGODA_CACHE_DRIVER=otter`
   - `PAGODA_JOBS_DRIVER=backlite`
2. Run `make run`.

This mode uses an embedded SQLite database, the Otter in-memory cache, and the Backlite in-process job queue. No Docker is required.

## Services and Infra

Docker Compose currently provisions:

- Redis (`goship_cache`)
- Mailpit (`goship_mailpit`)

Notes:

- **Postgres and Redis are optional.** The runtime can operate entirely with embedded SQLite, in-memory Otter cache, and Backlite jobs.
- External database and cache services remain fully supported by configuration.
- `make run` does not start Docker Compose or any accessory services; use `make dev` if you need the full infrastructure stack.

## Assets

JS build:

- `npm --prefix frontend run build` (via Vite)
- Builds `vanilla_bundle.js`, `islands-runtime.js`, and per-island chunks

CSS build:

- Tailwind CLI to `app/static/styles_bundle.css`

Templ generation:

- `make templ-gen`
- or `go run ./tools/cli/ship/cmd/ship templ generate --path app`
- Generated `*_templ.go` files are moved to `gen/` subdirectories beside each templ package.

## Database and Migrations

Canonical runtime:

- migrations: Goose
- query generation: Bob (`bobgen-sql`)
- command surface: `ship db:*`

Current nuance:

- `db/queries/*.sql` is the canonical source of SQL.
- `db/gen/` is still hybrid during the Bob transition: some query families have maintained wrappers there, while other callers use `dbqueries.Get(...)` directly.
- The pre-commit Bob drift check currently enforces sync only for query files that have a same-name wrapper sibling in `db/gen/`.

Common workflow:

1. Create migration: `go run ./tools/cli/ship/cmd/ship db:make add_feature_x`
2. Apply migration(s): `go run ./tools/cli/ship/cmd/ship db:migrate`
3. Generate DB query code: `go run ./tools/cli/ship/cmd/ship db:generate`
4. Check status: `go run ./tools/cli/ship/cmd/ship db:status`
5. Optional local reset loop: `go run ./tools/cli/ship/cmd/ship db:reset --yes` (use `--dry-run` first)

Module behavior:

- `db:migrate` runs core first, then enabled modules from `config/modules.yaml` in deterministic sorted order.
- `db:generate` runs core first, then enabled modules in deterministic sorted order.

Safety:

- `db:drop` and `db:reset` require confirmation (`--yes`), and non-local URLs additionally require `--force`.
- production targets require both `--force` and `--yes`.
- supported DB URL schemes are limited to `postgres`, `mysql`, `sqlite`, and `sqlite3`.

Use `ship db:*` as the canonical interface; avoid invoking Goose/Bob directly.

## Worker and Tasks

Run worker manually:

- `make worker`
- Worker process currently targets Asynq backend only; ensure `adapters.jobs` is set to `asynq`.

Single-binary mode:

- `make run` starts the web process only.
- Backlite jobs run in-process with the web server.
- No separate worker is required.

Asynq UI:

- `make workerui`

Task processor registration:

- `cmd/worker/main.go`

## App Commands

App-scoped CLI commands live under `app/commands` and are executed via `cmd/cli/main.go`.

Run commands through `ship`:

- `go run ./tools/cli/ship/cmd/ship run:command <name>`
- passthrough args: `go run ./tools/cli/ship/cmd/ship run:command send:test-email -- --to you@example.com --dry-run`

Generate a new command scaffold:

- `go run ./tools/cli/ship/cmd/ship make:command BackfillUserStats`

The generator writes `app/commands/<name>.go` and wires registration in `cmd/cli/main.go` between
`// ship:commands:start` and `// ship:commands:end`.

## Testing

Go tests:

- `make check-compile` (compile app/packages + route tests without execution)
- `bash tools/scripts/test-unit.sh` (Docker-free unit package set)
- `make test` (broader suite; may include Docker-backed packages depending on environment)
- `go run ./tools/cli/ship/cmd/ship test`
- `make cover`
- `bash tools/scripts/precommit-tests.sh` (full stateless gate used before commit/CI)

E2E tests:

- `make e2e-smoke` (single happy-path smoke; Playwright starts `go run ./cmd/web` automatically via `webServer`)
- `make e2e`
- `make e2eui`

CI uses the smoke spec only (`tests/e2e/tests/smoke.spec.ts`) to validate startup and basic app serving.
Note: broader legacy e2e specs are partially stale and should be treated as non-authoritative for GoShip behavior.

## Agent Command Policy

Canonical allowlist:

- `tools/agent-policy/allowed-commands.yaml`

Generated artifacts (for local tool import):

- `tools/agent-policy/generated/codex-prefixes.txt`
- `tools/agent-policy/generated/claude-prefixes.txt`
- `tools/agent-policy/generated/gemini-prefixes.txt`
- `tools/agent-policy/generated/allowed-prefixes.json`

Commands:

- `go run ./tools/cli/ship/cmd/ship agent:setup`
- `go run ./tools/cli/ship/cmd/ship agent:check`

Guardrails:

- `agent:check` runs in pre-commit and CI.
- `ship doctor` also validates these artifacts are in sync.

## Documentation Artifacts

LLM reference bundle:

- regenerate `LLM.txt` from `README.md` + `docs/**/*.md` with:
  - `make llm-txt`
  - or `bash tools/scripts/generate-llm-txt.sh`

Automation:

- pre-commit hook runs `tools/scripts/precommit-generate-llm-txt.sh` and stages updated `LLM.txt` automatically.
