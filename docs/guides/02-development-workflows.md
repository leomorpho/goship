# Development Workflows

## Local Startup

Primary commands:

- `make dev`: infra + web process (recommended default)
- `make dev-worker`: infra + worker process
- `make dev-full`: infra + web + worker + JS/CSS watchers
- `go run ./cli/ship/cmd/ship dev`: CLI equivalent of `make dev`

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

- `npm run build` (via `build.mjs`)
- Bundles Svelte entrypoints and vanilla JS

CSS build:

- Tailwind CLI to `static/styles_bundle.css`

Templ generation:

- `make templ-gen`
- or `go run ./cli/ship/cmd/ship templ generate --path app`
- Generated `*_templ.go` files are moved to `gen/` subdirectories beside each templ package.

## Database and Schema

Entity schema source:

- `ent/schema/*.go`

Common workflow:

1. `make ent-new name=YourEntity` (if new entity)
2. `make makemigrations name=your_change`
3. `make ent-gen`
4. `make migrate`

CLI wrapper equivalents (in progress) are documented in `docs/reference/01-cli.md`.

## Worker and Tasks

Run worker manually:

- `make worker`

Asynq UI:

- `make workerui`

Task processor registration:

- `cmd/worker/main.go`

## Testing

Go tests:

- `make test`
- `go run ./cli/ship/cmd/ship test`
- `make cover`

E2E tests:

- `make e2e`
- `make e2eui`

Note: current e2e specs are partially stale and should be treated as non-authoritative for GoShip behavior.
