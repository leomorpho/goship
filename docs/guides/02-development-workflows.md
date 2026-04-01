# Development Workflows
<!-- FRONTEND_SYNC: Landing capability explorer in app/views/web/pages/landing_page.templ links here for Database and Migrations and Testing. Keep both landing copy and this doc aligned. -->

## Configuration

Copy `.env.example` to `.env` and fill in the values your environment needs. All configuration is managed via environment variables. The application does not use YAML files for secrets or environment-specific overrides; the `.env` file is the single source of truth for local development.

## Local Startup

Primary commands:

- `ship dev`: canonical app-on dev loop (single-node web loop; distributed full loop)
- `make dev`: compatibility wrapper around `ship dev`
- `make run`: single-binary web process with SQLite + Otter + Backlite
- `make ship-install`: install the latest local `ship` binary to `GOBIN` (or `GOPATH/bin`)
- `go run ./tools/cli/ship/cmd/ship dev`: direct module-path invocation of the same canonical command

Recommended modes:

- Unified dev mode:
  `ship dev` (or `make dev` wrapper)
  Runs the canonical app-on loop: web mode for `single-node` and full multiprocess mode for `distributed`.
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
- `make run` does not start Docker Compose or any accessory services; use `ship dev` (or the `make dev` wrapper) if you need the full infrastructure stack.
- CI keeps dedicated boundary suites for `make test-module-isolation` and `make test-sql-portability`; run those targets directly when changing installable-module imports or SQL portability metadata/contracts.
- `make test-sql-portability` now checks `sql-core-v1` runtime metadata plus the branch annotations and placeholder conventions in the canonical migration/query SQL sources, so portability drift fails in a named lane instead of hiding in broad test output.
- `make test-module-isolation` reports module/file context for forbidden imports and fails when the allowlist contains stale entries; trim `tools/scripts/test/module-isolation-allowlist.txt` as soon as a temporary exception is removed.

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
- `ship make:command` currently targets the framework workspace and rejects the minimal starter scaffold.

The generator writes `app/commands/<name>.go` and wires registration in `cmd/cli/main.go` between
`// ship:commands:start` and `// ship:commands:end`.

## Testing

Go tests:

- `make check-compile` (compile app/packages + route tests without execution)
- `bash tools/scripts/test-unit.sh` (Docker-free unit package set)
- `make test` (broader suite; may include Docker-backed packages depending on environment)
- `make test-generator-contracts` (generator report snapshot + idempotency gate used by CI)
- `make test-generator-idempotency` (standalone generator duplicate-run matrix without snapshot refresh)
- `make test-alpha-contracts` (legacy frozen `v0.1.0-alpha` root-help + route-inventory gate; not the canonical v1 release-proof lane)
- `make test-doc-sync` (route/scope documentation guard used by CI)
- `make test-agent-evals` (cold-start agent eval gate with JSON score report at `artifacts/agent-eval-report.json`)
- `make test-dead-routes` (route inventory regression guard used by CI)
- `go run ./tools/cli/ship/cmd/ship test`
- `make cover`
- `bash tools/scripts/precommit-tests.sh` (full stateless gate used before commit/CI)

Test data factories:

- `framework/factory` provides generic test builders with `Build` and DB-backed `Create`.
- `tests/factories/user_factory.go` is the canonical example for user records + traits.
- Scaffold a new factory with: `go run ./tools/cli/ship/cmd/ship make:factory <Name>`.
- `ship make:factory` currently targets the framework workspace and rejects the minimal starter scaffold.
- Typical usage in tests:
  - `user := factories.User.Create(t, db)`
  - `admin := factories.User.Create(t, db, factories.WithAdminRole)`

HTTP integration test helpers:

- `framework/testutil` provides `NewTestServer(t)` for app-level request tests with a real container + in-memory DB.
- `PostForm` automatically fetches and submits CSRF tokens for form routes.
- `PostJSON` sends JSON request bodies for API routes without transport boilerplate.
- `PostMultipart` builds multipart form uploads (fields + files) for upload-oriented routes.
- `AsUser(userID)` signs an auth session cookie so tests can hit authenticated routes without manually building session cookies.
- Response assertions support fluent checks:
  - `AssertStatus(code)`
  - `AssertRedirectsTo(path)`
  - `AssertContains(text)`
  - `AssertJSON(&target)`
  - `AssertSSEEvent(event, data)`
- Example:
  - `s := testutil.NewTestServer(t)`
  - `s.PostForm("/user/login", form).AssertRedirectsTo("/welcome/preferences")`
  - `s.Get("/auth/logout", s.AsUser(userID)).AssertRedirectsTo("/")`

E2E tests:

- `npm --prefix tests/e2e run test:golden` (canonical GoShip golden browser contract: boot, landing/register/login entrypoints, anonymous protected-route redirects, and `/demo/islands` runtime mounts)
- `make e2e-smoke` (single happy-path smoke; Playwright starts `go run ./cmd/web` automatically via `webServer`)
- `make e2e-admin-smoke` (admin auth and managed-surface smoke lane)
- `make e2e`
- `make e2eui`

CI uses the smoke spec only (`tests/e2e/tests/smoke.spec.ts`) to validate startup and basic app serving.
These browser suites currently target the **framework repo app surface** (`/user/*` login/register plus `/auth/*` protected routes),
not the minimal starter/API-only generated-app auth surface (`/auth/login`, `/auth/register`).
The current GoShip golden-flow browser contract lives at `tests/e2e/tests/goship.spec.ts` and is the
authoritative suite for scaffolded public/auth/islands coverage.
The admin scaffold lane lives at `tests/e2e/tests/admin_scaffold.spec.ts` and covers the
critical admin auth/managed-settings/flags/trash surfaces as Playwright baseline smoke coverage.
UI-impacting changes should add or update Playwright coverage for the affected flow.
browser evidence should be attached or referenced in ticket or PR notes.
Primary artifact paths remain `tests/e2e/playwright-report` and `tests/e2e/test-results`.
The `verify_strict` CI job runs `ship verify --profile strict` and serves as the precondition for the downstream Cherie compatibility gate.
The `startup_smoke` CI job runs `go test ./tools/cli/ship/internal/commands -run TestFreshAppStartupSmoke -count=1` and enforces generated-app startup checks for migrations, web boot (`/health`, `/health/readiness`), and worker boot.
The `upgrade_readiness` CI job runs focused upgrade contract tests (`TestRunUpgrade_JSONReadinessReport_RedSpec` and `TestRunUpgrade_RejectsUnsupportedContractVersion_RedSpec`) so readiness-schema drift is caught in the default workflow.
The Cherie compatibility lane runs `tests/e2e/tests/cherie_compatibility.spec.ts` with a web-only process env (`PAGODA_PROCESSES_WEB=true`, `PAGODA_PROCESSES_WORKER=false`, `PAGODA_PROCESSES_SCHEDULER=false`, `PAGODA_PROCESSES_COLOCATED=false`) so the baseline only measures boot, auth, and realtime route compatibility.
Treat `verify_strict`, `startup_smoke`, and `cherie_compatibility_smoke` as the required status checks for Cherie-facing sync work.
Top-level CI gate lanes summarize the canonical status surface:
- `top_level_fresh_app` depends on `fresh_app_ci`.
- `top_level_upgrades` depends on `upgrade_readiness`.
- `top_level_batteries` depends on `module_isolation`, `module_matrix`, `sql_portability`, and `generator_contracts`.
The `module_matrix` CI job runs `make test-module-matrix` and directly executes the nested first-party module packages (`modules/jobs`, `modules/notifications`, and `modules/paidsubscriptions`) so battery repos are part of required CI instead of hiding behind root `go test ./...`.
The `split_frontend_contract` CI job runs `make test-sveltekit-contract`, which boots a fresh API-only app and proves the generated SvelteKit contract artifact matches the live `/api/v1/status` backend contract under the blessed same-origin assumptions.
- `top_level_frontend` depends on `split_frontend_contract`.
- `top_level_standalone_operations` depends on `bootstrap_budget`, `startup_smoke`, and `cleanroom_bob_verification`.
The `generator_contracts` CI job runs `make test-generator-contracts` and blocks merges on generator snapshot or idempotency drift.
Use `make test-generator-idempotency` when only the duplicate-run matrix is relevant and you do not need to touch snapshots.
When a generator output change is intentional, refresh the golden file locally with `UPDATE_GENERATOR_SNAPSHOTS=1 make test-generator-contracts` and commit the updated snapshot in the same change.
The `alpha_contract` CI job runs `make test-alpha-contracts` and freezes the historical `v0.1.0-alpha` root CLI help plus route inventory surface. Treat it as legacy compatibility evidence, not as the primary v1 release-proof lane.
Only refresh those snapshots when the alpha surface change has approved review before merge, then run `UPDATE_ALPHA_CONTRACTS=1 make test-alpha-contracts` and commit the snapshot update with the contract change.
The `doc_sync` CI job runs `make test-doc-sync` and keeps the HTTP route map plus project-scope docs aligned with canonical managed/admin/realtime surfaces.
The `agent_evals` CI job runs `make test-agent-evals`, enforces the cold-start eval success threshold, and uploads `artifacts/agent-eval-report.json` for regression triage.
The `dead_route_regression` CI job runs `make test-dead-routes` and keeps the canonical route inventory checks from silently regressing.
The `bootstrap_budget` CI job runs `make test-bootstrap-budget` and measures the canonical starter flow as `ship new <app> --no-i18n`, `ship db:migrate`, `go run ./cmd/web`, and HTTP checks against `/health/readiness` plus `/` inside the generated scaffold.
The default budget is 120 seconds via `BOOTSTRAP_BUDGET_SECONDS=120`; keep local reruns on comparable hardware or raise the variable only when investigating runner variance rather than changing the committed CI threshold.
Use `BOOTSTRAP_BUDGET_SECONDS` only as a local rerun override; the committed CI contract keeps the 120-second threshold.
The `fresh_app_ci` CI job runs `make test-fresh-app-ci`, which executes the real generated-app proof lane in one pass by running `TestFreshApp` plus `TestFreshAppStartupSmoke` and failing hard if either target returns `[no tests to run]` or `[no test files]`.
Use `make test-fresh-app-ci` locally when touching starter layout/runtime boot wiring to keep generation, batteries, verify, and smoke evidence coupled in one deterministic lane.
If the Cherie lane breaks:
1. Re-run the Playwright spec locally with `npm --prefix tests/e2e run test:cherie-smoke`.
2. Compare `/up`, `/user/login`, and `/auth/realtime` behavior against the **framework repo** baseline before widening scope.
3. Either land a framework fix or document the downstream breakage explicitly before merging.
The golden suite is intentionally narrow: visual regression coverage and a full optional-module browser
matrix remain out of scope for now.

## Internationalization

Canonical locale source files live in `locales/*.toml` (`en.toml` is the source of truth for new writes). Runtime/CLI still dual-read YAML during migration windows.

Common commands:

- `go run ./tools/cli/ship/cmd/ship make:locale fr` to scaffold `locales/fr.toml` with the same keys and empty values.
- `ship make:locale` remains starter-safe only when the app already has a locale baseline.
- `go run ./tools/cli/ship/cmd/ship i18n:migrate` to convert legacy `locales/*.yaml` catalogs to canonical TOML.
- `go run ./tools/cli/ship/cmd/ship i18n:normalize` to rewrite TOML catalogs into deterministic canonical ordering.
- `go run ./tools/cli/ship/cmd/ship i18n:missing` to list missing/empty translation keys per locale.
- `go run ./tools/cli/ship/cmd/ship i18n:unused` to list locale keys not referenced in `.go`/`.templ` `I18n.T(...)` calls.

## Agent Command Policy

Canonical allowlist:

- `tools/agent-policy/allowed-commands.yaml`

Generated artifacts (for local tool import):

- `tools/agent-policy/generated/agent-prefixes.txt`
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
