# Bob + Goose Transition Plan (Temporary Living Checklist)

Status: active temporary execution tracker. Delete this file when all checklist items are complete.

Last updated: 2026-03-05

## Scope

Transition GoShip from Ent-centric data ownership to Bob + Goose with module-owned database boundaries.

Non-goals for this plan:
- Building a custom Rails/Laravel/Django-style model DSL now.
- Supporting multi-database per mini-app now.

## Locked Decisions

- [x] Use **Bob** for DB code generation.
- [x] Use **Goose** as the default migration tool.
- [x] Keep one physical DB per app repo.
- [ ] Keep Ent only as temporary compatibility during migration.
- [ ] Remove Ent entirely when parity is complete.

## Architecture Target (Definition of Done)

- [ ] Core app owns only core DB artifacts.
- [ ] Each installable module owns its DB artifacts end-to-end.
- [ ] `ship db:*` commands run Goose + Bob in a deterministic way.
- [ ] No module DB package imports from app core DB package.
- [ ] No app feature logic depends directly on Ent.

Target ownership layout:

- Core:
  - `db/migrate/migrations/*` (transition baseline)
  - `db/schema/*.go` (Ent transition baseline)
  - `db/queries/*.sql` (Bob target)
  - `db/gen/*` (Bob target)

- Module:
  - `modules/<name>/db/migrations/*`
  - `modules/<name>/db/schema/*.sql` (optional)
  - `modules/<name>/db/queries/*.sql`
  - `modules/<name>/db/gen/*`

## Global Test Gate (Mandatory)

No checklist item is marked done unless all applicable tests pass.

Required gates for behavior changes:
- [ ] Unit tests added/updated for changed command/runtime logic.
- [ ] Integration test added/updated for end-to-end behavior.
- [ ] `bash tools/scripts/precommit-tests.sh` passes.
- [ ] If DB workflow changed: integration path validates real migration + generation flow.

Test design rules:
- [ ] Prefer table-driven unit tests for parser/flag/safety logic.
- [ ] Keep integration tests Docker-free where practical.
- [ ] Use deterministic test fixtures and temp dirs; no writes to live repo tree.
- [ ] Avoid hidden coupling between CLI package tests and app runtime tree.

## Current Baseline Snapshot (As of 2026-03-05)

- [x] Migration runtime is Atlas + Ent today (not Goose + Bob yet).
- [x] Current DB commands are `db:create`, `db:make`, `db:migrate`, `db:status`, `db:drop`, `db:reset`, `db:rollback`, `db:seed`.
- [x] Destructive DB safety contract exists (`--yes`, `--force`, `--dry-run`, production guards).
- [x] Installed-module registry file exists at `config/modules.yaml` (`modules: []` baseline).
- [ ] `db:generate` for Bob does not exist yet.
- [ ] `make:migration` alias does not exist; `db:make` is current canonical command.
- [ ] Goose runtime package/invocation path does not exist yet.

---

## Source-backed Constraints (Bob)

- Bob is database-first and expects schema to exist.
- Migrations are managed outside Bob.
- Query generation from `.sql` is first-class.

Sources:
- https://bob.stephenafamo.com/docs/code-generation/intro
- https://bob.stephenafamo.com/docs/code-generation/queries
- https://bob.stephenafamo.com/docs/code-generation/sql/
- https://bob.stephenafamo.com/vs/ent/

---

## Phase 0: Finalize Contracts and CLI Behavior

Goal: lock behavior before implementation drift starts.

### 0.1 DB command contract
- [x] Define current command set (Atlas/Ent baseline):
  - `ship db:migrate`
  - `ship db:status`
  - `ship db:create`
  - `ship db:drop`
  - `ship db:reset`
  - `ship db:make <migration_name>`
- [ ] Decide transition contract for Bob/Goose:
  - keep `ship db:make <migration_name>` as canonical, or add `ship make:migration <name>` alias.
  - add `ship db:generate` for Bob.
- [x] Define command help text and examples for current DB commands.
- [x] Define dry-run behavior (`--dry-run`) for destructive commands.

Acceptance criteria:
- [x] Commands and flags are documented in `docs/reference/01-cli.md`.
- [x] Commands and flags are covered by CLI unit tests.
- [x] Integration test coverage exists for DB flow and safety paths.
- [ ] Add integration test coverage for new Bob/Goose commands when introduced.

### 0.2 Installed-module registry contract
- [x] Define one canonical installed-module registry location and format (`config/modules.yaml`).
- [ ] Define deterministic execution order (core first, then modules sorted) for DB migrate/generate orchestration.
- [ ] Define module enable/disable semantics for migration/generation participation.

Acceptance criteria:
- [ ] `ship doctor` validates registry file format.
- [ ] Integration test proves deterministic order.
- [ ] Unit tests cover registry parsing and sort behavior.

### 0.3 Migration safety contract
- [x] Define local/dev/prod safety rules for `db:drop` and `db:reset`.
- [x] Define explicit confirmation semantics (`--yes`).
- [x] Define non-local DB protection behavior.

Acceptance criteria:
- [x] Safety behavior is unit-tested and integration-tested.
- [x] Negative tests cover non-local DB protection paths.

---

## Phase 1: Goose Foundation in Ship CLI

Goal: deliver Goose-backed migration commands without changing app data access yet.

Transition note:
- Goose is now wired behind the existing `ship db:*` surface for a constrained DB subset (`postgres`, `mysql`, `sqlite/sqlite3`).
- Remaining Phase 1 work is integration coverage and command-contract cleanup (`db:drop` semantics and migration alias decisions).

### 1.1 Goose runtime wiring
- [x] Add Goose runner path in CLI runtime execution flow.
- [x] Support DB URL resolution from one canonical env var path.
- [x] Remove duplicate/ambiguous DB URL env resolution.

Acceptance criteria:
- [x] Unit tests for URL resolution and Goose invocation.
- [x] Integration smoke test verifies Goose runtime can execute with temp DB.

### 1.2 Core migration commands
- [x] Implement `ship db:migrate` (core app migrations only initially).
- [x] Implement `ship db:status`.
- [x] Implement `ship db:make <migration_name>` on Goose runtime (optionally add `ship make:migration <name>` alias if chosen in Phase 0.1).

Acceptance criteria:
- [x] Unit tests for command parsing and runtime calls.
- [x] Integration test: create migration -> migrate -> status reflects applied.
- [x] Regression test: rerun migrate is idempotent.

### 1.3 Destructive lifecycle commands
- [x] Implement `ship db:create`.
- [x] Implement `ship db:drop`.
- [x] Implement `ship db:reset` (`drop + create + migrate`, optional seed step).

Acceptance criteria:
- [x] Safety checks tested.
- [x] Integration test for local reset happy path.
- [x] Integration test for reset refusal on protected/non-local target.

---

## Phase 2: Bob Foundation (Core Only)

Goal: introduce Bob generation for core with minimal service-layer disruption.

### 2.1 Bob config and generation entrypoint
- [ ] Add core Bob config file(s).
- [ ] Add `ship db:generate` for core generation.
- [ ] Add deterministic generation output locations.

Acceptance criteria:
- [ ] `ship db:generate` idempotency test (no diff on second run).
- [ ] Pre-commit check ensures generated code is current.
- [ ] Integration test verifies generate succeeds from clean temp project.

### 2.2 Core DB adapter seam
- [ ] Create app-internal data adapter interfaces independent of Ent/Bob.
- [ ] Implement Bob-backed adapter for first core path.
- [ ] Keep Ent adapter temporarily for fallback during migration.

Acceptance criteria:
- [ ] Unit tests for adapter behavior parity.
- [ ] No controller imports from Ent package.
- [ ] Integration test covers one migrated read/write path through app API.

---

## Phase 3: Module-owned DB Boundaries (First Module)

Goal: prove full module isolation end-to-end.

### 3.1 Select and migrate first module
- [ ] Pick first module (recommended: `emailsubscriptions` for lower risk).
- [ ] Create module-local Goose migrations.
- [ ] Add module-local Bob config + generated package.
- [ ] Refactor module services to use module-local generated DB package.

Acceptance criteria:
- [ ] Module compiles standalone.
- [ ] Module tests pass standalone.
- [ ] Module no longer imports core DB package.
- [ ] Integration test covers module install + migrate + functional happy path.

### 3.2 CLI orchestration for module migrations
- [ ] Update `ship db:migrate` to run:
  - core migrations
  - installed modules in deterministic order
- [ ] Update `ship db:status` to display core + module status.
- [ ] Update `ship db:generate` for core + installed modules.

Acceptance criteria:
- [ ] Integration test: fresh project + install module + migrate + generate succeeds.
- [ ] Failure behavior tested (module migration fails -> aborts cleanly).
- [ ] Regression test: ordering remains deterministic across repeated runs.

---

## Phase 4: Expand Module Coverage

Goal: move all installable modules to module-owned DB layers.

### 4.1 Migrate remaining modules
- [ ] `jobs`
- [ ] `notifications`
- [ ] `paidsubscriptions`
- [ ] any additional module added during transition

Acceptance criteria:
- [ ] Every module has local migrations + generation.
- [ ] No module imports core DB package.
- [ ] Module-isolation checks enforce boundary automatically in precommit.

### 4.2 Module contract hardening
- [ ] Standardize module DB package structure template.
- [ ] Add `ship` scaffolder support for new module DB skeleton.

Acceptance criteria:
- [ ] New module generation integration test covers DB scaffold.
- [ ] Test verifies scaffold is generated in module-local DB paths only.

---

## Phase 5: Ent Decommission

Goal: remove Ent once runtime parity is complete.

### 5.1 Removal readiness checks
- [ ] Confirm no runtime code paths require Ent.
- [ ] Confirm migrations and generators run fully via Goose + Bob.

### 5.2 Remove Ent toolchain
- [ ] Remove Ent generation commands from Ship and Make flow.
- [ ] Remove `db/ent` and Ent-specific docs.
- [ ] Remove Ent dependencies from `go.mod`/`go.work`.

Acceptance criteria:
- [ ] `ship doctor` has no Ent-related expectations.
- [ ] Full pre-commit and integration suite green without Ent.
- [ ] Clean-room build/test verifies no hidden Ent dependency remains.

---

## Phase 6: DX Hardening and Docs

Goal: ensure long-term maintainability and agent reliability.

### 6.1 Docs and runbooks
- [ ] Update `docs/reference/01-cli.md` for final DB command behavior.
- [ ] Add/refresh how-to guides for migration + generation workflows.
- [ ] Keep `LLM.txt` generation aligned.

### 6.2 Guardrails
- [ ] Add doctor checks for:
  - missing module migrations dir
  - missing module Bob config
  - forbidden cross-boundary imports
- [ ] Add CI checks for generated-code drift.

Acceptance criteria:
- [ ] `ship doctor` enforces architecture contracts.
- [ ] CI catches drift and boundary violations.
- [ ] Docs checks fail CI when CLI behavior and docs diverge.

---

## Cross-phase Risk Register

- [ ] Risk: command churn during transition confuses users.
  - Mitigation: lock command contract in Phase 0 and stick to it.

- [ ] Risk: mixed Ent/Bob period creates duplicate write paths.
  - Mitigation: adapter seam + strict feature-by-feature cutover.

- [ ] Risk: module migration order bugs.
  - Mitigation: deterministic ordering + integration tests + failure tests.

- [ ] Risk: generated code noise in PRs.
  - Mitigation: deterministic generation + pre-commit drift checks.

---

## Execution Log (append-only)

- 2026-03-04: locked Bob + Goose decisions and created this temporary checklist.
- 2026-03-05: aligned transition checklist with current Atlas/Ent baseline and clarified Bob/Goose delta tasks.
- 2026-03-05: switched `ship db:*` command execution to Goose with initial supported subset (`postgres`, `mysql`, `sqlite/sqlite3`) and updated CLI unit coverage.
- 2026-03-05: verified Goose DB flow integration (`make -> migrate -> status -> reset`) and added migrate idempotency assertion.

---

## Deletion Conditions

Delete this file when all are true:

- [ ] All phases complete.
- [ ] No active Ent runtime dependency.
- [ ] Permanent docs fully reflect Goose + Bob workflows.
- [ ] Temporary transition tasks are no longer referenced.


## Phase Completion Rule

A phase can be marked complete only when:
- [ ] All phase checkboxes are done.
- [ ] All phase acceptance criteria are done.
- [ ] Global Test Gate is satisfied for all changes in the phase.
- [ ] Execution log contains a dated completion note.
