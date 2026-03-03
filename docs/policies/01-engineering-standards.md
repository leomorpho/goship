# Repository Requirements (Engineering Standards)

This file defines baseline requirements for any GoShip repository (framework, CLI, or app) to stay maintainable.

## 1) Repository Layout

1. Keep runtime/app code, framework code, and CLI code clearly separated.
2. Keep architecture and operational docs in `docs/`.
3. Keep one living plan document for roadmap/decisions.

## 2) Local Developer Workflow

Required commands must exist and work:

1. `make dev` (web-only default)
2. `make test` (fast Docker-free unit set)
3. `make test-integration` (infra-backed integration set)
4. `make testall` (unit + integration)

If CLI exists, equivalent commands must exist in CLI:

1. `ship dev`
2. `ship test`
3. `ship test --integration`

## 3) Pre-Commit Hooks (Required)

Use `lefthook` with at least:

1. unit test package set
2. formatting checks for touched languages
3. basic static checks (as adopted by repo stage)

Rules:

1. No bypass by default.
2. Hook runtime should stay fast (target under ~60s on normal changes).
3. Integration tests are not required in pre-commit.

## 4) CI Requirements (Required)

Every PR must run:

1. lint/format checks
2. `make test`
3. selected integration tests (or full `make test-integration` where feasible)

Main branch protection should require CI green before merge.

## 5) Test Strategy

1. Prefer table-driven unit tests.
2. Keep business logic testable without Docker where possible.
3. Use integration tests for external systems and process boundaries only.
4. Keep package-level coverage trending to 90%+ over time.

## 6) Versioning and Tooling

1. Pin project tools to declared versions (do not auto-latest on normal commands).
2. Provide a doctor/check command to detect version drift.
3. Use explicit upgrade workflows for intentional version bumps.

## 7) Documentation Requirements

For each behavior/architecture change:

1. Update relevant `docs/*.md` files in the same change stream.
2. Keep CLI contract in `docs/reference/01-cli.md` current.
3. Keep framework plan (`docs/roadmap/01-framework-plan.md`) aligned with decisions.

## 8) Commit and PR Standards

1. Use conventional commit prefixes (`feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`).
2. Keep commits scoped and reviewable.
3. PR description must include:
: what changed
: why it changed
: test evidence
: docs updated

## 9) Standalone Repo Readiness Checklist

A module/repo is ready to stand alone when:

1. it has its own `README` and usage commands
2. it has independent tests passing in CI
3. it has pinned toolchain policy documented
4. it has release/versioning policy documented
5. it can be developed with minimal implicit dependency on sibling repos
