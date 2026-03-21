# Extension Zones

This document defines where GoShip consumers are expected to customize freely and which seams are protected framework contracts that must stay stable for generators, doctor checks, and downstream adopters.

## Extension Zones

These areas are intended for product-specific divergence:

- `framework/` for reusable infrastructure evolution, so long as changes preserve documented protected seams and stable interfaces
- `modules/` for installable capability packages that expose stable module-owned contracts
- `modules/admin/` for admin panel resources, routes, and baseline UI contracts that stay behind the canonical `/auth/admin` surface
- `frontend/`, `styles/`, and `static/` for framework-owned UI assets that can evolve without changing protected runtime seams
- `docs/guides/` and operational docs that explain local workflows and product behavior

Rule:

- changes inside an extension zone may diverge freely as long as they continue to satisfy any protected contract they touch.

## Protected Contract Zones

These seams are protected because generators, policy checks, or downstream adopters rely on them:

- `container.go`
- `router.go`
- `schedules.go`
- `config/modules.yaml`
- `docs/reference/01-cli.md`
- `tools/agent-policy/allowed-commands.yaml`

Protected-zone expectations:

- `router.go` must keep canonical route registration seams and generator markers intact
- `container.go` must keep canonical container markers and framework/module wiring seams intact
- `schedules.go` must keep canonical schedule registration seams intact
- `modules/admin/` should only couple to the framework repo through the canonical root router/container seams and the admin route surface
- `config/modules.yaml` remains the canonical module enablement manifest
- `docs/reference/01-cli.md` remains the living CLI contract for agents and operators
- `tools/agent-policy/allowed-commands.yaml` remains the source of truth for generated agent allowlist artifacts

## Enforcement

`ship doctor` validates this extension-zone manifest so protected seams stay documented in one place:

- the manifest must keep the `Extension Zones` and `Protected Contract Zones` sections
- the manifest must continue to list the canonical protected contract files above
- CLI docs should describe this check as part of repo-shape enforcement
