# Module Surface Reset

This file freezes the old mixed module tree as legacy reference and defines the new first-party battery direction.

Canonical machine-readable source: `config/module-surface.yaml`.

## Canonical Battery Contract

A first-party installable battery must satisfy all of the following:

- one public module entrypoint in `modules/<id>/module.go`
- standalone Go module at `modules/<id>/go.mod`
- explicit install contract used by `ship module:add` (`routes`, `config`, `assets`, `jobs`, `templates`, `migrations`, `tests`)
- no app-domain ownership leakage into framework core seams
- no historical marker/TODO stubs in runtime wiring

`framework/` is core seam ownership only. `modules/` is installable battery ownership only.

## Decision Matrix

| Candidate | Class | Decision | Direction |
| --- | --- | --- | --- |
| `2fa` | `starter-app` | `eject` | move to starter-app auth hardening preset; remove from first-party installable battery surface |
| `admin` | `starter-app` | `eject` | move to starter-app/operator UI preset; remove from first-party installable battery surface |
| `ai` | `battery` | `rewrite` | keep capability but rebuild as standalone battery contract |
| `auditlog` | `battery` | `rewrite` | keep capability but rebuild as standalone battery contract |
| `auth` | `starter-app` | `eject` | move auth flows out of framework battery surface |
| `authsupport` | `core` | `rewrite` | move core auth seams under `framework/` ownership |
| `emailsubscriptions` | `battery` | `keep` | canonical standalone battery |
| `flags` | `battery` | `rewrite` | keep capability but rebuild as standalone battery contract |
| `i18n` | `battery` | `rewrite` | keep capability but rebuild as standalone battery contract |
| `jobs` | `battery` | `keep` | canonical standalone battery |
| `notifications` | `battery` | `keep` | keep as temporary bridge battery while split batteries land |
| `paidsubscriptions` | `battery` | `keep` | canonical standalone battery |
| `profile` | `starter-app` | `eject` | remove app-domain profile flows from framework battery surface |
| `pwa` | `battery` | `rewrite` | keep capability but rebuild as standalone battery contract |
| `storage` | `battery` | `keep` | canonical standalone battery |

Additional catalog-only candidates:

- `realtime`: `battery` + `rewrite` (promote to explicit standalone battery contract or fold into core seam contract)
- `billing` alias: `delete` + `eject` (stop treating naming alias as a first-party battery candidate)

## Legacy Quarantine Boundary

The mixed historical tree under `modules/` is frozen as quarry material only.

- old in-tree packages are reference input, not architecture truth
- new first-party installable capability work must follow the canonical battery contract
- keep/rewrite/eject decisions in this file are the policy source for strict verify checks

## Notifications Replacement Plan

`modules/notifications` is no longer the long-term architecture target. It is an interim bridge.

Target split batteries:

- `notifications-inbox`: notification center/read state surfaces
- `notifications-push`: browser/mobile push delivery + permission surfaces
- `notifications-email`: email notification delivery contracts
- `notifications-sms`: SMS notification delivery contracts
- `notifications-schedule`: planned/scheduled notification orchestration

Split rule:

- no new cross-cutting behavior lands in the current notifications monolith unless required to keep existing behavior functional
- new behavior lands directly in one of the target split batteries
