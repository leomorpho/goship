# Development Workflows

## Local Startup

Primary commands (from `Makefile`):

- `make init`: reset containers, build assets, seed data, start watch mode
- `make watch`: start process group via Overmind (`Procfile`)

`Procfile` runs:

- `watch-js`
- `watch-go`
- `watch-css`
- `watch-go-worker`

## Services and Infra

Docker Compose currently provisions:

- Redis (`goship_cache`)
- Mailpit (`goship_mailpit`)

Notes:

- Postgres service is present but commented out in `docker-compose.yml`.
- Default config DB mode is embedded SQLite.

## Assets

JS build:

- `npm run build` (via `build.mjs`)
- Bundles Svelte entrypoints and vanilla JS

CSS build:

- Tailwind CLI to `static/styles_bundle.css`

## Database and Schema

Entity schema source:

- `ent/schema/*.go`

Common workflow:

1. `make ent-new name=YourEntity` (if new entity)
2. `make makemigrations name=your_change`
3. `make ent-gen`
4. `make migrate`

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
- `make cover`

E2E tests:

- `make e2e`
- `make e2eui`

Note: current e2e specs are partially stale and should be treated as non-authoritative for GoShip behavior.

