# Documentation Index

This `docs/` directory is internal and implementation-focused.
It is not intended as user-facing product documentation.

Last updated: 2026-03-04

## Goals

- Explain what the project does today based on code, not assumptions.
- Give developers and AI agents a fast map of where to make changes.
- Capture system risks and incomplete areas so work is directed intentionally.

## Entrypoints

1. `README` - `../README.md`: concise repo landing page for new contributors.
2. `DOCS` - `00-index.md`: documentation hub and map of all internal docs.
3. `MCP` - `../MCP_TOOLS.md`: recommended external MCP tools for contributors (GitHub MCP, GoShip MCP, Playwright MCP).

## Structure

### Architecture

1. `A01` - `architecture/01-architecture.md`: runtime architecture, request flow, and service composition.
2. `A02` - `architecture/02-structure-and-boundaries.md`: canonical placement rules for app vs framework code.
3. `A03` - `architecture/03-project-scope-analysis.md`: end-to-end feature and capability analysis.
4. `A04` - `architecture/04-http-routes.md`: route inventory grouped by access level and purpose.
5. `A05` - `architecture/05-data-model.md`: Data queries and domain model coverage.
6. `A06` - `architecture/06-known-gaps-and-risks.md`: confirmed implementation gaps and technical risks.
7. `A07` - `architecture/07-core-interfaces.md`: backend-agnostic adapter seam contracts.
8. `A08` - `architecture/08-cognitive-model.md`: cognitive model and DX/LLM reliability guardrails.
9. `A09` - `architecture/09-standalone-and-managed-mode.md`: canonical boundary between standalone GoShip capability and external managed-service authority.

### Guides

1. `G01` - `guides/01-ai-agent-guide.md`: practical guide for AI agents working in this repo.
2. `G02` - `guides/02-development-workflows.md`: day-to-day run/build/test/migration workflows.
3. `G03` - `guides/03-how-to-playbook.md`: prioritized how-to guide backlog and writing template.
4. `G04` - `guides/04-deployment-kamal.md`: current deployment workflow via Kamal.
5. `G05` - `guides/05-jobs-module.md`: jobs module install/wiring contract, backend rules, and migration notes.
6. `G06` - `guides/06-ai-module.md`: AI module provider selection, request contract, and SSE demo pattern.
7. `G07` - `guides/07-domain-events.md`: domain event bus, shared event types, and async enqueue contract.
8. `G08` - `guides/08-building-an-api.md`: JSON response helpers, content negotiation, and versioned API route convention.
9. `G09` - `guides/09-i18n-adapter-contract.md`: installable i18n adapter interface contract, compatibility harness, and implementer checklist.
10. `G10` - `guides/10-i18n-llm-migration-workflow.md`: LLM-first i18n migration loop, diagnostics schema, and strict-mode rollout contract.

### Reference

1. `R01` - `reference/01-cli.md`: living CLI specification (`ship`) for developers and agents.
2. `R02` - `reference/02-mcp.md`: living MCP spec (`ship-mcp`) for LLM-facing docs and CLI support.

### UI

1. `U01` - `ui/style-guide.md`: design system reference — theme tokens, typography, dark mode, layout patterns, HTMX swap patterns, component libraries. **Read before any UI work.**
2. `U02` - `ui/convention.md`: `data-component`, `data-slot`, `data-action`, `// Renders:`, `// Route(s):` annotation rules for templ components.

### Policies

1. `P01` - `policies/01-engineering-standards.md`: baseline requirements for maintainable repositories (hooks, CI, tests, docs, versioning).

### Roadmap

1. `M01` - `roadmap/01-framework-plan.md`: long-term framework strategy and execution tracker.
2. `M02` - `roadmap/02-architecture-evolution.md`: architectural direction — islands JS, module extraction, app split, MCP expansion.
3. `M03` - `roadmap/03-atomic-tasks.md`: atomic task list for implementing M02, pickup-ready for any LLM agent.
4. `M04` - `roadmap/04-pagoda-and-dx-improvements.md`: ideas from Pagoda and Rails/Laravel — single binary mode, Backlite, Otter, admin panel, and more.
5. `M05` - `roadmap/05-llm-dx-agent-friendly.md`: convention-over-configuration enforcement, ship verify, ship describe, hierarchical CLAUDE.md, route contracts, test-first scaffolding, agent worktree workflow, MCP tool expansion.
6. `M06` - `roadmap/06-dx-and-infrastructure.md`: ship dev unified command, GitHub Actions CI/CD, SQLite multi-process safety, slog structured logging, security headers, health checks, email system (mailer interface, templ templates, dev previews), cron scheduling, app-level CLI commands.
7. `M07` - `roadmap/07-modules-and-capabilities.md`: OAuth/social login, 2FA/TOTP, AI module (Anthropic + OpenAI + OpenRouter + streaming), domain events, soft deletes, feature flags, audit log, SSE, JSON API pattern, OpenAPI spec generation, test data factories, HTTP test helpers, i18n.
8. `M08` - `roadmap/08-ui-agent-context.md`: work order for applying UI agent convention (data-* attributes, templ comments, route annotations) across this repo.

## Primary Source Files Used For This Analysis

- `cmd/web/main.go`
- `cmd/worker/main.go`
- `cmd/seed/main.go`
- `tools/cli/ship/cmd/ship/main.go`
- `tools/cli/ship/internal/cli/cli.go`
- `app/foundation/container.go`
- `framework/core/interfaces.go`
- `framework/core/adapters/registry.go`
- `framework/core/adapters/resolve.go`
- `app/foundation/core_cache_adapter.go`
- `app/foundation/core_jobs_adapter.go`
- `app/foundation/core_pubsub_adapter.go`
- `app/router.go`
- `app/web/controllers/*.go`
- `app/jobs/*.go`
- `framework/repos/**/*.go`
- `db/queries/*.sql`
- `config/config.go`
- `.env.example`
- `config/modules.yaml`
- `Makefile`
- `frontend/vite.config.ts`
- `frontend/package.json`
- `tests/e2e/tests/goship.spec.ts`
