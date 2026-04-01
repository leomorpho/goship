<!-- FRONTEND_SYNC: Keep this cognitive model aligned with framework runtime seams and ship doctor checks. -->
# Cognitive Model

GoShip is optimized for recall, not discovery.

The framework has one canonical flow per concern. If a change does not fit these flows, refactor to fit the model instead of introducing parallel patterns.

## Runtime Seam Rule (Always)

`app/container.go` -> `app/router.go` -> `app/schedules.go`

- `app/container.go` is the framework-first composition seam.
- `app/router.go` is the framework-first HTTP route + middleware seam.
- `app/schedules.go` is the framework-first recurring-work seam.

## Request Flow (Always)

`router -> controller -> service/module -> viewmodel -> templ`

- Route registration is declared in `router.go`.
- HTTP handlers live in `framework/http/controllers` and enabled module route packages.
- Domain/business logic lives in framework and module service packages (`framework/*`, `modules/*`).
- Template payload shapes live in `framework/http/viewmodels`.
- Rendering helpers live in `framework/http/ui`.
- Templ sources live in `framework/http/*` and `framework/views/*`.

## Async Flow (Always)

`schedules/jobs -> service/module -> adapters`

- Schedule registration lives in `schedules.go`.
- Jobs delegate business behavior to framework/module services.
- Infrastructure dependencies are provided through the container adapters.

## Boot Flow (Always)

`container -> adapters -> modules -> router`

- Composition root is `app/container.go`.
- Runtime wiring is explicit and deterministic.
- Installable module construction is injected into router/container seams instead of ad-hoc package globals.

## Tooling Rule

Use `ship` as the single mutation interface for scaffolding and diagnostics.

- `ship make:model`
- `ship make:controller`
- `ship make:resource`
- `ship make:scaffold`
- `ship doctor`

## Non-Negotiable Conventions

- One framework-first runtime seam set under `app/`: `app/container.go`, `app/router.go`, `app/schedules.go`.
- No deleted app-shell paths in canonical runtime guidance.
- Optional capabilities belong in `modules/*` and wire through explicit seam contracts.
- Route/schedule/container markers in those `app/` seam files are preserved for deterministic generator wiring.
