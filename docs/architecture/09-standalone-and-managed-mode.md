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
- managed settings should also surface drift detection and rollback semantics so operators can see when the runtime has diverged from the intended override state
- the managed-key registry itself is a versioned artifact shared with the control plane, so runtime and orchestration tooling can agree on an authoritative schema mapping
- runtime metadata publishes that registry contract through `config.RuntimeMetadata().Managed` using `managed-key-registry-v1` and `managed-key-schema-v1`
- runtime report surfaces now expose per-module adoption metadata, so external control planes can
  track module identity, module path, version, and source without parsing repo internals.
- runtime report surfaces also expose a divergence classification contract (`divergence-classification-v1`)
  plus escalation policy (`divergence-escalation-v1`) so control-plane tooling can distinguish
  ordinary extension-zone drift from protected-contract recovery work and repeated divergence that
  should be reviewed for upstreaming.

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
- `X-GoShip-Key-Version` (active/previous key version label for rollout-safe key rotation)

Canonical signature payload:

- `METHOD + "\n" + PATH_WITH_QUERY + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + RAW_BODY`

shared signature vectors and the canonical payload library now live on the shared security seam so
both the runtime bridge and control-plane tooling can reuse the same signing fixtures.

Verification/runtime rules:

- shared secret: `PAGODA_MANAGED_HOOKS_SECRET`
- max clock skew: `PAGODA_MANAGED_HOOKS_MAX_SKEW_SECONDS` (default `300`)
- replay nonce TTL: `PAGODA_MANAGED_HOOKS_NONCE_TTL_SECONDS` (default `300`)
- durable replay store path: `PAGODA_MANAGED_HOOKS_NONCE_STORE_PATH` (default temp-file path)
- replay protection rejects reuse of the same nonce+timestamp tuple inside the TTL window
- replay protection uses a durable local store by default and still exposes a shared/distributed replay store contract for multi-replica deployments
- the same signed-request pattern is used for the control-plane cron entrypoint contract so scheduler-driven actions reuse the replay/timestamp verification model
- security event reporting emits explicit managed-hook events for invalid signature and replay failures so runtime telemetry can classify auth failures without parsing response bodies

Managed hook key rotation:

- `PAGODA_MANAGED_HOOKS_SECRET` is the active signing key.
- `PAGODA_MANAGED_HOOKS_PREVIOUS_SECRET` may be supplied during a rotation window so in-flight managed-hook callers can continue to verify successfully.
- during rotation windows, both old/new signatures can validate when paired with their key-version headers, so rollout does not require managed-hook downtime.
- The previous secret is read-only compatibility material and must not become the long-term signing key for new requests.
- The active and previous secrets share the same replay/timestamp contract, so key rotation does not bypass nonce or skew checks.

Response contract:

- `401` for missing/invalid signature material
- `409` for replayed nonce+timestamp
- `503` when the managed hook secret is not configured
- `404` when managed mode is disabled
- `POST /managed/backup` returns a typed `backup-manifest-v1` payload with SQLite source metadata,
  SHA-256 artifact checksum, and storage target metadata
- `POST /managed/restore` returns `restore_evidence` with the accepted manifest version field
  (`accepted_manifest_version`), artifact checksum, database descriptor, and named post-restore
  validation checks, plus optional `record_links` identifiers (`incident_id`, `recovery_id`,
  `deploy_id`) that mirror the control-plane audit context supplied with the request. The canonical
  checks are `manifest.validated`,
  `artifact.checksum.sha256`, and `database.schema_version.present`.

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

Canonical state machine (`promotion-state-machine-v1`):

- `sqlite-source-ready`: safe source state; `ship db:promote` may apply the canonical config mutation.
- `config-mutated-awaiting-import`: partial state; config is already flipped and the runtime must run import/verification follow-up instead of re-running promotion.
- `import-complete-awaiting-verify`: partial state; verification evidence is still required before treating promotion as complete.
- `inconsistent-runtime-state`: unsafe state; runtime metadata does not match either the canonical SQLite source or a recognized partial state.

Blocking rule:

- `ship db:promote` must reject invocations from `config-mutated-awaiting-import`,
  `import-complete-awaiting-verify`, and `inconsistent-runtime-state`.

SQL portability constraints (`sql-core-v1`):

- Migrations must keep SQLite and Postgres compatibility as the default path.
- Engine-specific SQL is allowed only behind explicit dialect branches.
- Modules should avoid assumptions that rely on SQLite-only behavior (for example implicit rowid dependence).
- Backfills and data migrations should be idempotent and restart-safe for offline export/import workflows.

Minimum framework hook surface to support promotion:

- Runtime metadata reporting hook.
- The runtime metadata report is expected to be versioned and to carry a metadata handshake envelope for orchestration preflight.
- Export hook with deterministic schema/version manifest output.
- Import hook with manifest validation and idempotent apply behavior.
- Post-import verification hook.

## Staged Rollout And Canary Decision Contract (v1)

GoShip defines one canonical decision payload for staged rollout and canary evaluation:
`staged-rollout-decision-v1`.

This payload is owned jointly by the runtime/control-plane seam:

- `ship runtime:report --json` supplies runtime facts such as active profile, adapters, DB metadata,
  current framework version, per-module adoption metadata, managed-key sources, and handshake/version identifiers.
- control-plane policy supplies rollout intent such as cohort rules, target percentage, promotion
  guardrails, and rollback thresholds.
- the decision payload composes those two inputs into one externalized verdict so downstream
  tooling does not need a second runtime-specific format.

Canonical payload fields:

- `schema_version`: must equal `staged-rollout-decision-v1`
- `runtime_contract_version`: version token echoed from the runtime report contract consumed for the decision
- `policy_input_version`: version token for the control-plane policy bundle that produced the decision
- `generated_at`: RFC3339 timestamp for the decision artifact
- `target`: object naming the deployment unit under evaluation (`app`, `environment`, optional `module`)
- `readiness`: summarized runtime readiness facts copied from the runtime report without renaming their meaning
- `policy`: summarized control-plane policy inputs copied without restating runtime facts
- `decision`: one of `hold`, `canary`, `promote`, or `rollback`
- `canary`: object describing the selected canary cohort, percentage, and exit criteria when `decision=canary`
- `blockers`: ordered list of machine-readable reasons preventing promotion
- `verification`: minimum evidence that downstream tooling must check before acting on the decision

Versioning rules:

- A new `schema_version` is required when field meaning changes, required fields change, or enum
  semantics change.
- Additive optional fields may ship within `staged-rollout-decision-v1` as long as existing field
  meaning stays stable.
- `runtime_contract_version` and `policy_input_version` must be preserved verbatim so audit tools
  can trace which runtime facts and control-plane policy produced the decision.
- deploy and upgrade entrypoints must reject unsupported contract identifiers before orchestration
  proceeds; the current allowlist is `runtime-contract-v1` for runtime reports and
  `upgrade-readiness-v1` for upgrade readiness.

Minimum verification semantics:

- verify `schema_version == staged-rollout-decision-v1`
- verify the referenced `runtime_contract_version` is the exact runtime report version consumed for the decision
- verify the runtime report indicated readiness for the evaluated action instead of re-deriving that state from scratch
- verify the referenced `policy_input_version` matches the policy bundle approved by the external authority
- verify `blockers` is empty before acting on `promote`
- verify `canary` is present and complete before acting on `canary`

Composition rule:

- runtime facts answer what the app can safely do right now
- control-plane policy answers what the operator wants to do
- `staged-rollout-decision-v1` answers what action is authorized after those two inputs are checked

Out of scope for v1:

- a built-in rollout engine inside GoShip
- traffic shaping or request-splitting infrastructure
- vendor-specific deployment-controller adapters

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
