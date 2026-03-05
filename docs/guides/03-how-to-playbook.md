# How-To Playbook

Task-focused guides for common GoShip workflows.

## 1) Add a New Endpoint (Resource + Route + Name)

Goal:
- add a new public endpoint with controller + route wiring + route-name constant.

Preconditions:
- run from repo root.
- `app/router.go` has ship route markers.

Steps:
```bash
go run ./tools/cli/ship/cmd/ship make:resource contact_form --path app --auth public --views templ --wire
```

Validation:
```bash
go test ./tools/cli/ship -count=1
go run ./tools/cli/ship/cmd/ship doctor
```
- expect `ship doctor: OK`.

Common failures:
1. Missing markers in `app/router.go`: restore marker pairs.
2. Existing controller file: pick another name or remove conflicting file.
3. Wrong auth group: use `--auth public|auth`.

## 2) Add a New Migration + Bob Generation

Goal:
- add a DB change, apply it, and regenerate query code.

Steps:
```bash
go run ./tools/cli/ship/cmd/ship db:make add_posts
go run ./tools/cli/ship/cmd/ship db:migrate
go run ./tools/cli/ship/cmd/ship db:generate
```

Validation:
```bash
go run ./tools/cli/ship/cmd/ship db:status
bash tools/scripts/check-bobgen-drift.sh
```

Common failures:
1. Unsupported DB URL scheme: use `postgres`, `mysql`, `sqlite`, or `sqlite3`.
2. Missing module migrations/config for enabled module: fix module DB scaffold (`db/migrate/migrations`, `db/bobgen.yaml`).
3. `PAGODA_DATABASE_URL` set: use `DATABASE_URL` only.

## 3) Add a New Controller (No View)

Goal:
- add a controller with explicit actions and route wiring.

Steps:
```bash
go run ./tools/cli/ship/cmd/ship make:controller Posts --actions index,show,create --auth auth --wire
```

Validation:
```bash
go test ./tools/cli/ship -count=1
go run ./tools/cli/ship/cmd/ship doctor
```
- confirm one generated block in `app/router.go`.

Common failures:
1. Duplicate controller file: rename or delete existing file.
2. Missing route markers: restore `ship:routes:*` markers.
3. Invalid action name: use only `index,show,create,update,destroy`.

## 4) Add a Background Job

Goal:
- add a jobs processor path and validate worker startup surface.

Steps:
1. add/update job logic under `app/jobs`.
2. wire dependencies via `app/foundation/container.go` as needed.
3. run worker locally:
```bash
go run ./cmd/worker
```

Validation:
```bash
go test ./app/jobs ./cmd/worker -count=1
```

Common failures:
1. Job depends on uninitialized adapter: check container wiring.
2. Worker-runtime mismatch with web runtime plan: verify config/process topology.
3. Missing test seam: extract pure logic into testable functions.

## 5) Add Tests (Unit + Integration)

Goal:
- keep fast stateless default tests and explicit integration tests.

Steps:
1. add table-driven unit tests near changed package.
2. add integration tests in `tools/cli/ship` or affected package with build tag:
```go
//go:build integration
```
3. run:
```bash
go run ./tools/cli/ship/cmd/ship test
go run ./tools/cli/ship/cmd/ship test --integration
```

Validation:
- unit and integration paths pass independently.

Common failures:
1. Integration tests running in unit path: missing build tag.
2. Fixture tests touching live repo tree: use temp dirs.
3. Slow tests in unit path: move external/process tests behind integration tag.

## 6) Add/Swap an Adapter Boundary

Goal:
- integrate a backend-specific implementation behind core interfaces.

Steps:
1. confirm interface contract in `framework/core/interfaces.go`.
2. implement adapter in `framework/repos/<area>` or app-scoped package if app-specific.
3. wire in `app/foundation`.
4. validate with:
```bash
go run ./tools/cli/ship/cmd/ship doctor
go test ./... 
```

Common failures:
1. App-specific logic placed in framework package.
2. Missing shutdown/lifecycle handling in container.
3. Route/controller code directly using backend package instead of interface seam.

## 7) Add DB Artifacts to an Installable Module

Goal:
- keep module DB ownership isolated from app core.

Steps:
```bash
# scaffold or verify module DB layout
go run ./tools/cli/ship/cmd/ship make:module MyModule --path modules --dry-run

# for an installed module, add migration + queries + generation config under:
# modules/<name>/db/migrate/migrations
# modules/<name>/db/queries
# modules/<name>/db/bobgen.yaml

go run ./tools/cli/ship/cmd/ship db:migrate
go run ./tools/cli/ship/cmd/ship db:generate
```

Validation:
```bash
bash tools/scripts/check-module-isolation.sh
go run ./tools/cli/ship/cmd/ship doctor
```

Common failures:
1. Module imports app/core DB package directly: fix imports to module-local DB package.
2. Missing `db/bobgen.yaml` in enabled module: doctor and `db:generate` fail.
3. Migration order assumptions: module execution is deterministic sorted order only.

## References

- `docs/reference/01-cli.md`
- `docs/architecture/02-structure-and-boundaries.md`
- `docs/architecture/08-cognitive-model.md`
- `docs/guides/05-jobs-module.md`
