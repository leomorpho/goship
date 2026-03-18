# AGENTS.md

Canonical instructions for AI coding agents working in this repository.

## Project Intent

GoShip is being evolved into a Rails-inspired Go framework with strong developer ergonomics, modular adapters, and LLM-friendly workflows.

Naming normalization from roadmap:

- `shiri`, `jerry`, `sherry` => `Cherie`

## Read Order (Before Editing)

1. `docs/00-index.md`
2. `docs/architecture/02-structure-and-boundaries.md`
3. `docs/architecture/09-standalone-and-managed-mode.md`
4. `docs/architecture/01-architecture.md`
5. `docs/architecture/07-core-interfaces.md`
6. `docs/guides/01-ai-agent-guide.md`
7. `docs/reference/01-cli.md`
8. `docs/roadmap/01-framework-plan.md`
9. `docs/guides/05-jobs-module.md`

## Current Priorities

1. Preserve and improve developer ergonomics.
2. Keep docs as first-class, updated with code.
3. Keep default test loop fast and mostly stateless.
4. Keep Cherie compatibility visible for framework changes.
5. Keep files LLM-friendly: target <= 500 lines per file when practical.

## Architectural Placement Rules

- App-specific web code: `app/*`
- HTTP handlers/routes: `app/web/controllers/*`
- Canonical app router entrypoint: `app/router.go`
- App composition container/adapters: `app/foundation/*`
- Web middleware: `app/web/middleware/*`
- View models: `app/web/viewmodels/*`
- Web rendering helpers: `app/web/ui/*`
- Background jobs/processors: `app/jobs/*`
- Framework/infrastructure layer: `framework/*`
- DB queries + generation: `db/queries/*`, `db/gen/*`
- Config: `config/*`
- Templates: `app/views/**/*.templ`

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
- Run integration paths with `go run ./tools/cli/ship/cmd/ship test --integration`.
- Pre-commit must pass (`lefthook`).
- Aim for 90%+ package coverage trend over time.

Common commands:

- `make dev`
- `make test`
- `make testall`
- `make templ-gen`
- `make makemigrations name=your_change`
- `make migrate`
- `make db-status`
- `make db-create`
- `make db-reset`
- `go run ./tools/cli/ship/cmd/ship dev`
- `go run ./tools/cli/ship/cmd/ship test`
- `go run ./tools/cli/ship/cmd/ship test --integration`
- `go run ./tools/cli/ship/cmd/ship db:make your_change`
- `go run ./tools/cli/ship/cmd/ship db:migrate`
- `go run ./tools/cli/ship/cmd/ship db:status`
- `go run ./tools/cli/ship/cmd/ship db:reset --yes`

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

<!-- docket:skill-pack:start -->
# Docket Skill Pack (Codex)

<!-- docket.skill.pack.version: docket.skills/v1 -->
<!-- docket.contract.hash: 4215e96e76b073e7c5b58adccdafa2958d65153bd2e869b3255f7560e863f2e0 -->
<!-- docket.skill.metadata.checksum: 4bbadff18330725650ed9e6233332d2f19ad7494eecfb23b9f4cb939b3b375fc -->
<!-- docket.skill.ids: ticket-discovery,ticket-authoring-apply,context-optimize,learning-replay,wrap-up-readiness -->

Use `docket start` to pick up prioritized ticket work.

### Skills
- `ticket-discovery` (required)
  - title: Discover Next Ticket
  - intent: planning
  - command: docket list --state open --format context
  - triggers: session_start, resume, task_selection
  - summary: Find the next actionable ticket and inspect its working context before coding.
- `ticket-authoring-apply` (required)
  - title: Transactional Ticket Authoring
  - intent: authoring
  - command: docket ticket scaffold --format json
  - triggers: multi_line_ticket_edit, bulk_ticket_changes, automation_mode
  - summary: Use scaffold/apply commands to author or update ticket specs without fragile shell quoting.
- `context-optimize` (optional)
  - title: Compact Ticket Brief
  - intent: context
  - command: docket context-optimize {ticket_id}
  - triggers: llm_context_budget, ticket_handoff, task_brief
  - summary: Generate a bounded brief from ticket context, learnings, and recent activity.
- `learning-replay` (optional)
  - title: Replay Relevant Learnings
  - intent: quality
  - command: docket learn replay {ticket_id}
  - triggers: pre_implementation, incident_recurrence, ticket_resume
  - summary: Replay top ranked learned rules for a ticket using the same ranking model as start.
- `wrap-up-readiness` (optional)
  - title: End-of-Session Wrap-Up
  - intent: review
  - command: docket wrap-up {ticket_id}
  - triggers: session_end, pre_review, handoff
  - summary: Run wrap-up readiness checks for AC completion, handoff quality, blockers, and review transition readiness.
<!-- docket:skill-pack:end -->

