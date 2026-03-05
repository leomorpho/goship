# Development Workflows
<!-- FRONTEND_SYNC: Landing capability explorer in app/views/web/pages/landing_page.templ links here for Database and Migrations and Testing. Keep both landing copy and this doc aligned. -->

## Local Startup

Primary commands:

- `make dev`: infra + web process (recommended default)
- `make dev-worker`: infra + worker process
- `make dev-full`: infra + web + worker + JS/CSS watchers
- `go run ./tools/cli/ship/cmd/ship dev`: CLI equivalent of `make dev`

Legacy aliases still exist (`make init`, `make watch`) but they are no longer the preferred path.

`dev-full` process group runs:

- `watch-js`
- `watch-go`
- `watch-css`
- `watch-go-worker`

## Services and Infra

Docker Compose currently provisions:

- Redis (`goship_cache`)
- Mailpit (`goship_mailpit`)

Notes:

- Postgres service is currently not started by default.
- Runtime can operate with embedded DB mode; external DB remains supported by config.

## Assets

JS build:

- `npm --prefix frontend run build` (via `frontend/build.mjs`)
- Bundles Svelte entrypoints and vanilla JS

CSS build:

- Tailwind CLI to `app/static/styles_bundle.css`

Templ generation:

- `make templ-gen`
- or `go run ./tools/cli/ship/cmd/ship templ generate --path app`
- Generated `*_templ.go` files are moved to `gen/` subdirectories beside each templ package.

## Database and Schema

Entity schema source:

- `db/schema/*.go`

Common workflow:

1. `make ent-new name=YourEntity` (if new entity)
2. `go run ./tools/cli/ship/cmd/ship db:make your_change`
3. `make ent-gen` (or `go run ./tools/cli/ship/cmd/ship make:model ...` when scaffolding a new model)
4. `go run ./tools/cli/ship/cmd/ship db:migrate`
5. `go run ./tools/cli/ship/cmd/ship db:status`
6. Optional reset loop: `go run ./tools/cli/ship/cmd/ship db:reset --yes` (use `--dry-run` first)

Use `ship db:*` as the canonical migration interface; avoid calling migration tools directly.

## Worker and Tasks

Run worker manually:

- `make worker`
- Worker process currently targets Asynq backend only; ensure `adapters.jobs` is set to `asynq`.

Asynq UI:

- `make workerui`

Task processor registration:

- `cmd/worker/main.go`

## Testing

Go tests:

- `make check-compile` (compile app/packages + route tests without execution)
- `bash tools/scripts/test-unit.sh` (Docker-free unit package set)
- `make test` (broader suite; may include Docker-backed packages depending on environment)
- `go run ./tools/cli/ship/cmd/ship test`
- `make cover`

E2E tests:

- `make e2e`
- `make e2eui`

Note: current e2e specs are partially stale and should be treated as non-authoritative for GoShip behavior.

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
