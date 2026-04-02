# Context Snapshot — Post-V1 Unified Generation / Default App / Batteries

## Task statement
Use Ralph to execute the full post-v1 program that turns GoShip from a truthful framework foundation into a generator-first, default-app-rich, module-real productivity framework.

## Desired outcome
- Program 1: unify the generation substrate
- Program 2: deepen the default generated app on top of that substrate
- Program 3: make downstream batteries genuinely installable and machine-readable

## Known facts / evidence
- The starter path is currently template-snapshot based via `ship new` and starter scaffold rendering in `tools/cli/ship/internal/commands/project_new.go`.
- Starter web runtime currently centralizes auth/account flows and generated CRUD dispatch in `tools/cli/ship/internal/templates/starter/testdata/scaffold/cmd/web/main.go`.
- `make:resource` has a starter-specific backend in `tools/cli/ship/internal/generators/resource.go`.
- `make:controller` still rejects the starter scaffold in `tools/cli/ship/internal/generators/controller.go`.
- `make:model` is still query-stub heavy in `tools/cli/ship/internal/generators/model.go`.
- Module policy says batteries belong in `modules/`, not `framework/`, per `docs/architecture/12-boundary-charter.md` and `docs/architecture/11-module-surface-reset.md`.
- Existing planning artifacts already define the broad roadmap in `.omx/plans/p1-p3-groomed-todo-plan.md` and `.omx/plans/post-v1-work-items-ledger.md`.

## Constraints
- Follow the boundary charter: core seams in `framework/`, product capabilities in `modules/`, default leverage in starter preset.
- Keep `make:scaffold` closed until controller + model generation are truthful on the starter path.
- Commit after every major change.
- No new dependencies without explicit request.
- Generated-app truth beats framework-internal convenience.

## Unknowns / open questions
- What minimal generator substrate refactor is enough to make starter-safe controller generation truthful?
- How much of the current starter runtime special-casing should be retired in Program 1 versus tolerated temporarily?
- What packaging/versioning changes are minimally sufficient to make batteries genuinely downstream-installable?

## Likely codebase touchpoints
- `tools/cli/ship/internal/generators/*.go`
- `tools/cli/ship/internal/commands/*`
- `tools/cli/ship/internal/templates/starter/testdata/scaffold/*`
- `docs/architecture/*`
- `modules/*`
- `.github/workflows/*`, `Makefile`, `tools/scripts/*`
