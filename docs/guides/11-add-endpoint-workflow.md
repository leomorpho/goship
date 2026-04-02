# Add Endpoint Workflow

Canonical contributor workflow for adding a new endpoint to a GoShip app.

## Goal

Add a route, handler, route-name constant, and optional templ view using the canonical generator and router flow.

## Use This Workflow When

- You need a new page or handler under the app router.
- You want `ship` to scaffold the controller/resource baseline.
- You need route wiring that stays aligned with doctor and route inventory checks.

## Preferred Path

For a page/resource-style endpoint:

```bash
go run ./tools/cli/ship/cmd/ship make:resource contact_form --path app --auth public --views templ --wire
```

For a controller with explicit actions:

```bash
go run ./tools/cli/ship/cmd/ship make:controller Contact --actions index,create --auth public --wire
```

Scope note:
- `make:resource` is the starter-safe generated-app path today.
- `make:controller` is now starter-safe too, but on starter apps it uses the generated CRUD/runtime route backend instead of creating Echo controller files.

## What This Should Change

- `app/web/controllers/*`
- optional templ files under `app/views/**/*`
- route-name constants under `app/web/routenames/routenames.go`
- route wiring in `app/router.go`

## Verification

```bash
go run ./tools/cli/ship/cmd/ship routes
go run ./tools/cli/ship/cmd/ship doctor
go test ./app ./tools/cli/ship/internal/generators -count=1
```

## Common Failure Modes

1. Missing `ship:routes:*` markers in `app/router.go`: restore canonical marker blocks before wiring.
2. Route-name drift: confirm `app/web/routenames/routenames.go` has the generated constant and the route uses it.
3. Wrong placement: app-owned HTTP code belongs under `app/web/controllers`, not `framework/*`.

## Related References

- `docs/reference/01-cli.md`
- `docs/architecture/02-structure-and-boundaries.md`
- `docs/architecture/04-http-routes.md`
- `docs/guides/08-building-an-api.md`
