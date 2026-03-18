# Extension Zones

This document defines where GoShip consumers are expected to customize freely and which seams are protected framework contracts that must stay stable for generators, doctor checks, and downstream adopters.

## Extension Zones

These areas are intended for product-specific divergence:

- `app/` for app behavior, routes, handlers, jobs, pages, and composition choices that belong to the first-party app
- `framework/` for reusable infrastructure evolution, so long as changes preserve documented protected seams and stable interfaces
- `modules/` for installable capability packages that expose stable module-owned contracts
- `modules/admin/` for admin panel resources, routes, and baseline UI contracts that stay behind the canonical `/auth/admin` surface
- `frontend/` and `app/views/` for UI implementation details that can vary without changing framework-owned contracts
- `docs/guides/` and app-facing operational docs that explain local workflows and product behavior

Rule:

- changes inside an extension zone may diverge freely as long as they continue to satisfy any protected contract they touch.

## Protected Contract Zones

These seams are protected because generators, policy checks, or downstream adopters rely on them:

- `app/router.go`
- `app/foundation/container.go`
- `config/modules.yaml`
- `docs/reference/01-cli.md`
- `tools/agent-policy/allowed-commands.yaml`

Protected-zone expectations:

- `app/router.go` must keep canonical route registration seams and generator markers intact
- `app/foundation/container.go` must keep canonical container markers and framework/module wiring seams intact
- `modules/admin/` should only couple to the app through the canonical router/container seams and the admin route surface
- `config/modules.yaml` remains the canonical module enablement manifest
- `docs/reference/01-cli.md` remains the living CLI contract for agents and operators
- `tools/agent-policy/allowed-commands.yaml` remains the source of truth for generated agent allowlist artifacts

## Enforcement

`ship doctor` validates this extension-zone manifest so protected seams stay documented in one place:

- the manifest must keep the `Extension Zones` and `Protected Contract Zones` sections
- the manifest must continue to list the canonical protected contract files above
- CLI docs should describe this check as part of repo-shape enforcement
