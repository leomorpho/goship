# Add Module Workflow

Canonical contributor workflow for adding or installing a module in GoShip.

## Goal

Add a module with deterministic wiring and keep module boundaries isolated from app-core code.

## Use This Workflow When

- You are creating a new installable module scaffold.
- You are enabling module-owned DB/runtime files.
- You need to validate that module boundaries remain isolated.

## Preferred Path

Preview a new module scaffold:

```bash
go run ./tools/cli/ship/cmd/ship make:module Billing --path modules --dry-run
```

Generate the scaffold:

```bash
go run ./tools/cli/ship/cmd/ship make:module Billing --path modules
```

If the workflow is about enabling an already-existing module, follow the living CLI contract for:

```bash
go run ./tools/cli/ship/cmd/ship module:add Billing
```

## What This Should Change

- `modules/<name>/*`
- optional module DB assets under `modules/<name>/db/*`
- module docs or contracts colocated with the module
- app/runtime wiring only through approved module seams

## Verification

```bash
go run ./tools/cli/ship/cmd/ship doctor
bash tools/scripts/check-module-isolation.sh
go test ./modules/... -count=1
```

If the module owns DB assets:

```bash
go run ./tools/cli/ship/cmd/ship db:migrate
go run ./tools/cli/ship/cmd/ship db:generate
```

## Common Failure Modes

1. Importing app packages from a module: keep module code isolated from `app/*`.
2. Missing module DB layout: add `db/migrate/migrations`, `db/queries`, and `db/bobgen.yaml` under the module.
3. Runtime wiring in the wrong place: compose modules from the canonical app/runtime seam, not ad hoc package globals.

## Related References

- `docs/reference/01-cli.md`
- `docs/architecture/02-structure-and-boundaries.md`
- `docs/architecture/07-core-interfaces.md`
- `docs/guides/05-jobs-module.md`
