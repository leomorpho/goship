# SQLite To Postgres Promotion Runbook

This is the canonical manual-first runbook for `sqlite-to-postgres-manual-v1`.

Use this runbook whenever `ship db:promote` indicates a SQLite source is promotable.

## Scope

- Source runtime: embedded SQLite (`database.mode=embedded`, `database.driver=sqlite`)
- Target runtime: standalone Postgres (`database.mode=standalone`, `database.driver=postgres`)
- State machine: `promotion-state-machine-v1`

## Canonical Sequence

1. Freeze writes for the source app.
2. Record baseline runtime metadata (`ship runtime:report --json`) and migration status (`ship db:status`).
3. Apply or preview canonical config mutation with `ship db:promote [--dry-run] [--json]`.
4. Run target migrations with `ship db:migrate`.
5. Export SQLite data with `ship db:export --json`.
6. Import into Postgres with `ship db:import --json`.
7. Verify import evidence with `ship db:verify-import --json`.
8. Unfreeze writes after verification checks are accepted.

## State Handling

- If `ship db:promote` reports `sqlite-source-ready`, continue with the sequence above.
- If it reports `config-mutated-awaiting-import` or `import-complete-awaiting-verify`, do not rerun `ship db:promote`; resume import/verification steps.
- If it reports `inconsistent-runtime-state`, recover runtime metadata to a known-good state before proceeding.

## Notes

- This runbook is intentionally manual-first.
- `ship db:export`, `ship db:import`, and `ship db:verify-import` currently expose deterministic planning/report contracts for humans and LLM tooling.
