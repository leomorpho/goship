# Extension Zones

This document defines where GoShip consumers are expected to customize freely and which seams are protected framework contracts that must stay stable for generators, doctor checks, and downstream adopters.

## Extension Zones

Framework repo ownership:

- `framework/` for reusable runtime contracts and framework-owned implementations
- `modules/` for installable capability packages that expose stable module-owned contracts
- `frontend/`, `styles/`, and `static/` for framework-owned UI assets that can evolve without changing protected runtime seams
- `docs/guides/` and operational docs that explain framework workflows and product behavior

Generated-app ownership:

- `app/` for app-specific controllers, views, UI assets, and product behavior outside protected seams
- `docs/` for app-local operating notes and architecture docs
- `styles/` and `static/` for app-owned presentation assets

Rule:

- changes inside an extension zone may diverge freely as long as they continue to satisfy any protected contract they touch.

## Protected Contract Zones

Framework repo seams:

- `app/container.go`
- `app/router.go`
- `app/schedules.go`

Generated-app seams:

- `app/router.go`
- `app/foundation/container.go`
- `config/modules.yaml`
- `docs/reference/01-cli.md`
- `tools/agent-policy/allowed-commands.yaml`

Protected-zone expectations:

- `app/router.go` must keep canonical route registration seams and generator markers intact in both the framework repo and generated apps
- `app/container.go` must keep the framework repo container seam stable for framework-owned runtime composition
- `app/schedules.go` must keep canonical schedule registration seams intact for the framework repo
- `app/foundation/container.go` must keep generated-app container markers and framework/module wiring seams intact
- `modules/admin/` should only couple to the framework repo through the canonical app router/container seams and the admin route surface
- `config/modules.yaml` remains the canonical module enablement manifest
- `docs/reference/01-cli.md` remains the living CLI contract for agents and operators
- `tools/agent-policy/allowed-commands.yaml` remains the source of truth for generated agent allowlist artifacts

## Enforcement

`ship doctor` validates this extension-zone manifest so protected seams stay documented in one place:

- the manifest must keep the `Extension Zones` and `Protected Contract Zones` sections
- the manifest must continue to list the canonical protected contract files above
- CLI docs should describe this check as part of repo-shape enforcement
