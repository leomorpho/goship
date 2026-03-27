# Deprecated Framework Surface

This directory is a holding area for framework code scheduled for deletion.

## Current status

- Legacy `framework/web` has been removed.
- Active runtime imports target `framework/http`.
- `framework/http` is the canonical HTTP kernel surface:
  - request/auth context
  - middleware stack
  - page/controller primitives
  - baseline health/error controllers
  - route wiring

## Why this split

This follows the same separation pattern used by Rails/Laravel/Django:

- Keep transport/runtime kernel in the framework (`framework/http`).
- Move app-specific presentation and feature pages into modules.
- Keep generated/template stubs minimal in the kernel.

## Guardrail

- Do not recreate `framework/web` or any `framework/deprecated/web` compatibility path.
- New framework HTTP code belongs in `framework/http/...`.
