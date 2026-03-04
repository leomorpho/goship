# GoShip App Layout

This directory is the reference application that demonstrates how to use the GoShip framework in a Rails/Laravel-inspired structure.

## Top-Level Areas

- `router.go`: canonical app router entrypoint (single place to scan route registration).
- `foundation/`: app composition root and runtime adapter wiring.
- `web/`: HTTP-facing presentation layer.
- `app/`: app-specific domain logic.
- `db/`: schema and migrations.
- `views/`: Templ source files and generated view code.
- `jobs/`: background task processors.
- `static/`: public static assets served by the app.
- `styles/`: source stylesheets compiled into `static/`.
- `testdata/`: app-scoped fixture data.

## Web Layer

- `web/controllers/`: route handlers (Rails/Laravel-style controllers).
- `web/middleware/`: HTTP middleware.
- `web/routenames/`: named route constants.
- `web/ui/`: rendering/page/controller helpers used by handlers.
- `web/viewmodels/`: typed page/form payloads for templates.
- `web/capabilities/`: landing-page capability metadata.

## Conventions

- Keep route registration centralized in `router.go`.
- Keep handlers thin; move durable business logic to `app/`.
- Prefer table-driven tests in the same package as implementation.
- Keep generated files adjacent to source templates under `views/**/gen`.
