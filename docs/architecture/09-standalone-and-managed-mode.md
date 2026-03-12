# Standalone and Managed Mode

This document defines the boundary between GoShip as an open source framework and any external managed-service control plane built on top of it.

## Core Principle

GoShip owns capability.

An external control plane owns authority.

That split keeps GoShip usable as standalone software while still allowing a hosted platform to operate many GoShip-derived apps safely.

## What Must Stay In GoShip

Anything that a single app can use on its own belongs in GoShip.

Examples:

- config schema
- adapter interfaces
- backup and restore capability
- admin/settings surfaces for standalone apps
- runtime metadata reporting
- database mode support (`sqlite`, later `postgres`)
- S3-compatible storage provider support

If a self-hosted GoShip app would reasonably need the feature, it should not be control-plane-only.

## What Must Stay Outside GoShip

Anything that only exists because many apps are being operated together belongs outside the runtime.

Examples:

- per-customer deployment provider selection
- region selection across many customers
- fleet scheduling and alerting
- customer intake workflow
- pricing and support policy
- cross-app audit and operator workflow
- multi-app profitability tracking

These are managed-service concerns, not framework runtime concerns.

## Two Supported Operating Contexts

### Self-Managed

The app is run directly by its owner.

Characteristics:

- local settings are editable in the app
- backup and restore actions are initiated from the app layer
- storage provider choice belongs to the app operator
- no external control plane is required

### Externally Managed

The app still runs as GoShip, but an external authority manages some settings and operations.

Characteristics:

- the app keeps the same underlying capability
- some settings may become read-only locally
- secrets and managed overrides are injected from outside the repo
- restore and lifecycle actions may be initiated from the control plane

Important:

- managed mode must not remove the standalone capability from GoShip
- it only changes who is allowed to control certain knobs

## Config Resolution Model

GoShip should resolve runtime config from layered sources:

1. framework defaults
2. app repo config
3. environment variables / secret injection
4. managed overrides, if managed mode is enabled

Rules:

- GoShip defines the schema
- secrets stay out of the repo
- managed overrides are allowlisted and inspectable
- the runtime should be able to report both the effective value and the source of that value

The external control plane should not rely on mutating arbitrary config files in application repos as its normal operating mechanism.

## Backup Boundary

Backups are a good example of the split.

GoShip should own:

- backup driver interfaces
- restore driver interfaces
- provider plugin support
- backup metadata format
- standalone backup UI/commands

The control plane should own:

- schedules
- managed storage credentials
- restore initiation for managed apps
- fleet policy and audit

The framework provides the engine. The control plane decides how it is used across many apps.

## Deployment Boundary

GoShip should not embed provider-specific deployment orchestration for Railway, Fly, Render, or other hosts.

Those concerns belong in:

- external control-plane adapters
- separate CLI tooling
- optional libraries outside the core app runtime

GoShip can expose runtime metadata and health/readiness contracts that make external deployment tooling easier, but the runtime should not be coupled to one hosted platform.

## Admin and Settings Boundary

GoShip should have a real standalone admin/settings experience.

Managed mode may:

- hide or lock certain settings
- show that a value is managed externally
- restrict who can trigger backup or restore actions

But managed mode should not require a different app architecture. It is an authority layer over the same runtime capability.

## Docket Tracking

The current follow-up work for this boundary is tracked in:

- `.docket/tickets/TKT-110.md`
- `.docket/tickets/TKT-111.md`
- `.docket/tickets/TKT-112.md`
- `.docket/tickets/TKT-113.md`
- `.docket/tickets/TKT-114.md`
