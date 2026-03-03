# How-To Playbook (Docs-First)

This file tracks practical how-to guides we want to provide for GoShip.

## Objective

Create docs quality equal to or better than Pagoda's onboarding experience, with concrete implementation guides for common tasks.

## Priority Guides

1. Add a new endpoint (route + handler + templ view + tests).
2. Add a new page with server-rendered UI.
3. Add a new Ent model and migration flow.
4. Add a new service/repository with boundaries (app vs framework placement).
5. Add a background job and wire worker behavior.
6. Add a module adapter (db/cache/jobs/pubsub/storage) with interface wiring.
7. Add realtime event flow (publish + subscribe + UI update).
8. Add authentication-protected endpoint and authorization check.
9. Add table-driven unit tests for routes/repos/services.
10. Add integration test (happy path only, Docker-minimal).

## Guide Template (Use For Every How-To)

1. Goal: one-sentence desired outcome.
2. Preconditions: files, commands, env vars needed.
3. Steps: exact edits/commands in order.
4. Validation: tests/commands and expected output.
5. Common failures: top 3 mistakes and fixes.
6. References: links to canonical docs and source files.

## Writing Rules

1. Prefer copy-pastable commands and exact file paths.
2. Use current numbered docs paths in all references.
3. Keep examples aligned with `ship` commands where available.
4. Every guide must include a test/verification section.
5. Keep sections short and task-oriented; avoid long conceptual digressions.

## Execution TODO

- [ ] Draft guide: Add a new endpoint.
- [ ] Draft guide: Add a new Ent model and migration.
- [ ] Draft guide: Add a background job.
- [ ] Draft guide: Add tests (table-driven + integration).
- [ ] Draft guide: Add a module adapter.
