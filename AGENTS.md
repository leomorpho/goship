# AGENTS.md

Canonical instructions for AI coding agents working in this repository.

## Project Intent

GoShip is being evolved into a Rails-inspired Go framework with strong developer ergonomics, modular adapters, and LLM-friendly workflows.

Naming normalization from roadmap:

- `shiri`, `jerry`, `sherry` => `Cherie`

## Read Order (Before Editing)

1. `docs/00-index.md`
2. `docs/architecture/02-structure-and-boundaries.md`
3. `docs/architecture/01-architecture.md`
4. `docs/architecture/07-core-interfaces.md`
5. `docs/guides/01-ai-agent-guide.md`
6. `docs/reference/01-cli.md`
7. `docs/roadmap/01-framework-plan.md`
8. `docs/roadmap/02-dx-llm-phases.md`

## Current Priorities

1. Preserve and improve developer ergonomics.
2. Keep docs as first-class, updated with code.
3. Keep default test loop fast and mostly stateless.
4. Keep Cherie compatibility visible for framework changes.
5. Keep files LLM-friendly: target <= 500 lines per file when practical.

## Architectural Placement Rules

- App-specific web code: `apps/site/*`
- HTTP handlers/routes: `apps/site/web/controllers/*`
- Canonical app router entrypoint: `apps/site/router.go`
- App composition container/adapters: `apps/site/foundation/*`
- Web middleware: `apps/site/web/middleware/*`
- View models: `apps/site/web/viewmodels/*`
- Web rendering helpers: `apps/site/web/ui/*`
- Background jobs/processors: `apps/site/jobs/*`
- Framework/infrastructure layer: `pkg/*`
- ORM schema: `apps/db/schema/*`
- Config: `config/*`
- Templates: `apps/site/views/**/*.templ`

When in doubt, follow `docs/architecture/02-structure-and-boundaries.md`.

## Execution Workflow

1. Identify change scope first: routing, service/repo, domain/schema, UI/templates, tooling/docs.
2. Make the smallest coherent change first.
3. Run targeted tests for touched packages.
4. Run broader tests when risk crosses package/process boundaries.
5. Update docs in the same change stream.
6. If a touched file grows past ~500 lines, split by responsibility unless there is a strong reason not to.

## Testing and Quality Gates

- Prefer table-driven unit tests.
- Keep logic testable without Docker where possible.
- Use integration tests for external/process boundaries.
- Mark integration tests with `//go:build integration`.
- Run integration paths with `go run ./cli/ship/cmd/ship test --integration`.
- Pre-commit must pass (`lefthook`).
- Aim for 90%+ package coverage trend over time.

Common commands:

- `make dev`
- `make test`
- `make testall`
- `make templ-gen`
- `make ent-gen`
- `make makemigrations name=your_change`
- `make migrate`
- `make db-status`
- `make db-create`
- `make db-reset`
- `go run ./cli/ship/cmd/ship dev`
- `go run ./cli/ship/cmd/ship test`
- `go run ./cli/ship/cmd/ship test --integration`
- `go run ./cli/ship/cmd/ship db:make your_change`
- `go run ./cli/ship/cmd/ship db:migrate`
- `go run ./cli/ship/cmd/ship db:status`
- `go run ./cli/ship/cmd/ship db:reset --yes`

## Documentation Sync (Required)

For any behavior/architecture change, update relevant docs.

At minimum:

- Capability changes: `docs/architecture/03-project-scope-analysis.md`
- Route surface changes: `docs/architecture/04-http-routes.md`
- Risk/known-gap changes: `docs/architecture/06-known-gaps-and-risks.md`
- CLI behavior changes: `docs/reference/01-cli.md`
- Plan/decision changes: `docs/roadmap/01-framework-plan.md`

If landing/docs UI capability sections change, keep `FRONTEND_SYNC` linked docs aligned.

## Git and Commit Policy

- Use Conventional Commits: `type(scope): imperative summary`.
- Keep commits scoped and reviewable.
- Do not rewrite `main` without explicit user request.
- Do not commit unless user explicitly asks.

## Branch Safety

When doing high-risk history operations:

1. Create a safety branch first.
2. Verify commit preservation explicitly.
3. Use `--force-with-lease` instead of raw `--force`.

## MCP and Tooling Notes

Recommended MCP setup is documented in `MCP_TOOLS.md`.

- GitHub MCP is recommended for repo/branch/history operations.
- GoShip MCP remains optional/future-facing.

## Known Pitfalls

- Assuming optional deps are initialized in container by default.
- Adding routes without registration in router composition.
- Schema/model changes without migration + generation workflow.
- Template/layout changes without regenerating templ outputs.

## Definition of Done (Agent Task)

A task is done when:

1. Code changes are complete and coherent.
2. Required tests were run and passed (or limitation documented).
3. Docs were updated where required.
4. Result and risks are clearly summarized to the user.
