# Documentation Index

This `docs/` directory is internal and implementation-focused.
It is not intended as user-facing product documentation.

## Goals

- Explain what the project does today based on code, not assumptions.
- Give developers and AI agents a fast map of where to make changes.
- Capture system risks and incomplete areas so work is directed intentionally.

## README Index

1. `README-01` - `../README.md`: project-level overview and onboarding.
2. `README-02` - `00-index.md`: documentation hub and map of all internal docs.

## Structure

### Architecture

1. `A01` - `architecture/01-architecture.md`: runtime architecture, request flow, and service composition.
2. `A02` - `architecture/02-structure-and-boundaries.md`: canonical placement rules for app vs framework code.
3. `A03` - `architecture/03-project-scope-analysis.md`: end-to-end feature and capability analysis.
4. `A04` - `architecture/04-http-routes.md`: route inventory grouped by access level and purpose.
5. `A05` - `architecture/05-data-model.md`: Ent entities and domain model coverage.
6. `A06` - `architecture/06-known-gaps-and-risks.md`: confirmed implementation gaps and technical risks.

### Guides

1. `G01` - `guides/01-ai-agent-guide.md`: practical guide for AI agents working in this repo.
2. `G02` - `guides/02-development-workflows.md`: day-to-day run/build/test/migration workflows.

### Reference

1. `R01` - `reference/01-cli.md`: living CLI specification (`ship`) for developers and agents.
2. `R02` - `reference/02-mcp.md`: living MCP spec (`ship-mcp`) for LLM-facing docs and CLI support.

### Policies

1. `P01` - `policies/01-engineering-standards.md`: baseline requirements for maintainable repositories (hooks, CI, tests, docs, versioning).

### Roadmap

1. `M01` - `roadmap/01-framework-plan.md`: long-term framework strategy and execution tracker.

## Primary Source Files Used For This Analysis

- `cmd/web/main.go`
- `cmd/worker/main.go`
- `cmd/seed/main.go`
- `cli/ship/cmd/ship/main.go`
- `cli/ship/cli.go`
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
