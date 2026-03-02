# Internal Documentation (Developers and AI Agents)

This `docs/` directory is internal and implementation-focused.
It is not intended as user-facing product documentation.

## Goals

- Explain what the project does today based on code, not assumptions.
- Give developers and AI agents a fast map of where to make changes.
- Capture system risks and incomplete areas so work is directed intentionally.

## Document Index

- `project-scope-analysis.md`: End-to-end feature and capability analysis.
- `architecture.md`: Runtime architecture, request flow, and service composition.
- `structure-and-boundaries.md`: Canonical placement rules for app vs framework code.
- `http-routes.md`: Route inventory grouped by access level and purpose.
- `data-model.md`: Ent entities and domain model coverage.
- `ai-agent-guide.md`: Practical guide for AI agents working in this repo.
- `known-gaps-and-risks.md`: Confirmed implementation gaps and technical risks.
- `development-workflows.md`: Day-to-day run/build/test/migration workflows.

## Primary Source Files Used For This Analysis

- `cmd/web/main.go`
- `cmd/worker/main.go`
- `cmd/seed/main.go`
- `pkg/services/container.go`
- `app/goship/web/routes/router.go`
- `app/goship/web/routes/*.go`
- `pkg/tasks/*.go`
- `pkg/repos/**/*.go`
- `ent/schema/*.go`
- `config/config.go`
- `config/config.yaml`
- `Makefile`
- `build.mjs`
- `package.json`
- `e2e_tests/tests/goship.spec.ts`
