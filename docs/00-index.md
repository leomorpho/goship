# Documentation Index

This `docs/` directory is internal and implementation-focused.
It is not intended as user-facing product documentation.

Last updated: 2026-03-03

## Goals

- Explain what the project does today based on code, not assumptions.
- Give developers and AI agents a fast map of where to make changes.
- Capture system risks and incomplete areas so work is directed intentionally.

## Entrypoints

1. `README` - `../README.md`: concise repo landing page for new contributors.
2. `DOCS` - `00-index.md`: documentation hub and map of all internal docs.

## Structure

### Architecture

1. `A01` - `architecture/01-architecture.md`: runtime architecture, request flow, and service composition.
2. `A02` - `architecture/02-structure-and-boundaries.md`: canonical placement rules for app vs framework code.
3. `A03` - `architecture/03-project-scope-analysis.md`: end-to-end feature and capability analysis.
4. `A04` - `architecture/04-http-routes.md`: route inventory grouped by access level and purpose.
5. `A05` - `architecture/05-data-model.md`: Ent entities and domain model coverage.
6. `A06` - `architecture/06-known-gaps-and-risks.md`: confirmed implementation gaps and technical risks.
7. `A07` - `architecture/07-core-interfaces.md`: backend-agnostic adapter seam contracts.
8. `A08` - `architecture/08-cognitive-model.md`: cognitive model and DX/LLM reliability guardrails.

### Guides

1. `G01` - `guides/01-ai-agent-guide.md`: practical guide for AI agents working in this repo.
2. `G02` - `guides/02-development-workflows.md`: day-to-day run/build/test/migration workflows.
3. `G03` - `guides/03-how-to-playbook.md`: prioritized how-to guide backlog and writing template.
4. `G04` - `guides/04-deployment-kamal.md`: current deployment workflow via Kamal.

### Reference

1. `R01` - `reference/01-cli.md`: living CLI specification (`ship`) for developers and agents.
2. `R02` - `reference/02-mcp.md`: living MCP spec (`ship-mcp`) for LLM-facing docs and CLI support.

### Policies

1. `P01` - `policies/01-engineering-standards.md`: baseline requirements for maintainable repositories (hooks, CI, tests, docs, versioning).

### Roadmap

1. `M01` - `roadmap/01-framework-plan.md`: long-term framework strategy and execution tracker.
2. `M02` - `roadmap/02-dx-llm-phases.md`: active multi-phase DX + LLM reliability execution tracker.

## Primary Source Files Used For This Analysis

- `cmd/web/main.go`
- `cmd/worker/main.go`
- `cmd/seed/main.go`
- `cli/ship/cmd/ship/main.go`
- `cli/ship/cli.go`
- `apps/goship/foundation/container.go`
- `pkg/core/interfaces.go`
- `pkg/core/adapters/registry.go`
- `pkg/core/adapters/resolve.go`
- `apps/goship/foundation/core_cache_adapter.go`
- `apps/goship/foundation/core_jobs_adapter.go`
- `apps/goship/foundation/core_pubsub_adapter.go`
- `apps/goship/router.go`
- `apps/goship/web/controllers/*.go`
- `pkg/tasks/*.go`
- `pkg/repos/**/*.go`
- `apps/db/schema/*.go`
- `config/config.go`
- `config/application.yaml`
- `config/environments/*.yaml`
- `config/processes.yaml`
- `Makefile`
- `build.mjs`
- `package.json`
- `e2e_tests/tests/goship.spec.ts`
