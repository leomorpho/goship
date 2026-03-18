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

## Signed Managed Hook Contract (v1)

When managed mode is enabled, GoShip exposes a narrow signed HTTP bridge for control-plane actions:

- `GET /managed/status`
- `POST /managed/backup`
- `POST /managed/restore`

All managed hook requests must include:

- `X-GoShip-Timestamp` (unix seconds)
- `X-GoShip-Nonce` (single-use nonce)
- `X-GoShip-Signature` (hex HMAC-SHA256)

Canonical signature payload:

- `METHOD + "\n" + PATH_WITH_QUERY + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + RAW_BODY`

Verification/runtime rules:

- shared secret: `PAGODA_MANAGED_HOOKS_SECRET`
- max clock skew: `PAGODA_MANAGED_HOOKS_MAX_SKEW_SECONDS` (default `300`)
- replay nonce TTL: `PAGODA_MANAGED_HOOKS_NONCE_TTL_SECONDS` (default `300`)
- replay protection rejects reuse of the same nonce+timestamp tuple inside the TTL window
- replay protection should run through a shared/distributed replay store contract in multi-replica deployments; process-local memory is only the default fallback implementation

Response contract:

- `401` for missing/invalid signature material
- `409` for replayed nonce+timestamp
- `503` when the managed hook secret is not configured
- `404` when managed mode is disabled
- `POST /managed/backup` returns a typed `backup-manifest-v1` payload with SQLite source metadata,
  SHA-256 artifact checksum, and storage target metadata
- `POST /managed/restore` returns `restore_evidence` with the accepted manifest version, artifact
  checksum, database descriptor, and named post-restore validation checks

Backup manifest v1 invariants:

- version must be exactly `backup-manifest-v1`
- current schema is SQLite-first: `database.mode=embedded`, `database.driver=sqlite`
- `artifact.checksum_sha256` must be a 64-character hex SHA-256 digest
- `database.schema_version`, `database.source_path`, and storage target metadata are required

## SQLite-First Promotion Contract (v1)

GoShip defines a first promotion contract from SQLite to Postgres that stays framework-native and control-plane-agnostic.

Runtime metadata required for reporting:

- `database.mode`: `embedded` or `standalone`
- `database.driver`: normalized driver (`sqlite` or `postgres`)
- `database.migration_tracking_table`: currently `goship_schema_migrations`
- `database.migration_dialect`: current migration dialect (`sqlite` or `postgres`)
- `database.migration_portability`: SQL portability profile (`sql-core-v1`)
- `database.compatible_targets`: target drivers supported from current mode (SQLite source supports `postgres`)
- `database.promotion_path`: current supported workflow (`sqlite-to-postgres-manual-v1` for SQLite source)

First supported workflow (`sqlite-to-postgres-manual-v1`):

1. Freeze writes for the source app.
2. Record runtime metadata and migration baseline.
3. Export data from SQLite through framework-supported export hooks.
4. Provision Postgres target and run canonical migrations.
5. Import exported data into Postgres through framework-supported import hooks.
6. Run framework verification checks for row counts, migration baseline, and key integrity.
7. Switch config to Postgres and unfreeze writes.

This is intentionally manual-first and deterministic before introducing control-plane automation.

SQL portability constraints (`sql-core-v1`):

- Migrations must keep SQLite and Postgres compatibility as the default path.
- Engine-specific SQL is allowed only behind explicit dialect branches.
- Modules should avoid assumptions that rely on SQLite-only behavior (for example implicit rowid dependence).
- Backfills and data migrations should be idempotent and restart-safe for offline export/import workflows.

Minimum framework hook surface to support promotion:

- Runtime metadata reporting hook.
- Export hook with deterministic schema/version manifest output.
- Import hook with manifest validation and idempotent apply behavior.
- Post-import verification hook.

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
