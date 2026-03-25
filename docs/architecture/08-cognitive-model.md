<!-- FRONTEND_SYNC: Keep this cognitive model aligned with framework runtime seams and ship doctor checks. -->
# Cognitive Model

GoShip is optimized for recall, not discovery.

The framework has one canonical flow per concern. If a change does not fit these flows, refactor to fit the model instead of introducing parallel patterns.

## Runtime Seam Rule (Always)

`container.go -> router.go -> schedules.go`

- `container.go` is the framework-first composition seam.
- `router.go` is the framework-first HTTP route + middleware seam.
- `schedules.go` is the framework-first recurring-work seam.

## Request Flow (Always)

`router -> controller -> service/module -> viewmodel -> templ`

- Route registration is declared in `router.go`.
- HTTP handlers live in `framework/web/controllers` and enabled module route packages.
- Domain/business logic lives in framework and module service packages (`framework/*`, `modules/*`).
- Template payload shapes live in `framework/web/viewmodels`.
- Rendering helpers live in `framework/web/ui`.
- Templ sources live in `framework/web/*` and `framework/views/*`.

## Async Flow (Always)

`schedules/jobs -> service/module -> adapters`

- Schedule registration lives in `schedules.go`.
- Jobs delegate business behavior to framework/module services.
- Infrastructure dependencies are provided through the container adapters.

## Boot Flow (Always)

`container -> adapters -> modules -> router`

- Composition root is `container.go`.
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

- One framework-first runtime seam set at repo root: `container.go`, `router.go`, `schedules.go`.
- No deleted app-shell paths in canonical runtime guidance.
- Optional capabilities belong in `modules/*` and wire through explicit seam contracts.
- Route/schedule/container markers in root seam files are preserved for deterministic generator wiring.
