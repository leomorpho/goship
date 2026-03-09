# GoShip Atomic Task List

**Reference docs (read before picking up any task):**
- `docs/roadmap/02-architecture-evolution.md` — JS islands, module system, app split, MCP
- `docs/roadmap/04-pagoda-and-dx-improvements.md` — single binary, admin panel, DX improvements
- `docs/guides/01-ai-agent-guide.md` — conventions, safe change workflow
- `docs/ui/convention.md` — data-component / data-slot / Renders: comment rules

**Task format:** Each task is self-contained. It includes full context, exact files to touch,
and a "done when" acceptance criterion. A task is complete only when its criterion is met.
Mark `[x]` before starting any task that depends on it.

**Parallelism:** Tasks marked `(parallel)` within a group have no inter-dependencies and can run simultaneously.

---

## Group A — Critical Bug Fixes

> Do these first. They are live correctness issues, not future work.

### A01 — Fix container initialization for optional services

**Status:** `[x] done`
**Depends on:** nothing
**Files:** `app/foundation/container.go`

**Context:** `initCache()`, `initNotifier()`, `initTasks()` are commented out in `NewContainer()`. Shutdown code calls `.Close()` on potentially nil fields — latent nil-pointer panics. See `docs/architecture/06-known-gaps-and-risks.md`.

**What to do:**
1. Read `app/foundation/container.go` fully.
2. Wrap each commented-out init in a config guard (e.g., `if c.Config.Cache.Enabled { c.initCache() }`). Add `Enabled` booleans to the config struct if they don't exist.
3. Audit all `.Close()` / shutdown calls — nil-check every optional service before calling.
4. Run `go build ./...` and `make test`.

**Done when:** No commented-out init calls remain. Shutdown is nil-safe. `go build` passes.

---

### A02 — Add `--json` output flag to `ship doctor`

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `tools/cli/ship/internal/commands/` (doctor command)

**Context:** `ship doctor` outputs human-readable text. MCP integration and LLM self-validation need machine-readable JSON output.

**What to do:**
1. Find and read the doctor command implementation.
2. Add a `--json` flag.
3. When set, output: `{"ok": bool, "issues": [{"type": "string", "file": "string", "detail": "string", "severity": "error|warning"}]}`
4. Exit code 0 if no errors, 1 if any errors.
5. Existing text output unchanged when flag is absent.

**Done when:** `ship doctor --json` outputs valid JSON matching the schema above. Existing output unchanged.

---

## Group B — JS Islands Architecture

### B01 — Set up Vite build config with island code-splitting (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `frontend/vite.config.ts` (new), `frontend/package.json`

**Context:** Replace esbuild with Vite for per-island code splitting. Each file in `frontend/islands/` becomes a separately loadable chunk. Read `docs/roadmap/02-architecture-evolution.md` section 1.

**What to do:**
1. Read `frontend/build.mjs`, `frontend/package.json`, `frontend/tailwind.config.js`.
2. Add to `package.json` devDependencies: `vite`, `@sveltejs/vite-plugin-svelte`, `vite-plugin-tailwindcss` (or keep postcss).
3. Create `frontend/vite.config.ts`:
   - Glob `frontend/islands/**/*.{svelte,tsx,jsx,vue}` as separate entry points.
   - Output chunks to `app/static/islands/[name]-[hash].js`.
   - Output manifest to `app/static/islands-manifest.json` (name → hashed URL).
   - Also bundle `frontend/javascript/vanilla/main.js` → `app/static/vanilla_bundle.js`.
   - Tailwind via postcss (reuse existing config).
4. Create `frontend/islands/` with a `.gitkeep`.
5. Update `Makefile`: add `js-build-vite` target; keep old `build-js` target during migration.

**Done when:** `make js-build-vite` succeeds. `app/static/islands-manifest.json` is produced. Old esbuild target still works.

---

### B02 — Write the islands runtime (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `frontend/javascript/vanilla/islands-runtime.js` (new), `app/views/web/components/core.templ`

**Context:** Small vanilla JS (~30 lines) that discovers `[data-island]` elements, fetches the Vite manifest, dynamically imports the right chunk, and calls `mount(el, props)`. Must re-run after every HTMX swap.

**What to do:**
1. Create `frontend/javascript/vanilla/islands-runtime.js`:
   - Fetch `/static/islands-manifest.json` once on load, cache in module scope.
   - On `DOMContentLoaded` and `htmx:afterSettle`: `querySelectorAll('[data-island]:not([data-island-mounted])')`.
   - For each: mark `data-island-mounted="true"`, dynamic `import(manifestUrl)`, call `mount(el, JSON.parse(el.dataset.props || '{}'))`.
   - Warn to console if island name not in manifest.
2. Add `<script src={ helpers.File("islands-runtime.js") }></script>` to `app/views/web/components/core.templ` JS block (after HTMX loads).
3. Do NOT remove existing `svelte_bundle.js` — both coexist during migration.

**Done when:** Script exists and is included. Manual test: a `[data-island]` element correctly imports and mounts. Existing Svelte bundle still loads.

---

### B03 — Migrate ThemeToggle to island pattern (parallel)

**Status:** `[ ] todo`
**Depends on:** B01, B02
**Files:** `frontend/islands/ThemeToggle.svelte` (new), `app/views/web/components/theme_toggle.templ`

**Context:** First island migration. Proves the pattern. Each island exports `mount(el, props)`. For Svelte: `export function mount(el, props) { new Component({ target: el, props }) }`.

**What to do:**
1. Read current `ThemeToggle` Svelte component and its templ mounting code.
2. Create `frontend/islands/ThemeToggle.svelte` with component logic + `mount` export.
3. Update templ: replace `<div id={id}> + @initThemeToggle(id)` with `<div data-island="ThemeToggle" data-props={ templ.JSONString(props) }></div>`.
4. Delete old `script` block for this component.
5. Run `make templ-gen`. Test in dev: theme toggle works.

**Done when:** ThemeToggle works via islands runtime. Old `renderSvelteComponent('ThemeToggle', ...)` call is gone.

---

### B04 — Migrate remaining Svelte components to islands

**Status:** `[ ] todo`
**Depends on:** B03 (use as proven pattern)
**Files:** All Svelte files in `frontend/javascript/svelte/`, corresponding templ files

**Context:** Current registry components: `MultiSelectComponent`, `PhotoUploader`, `SingleSelect`, `PhoneNumberPicker`, `PwaInstallButton`, `PwaSubscribePush`, `NotificationPermissions`. Migrate each one following the ThemeToggle pattern from B03. Do one component per commit.

**What to do:** For each component:
1. Read Svelte source + templ mounting code.
2. Create `frontend/islands/{ComponentName}.svelte` with `mount` export.
3. Update templ: `data-island` pattern, remove `script` block.
4. Test manually or via Playwright.
5. Commit.

**Done when:** All components migrated. `window.renderSvelteComponent` is not called anywhere.

---

### B05 — Remove old esbuild setup

**Status:** `[ ] todo`
**Depends on:** B04
**Files:** `frontend/build.mjs`, `frontend/javascript/svelte/main.js`, `Makefile`, `package.json`

**What to do:**
1. Delete `frontend/build.mjs` and `frontend/javascript/svelte/main.js` (registry file).
2. Remove esbuild from `package.json` devDependencies.
3. Remove `svelte_bundle.js` include from `app/views/web/components/core.templ`.
4. Update all `Makefile` targets that referenced old build commands.
5. Run `make js-build` and confirm clean build.

**Done when:** No esbuild references remain. `make js-build` uses Vite exclusively. App compiles and runs.

---

## Group C — Module System

### C01 — Define Module interface in framework (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `framework/core/interfaces.go` or new `framework/core/module.go`

**Context:** All installable modules must implement a common interface. Read `docs/roadmap/02-architecture-evolution.md` section 2.

**What to do:**
1. Read `framework/core/interfaces.go`.
2. Add:
```go
type Module interface {
    ID() string
    Migrations() fs.FS  // embedded migration files; nil if none
}

type RoutableModule interface {
    Module
    RegisterRoutes(r Router) error
}
```
Where `Router` is a minimal interface over Echo group registration (define it here).
3. Additive only — do not change existing interfaces.
4. Run `go build ./...`.

**Done when:** Interfaces defined, project compiles.

---

### C02 — Standardize marker comments in router and container (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `app/router.go`, `app/foundation/container.go`

**Context:** Marker comments are insertion points for `ship module:add`. Some exist already; standardize and extend.

**What to do:**
1. In `app/router.go`, ensure these exist at correct positions (add if missing):
   ```go
   // ship:routes:public:start / ship:routes:public:end
   // ship:routes:auth:start / ship:routes:auth:end
   // ship:routes:external:start / ship:routes:external:end
   ```
2. In `app/foundation/container.go`, add:
   ```go
   // ship:container:start / ship:container:end
   ```
3. Logic unchanged — comment additions only.

**Done when:** All markers exist in both files. `go build ./...` passes.

---

### C03 — Implement `ship module:add` CLI command

**Status:** `[ ] todo`
**Depends on:** C01, C02
**Files:** `tools/cli/ship/internal/commands/module.go` (new), `tools/cli/ship/internal/cli/cli.go`

**Context:** `ship module:add <name>` installs a module by inserting wiring at marker comments in container + router and updating `config/modules.yaml`. Supported initially: `notifications`, `paidsubscriptions`, `emailsubscriptions`, `jobs`, `pwa`, `admin`.

**What to do:**
1. Read an existing generator (e.g., `make:controller`) for the marker-insertion pattern.
2. Create `module.go` with `module:add <name>` subcommand:
   - For each known module: define import, container init line, and route registration line to insert.
   - Insert at `ship:container:start` and `ship:routes:*:start` markers.
   - Update `config/modules.yaml`.
3. Add `--dry-run` flag (prints diff, writes nothing).
4. Register in `cli.go`.

**Done when:** `ship module:add notifications --dry-run` shows correct diff. `ship module:add notifications` correctly wires (verify by reading modified files). `go build ./...` passes.

---

### C04 — Implement `ship module:remove` CLI command

**Status:** `[ ] todo`
**Depends on:** C03
**Files:** `tools/cli/ship/internal/commands/module.go`

**Context:** Reverse of C03. Print a reminder that DB migrations are NOT rolled back automatically.

**Done when:** `ship module:remove notifications` removes wiring. Compile passes. Reminder printed about migrations.

---

### C05 — Add `ship_doctor`, `ship_routes`, `ship_modules` to MCP server (parallel)

**Status:** `[ ] todo`
**Depends on:** A02
**Files:** `tools/mcp/ship/`

**Context:** Expand MCP from 3 read-only tools to include verification and inspection. These enable the LLM act→verify→fix loop. See `docs/roadmap/02-architecture-evolution.md` section 4.

**What to do:**
1. Read `tools/mcp/ship/` fully.
2. Add:
   - `ship_doctor`: runs `ship doctor --json`, returns parsed JSON.
   - `ship_routes`: parses `app/router.go` AST to extract route inventory, returns `[{method, path, auth, handler}]`.
   - `ship_modules`: reads `config/modules.yaml` + scans `modules/`, returns installed + available modules.
3. Each tool: clear description, input/output schema documented.

**Done when:** Three new tools exist, return valid JSON, existing tools unchanged.

---

## Group D — Module Extraction

> All D tasks are parallel with each other. All depend on C01 (module interface must exist).

### D01 — Extract auth controllers into `modules/auth`

**Status:** `[ ] todo`
**Depends on:** C01
**Files:** `app/web/controllers/login.go`, `register.go`, `logout.go`, `forgot_password.go`, new `modules/auth/`

**What to do:**
1. Read all four controllers + their templ views.
2. Create `modules/auth/`: `module.go` (ID: "auth"), `routes.go`, `service.go`, `views/`.
3. Move handler logic and templ views into the module.
4. `app/router.go`: call `authModule.RegisterRoutes(...)` instead of direct registration.
5. Delete original controllers.
6. `go build ./...` + `make test`.

**Done when:** Auth routes work via module. Old controllers deleted. Tests pass.

---

### D02 — Extract profile into `modules/profile`

**Status:** `[ ] todo`
**Depends on:** C01
**Files:** `app/profile/`, `app/web/controllers/profile.go`, `profile_photo.go`, `upload_photo.go`, new `modules/profile/`

**What to do:** Same pattern as D01. Module brings: `service.go` (wraps profile domain logic), `store.go`/`store_sql.go`, `routes.go`, `views/`.

**Done when:** Profile routes work via module. Tests pass.

---

### D03 — Move paidsubscriptions route handler into module

**Status:** `[ ] todo`
**Depends on:** C01
**Files:** `app/web/controllers/payments.go` → `modules/paidsubscriptions/routes.go` (new)

**What to do:** Move handler into module. Implement `RoutableModule`. Update router. Delete old controller.

**Done when:** Payments routes work via module. Old controller deleted.

---

### D04 — Move notifications route handlers into module

**Status:** `[ ] todo`
**Depends on:** C01
**Files:** `app/web/controllers/notifications.go`, `push_notifs.go` → `modules/notifications/routes.go` (new)

**Done when:** Notification routes work via module. Old controllers deleted.

---

### D05 — Create `modules/pwa`

**Status:** `[ ] todo`
**Depends on:** C01
**Files:** `app/web/controllers/install_app.go`, PWA templ components, service worker, manifest → `modules/pwa/`

**What to do:** Create `modules/pwa/` with `module.go` (ID: "pwa"), `routes.go`, `views/`, static assets (manifest template, service worker). Delete originals.

**Done when:** PWA install flow works via module. Old files deleted.

---

## Group E — App Split: Landing vs Starter

### E01 — Create `starter/` minimal skeleton

**Status:** `[ ] todo`
**Depends on:** D01, D02
**Files:** new `starter/` directory

**Context:** Minimal app used by `ship new`. Auth + profile + home feed only. No payments, push, PWA by default.

**What to do:**
1. Create `starter/` mirroring `app/` structure: `router.go`, `foundation/container.go`, `views/web/pages/home_feed.templ`, `views/web/pages/landing.templ`.
2. Include only auth + profile modules.
3. Ensure `go build ./...` from `starter/`.
4. Write `starter/README.md`: "Minimal GoShip starter. Add modules with `ship module:add`."

**Done when:** `starter/` compiles. Contains only auth + profile + home feed.

---

### E02 — Wire `ship new` to use `starter/` as template

**Status:** `[ ] todo`
**Depends on:** E01
**Files:** `tools/cli/ship/internal/commands/new.go`

**What to do:**
1. Read current `ship new` implementation.
2. Update to template from `starter/` (embedded in binary or fetched).
3. Replace placeholder names in generated files.
4. Print post-install: `cd myapp && ship module:add <module> && make run`.

**Done when:** `ship new testapp` generates working minimal app from starter.

---

## Group G — Config: Drop Viper, Adopt cleanenv + `.env`

> This group is high priority. It removes a major pain point and is a prerequisite for single-binary defaults (Group I).

### G01 — Replace Viper with cleanenv struct-tag config

**Status:** `[ ] todo`
**Depends on:** nothing (parallel)
**Files:** `config/config.go`, `go.mod`, all files importing `viper`

**Context:** Viper's multi-source merging creates "too many layers" pain (YAML → env override → Go). Replace with `cleanenv` (`github.com/ilyakaznacheev/cleanenv`) which reads directly from env vars into struct tags. One dependency, one source of truth.

**Chosen library:** `cleanenv` — handles struct tags, .env loading, required validation, defaults, and auto-generates help text. No separate godotenv needed.

**What to do:**
1. Run `grep -rn "viper" .` to find all usages.
2. `go get github.com/ilyakaznacheev/cleanenv`.
3. Rewrite `config/config.go`: convert all config fields to cleanenv struct tags:
   ```go
   type Config struct {
       DatabaseURL  string `env:"DATABASE_URL,required"`
       SecretKey    string `env:"SECRET_KEY,required"`
       Port         int    `env:"PORT" env-default:"8080"`
       SMTPHost     string `env:"SMTP_HOST"`
       RedisURL     string `env:"REDIS_URL"`
       // ...
   }
   ```
4. Replace `config.Load()` / viper init with:
   ```go
   func Load() (*Config, error) {
       cfg := &Config{}
       if err := cleanenv.ReadEnv(cfg); err != nil {
           return nil, err
       }
       return cfg, nil
   }
   ```
5. Remove viper from `go.mod`.
6. Update `app/foundation/container.go` to use new config loader.
7. Run `go build ./...` and `make test`.

**Done when:** Viper is removed from `go.mod`. Config loads from env vars via cleanenv. All tests pass.

---

### G02 — Add `.env` file loading

**Status:** `[ ] todo`
**Depends on:** G01
**Files:** `config/config.go`, `.env.example` (new), `.gitignore`

**Context:** cleanenv supports loading from `.env` files via `cleanenv.ReadConfig(".env", cfg)` before `ReadEnv`. The `.env` file is gitignored; `.env.example` is committed.

**What to do:**
1. Update config loader to:
   ```go
   func Load() (*Config, error) {
       cfg := &Config{}
       _ = cleanenv.ReadConfig(".env", cfg) // load .env if exists, ignore error if not
       if err := cleanenv.ReadEnv(cfg); err != nil {
           return nil, err
       }
       return cfg, nil
   }
   ```
2. Create `.env.example` with every key from the Config struct, empty values, and comments explaining each.
3. Add `.env` to `.gitignore` (it may already be there — verify).
4. Update `docs/guides/02-development-workflows.md`: "Copy `.env.example` to `.env` and fill in values before running locally."

**Done when:** `.env.example` exists with all keys. `config.Load()` reads `.env` if present. `.env` is gitignored.

---

### G03 — Remove YAML config files

**Status:** `[ ] todo`
**Depends on:** G01, G02
**Files:** `config/application.yaml`, `config/environments/`, all code reading YAML config

**Context:** With cleanenv + .env, YAML config is redundant. Non-secret structural config (feature flags, module list) can live in env vars too, or in a minimal `config/modules.yaml` that is committed (not secret).

**What to do:**
1. Identify any config that was YAML-only and has no env var equivalent — add struct tags for those.
2. Delete `config/application.yaml` and `config/environments/` if all values are now in struct tags with defaults.
3. Keep `config/modules.yaml` only if it serves a structural purpose distinct from secrets.
4. Update any `make` targets or docs that reference YAML config files.

**Done when:** No YAML config files for secrets or application settings. All config comes from `.env` + struct tag defaults. `go build` + tests pass.

---

### G04 — Add `ship config:validate` command

**Status:** `[ ] todo`
**Depends on:** G01
**Files:** `tools/cli/ship/internal/commands/config.go` (new)

**Context:** cleanenv can generate a description of all config fields (required/optional, defaults). Expose this as a CLI command and add to `ship doctor`.

**What to do:**
1. Add `ship config:validate` that calls `cleanenv.GetDescription(&Config{}, nil)` and prints the table.
2. Add `--json` flag.
3. Integrate into `ship doctor` check: if any required env var is missing, `ship doctor` reports it as an error.

**Done when:** `ship config:validate` lists all env vars with required/optional status. Missing required vars appear in `ship doctor` output.

---

## Group H — Nil Safety Architecture

> These tasks eliminate the entire class of nil-deref panics. Do H01 and H02 first (cheap wins), then H03–H06 in parallel.

### H01 — Add recovery middleware to Echo (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `app/web/wiring.go` or wherever global middleware is registered

**Context:** Recovery middleware catches panics in any request, logs them with stack trace, and returns a 500 — the app stays alive for all other users.

**What to do:**
1. Read the middleware registration file.
2. Add `e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{ LogErrorFunc: ... }))` as the FIRST middleware (must wrap everything).
3. `LogErrorFunc` should use the existing structured logger to emit the panic + stack trace.
4. Test: introduce a deliberate panic in a test route, verify the app returns 500 and stays running.

**Done when:** App does not crash on panics. Stack trace is logged. Returns 500 to the panicking request only.

---

### H02 — Add `nilaway` to CI and `ship doctor` (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `.github/workflows/` (CI), `tools/cli/ship/internal/commands/` (doctor)

**Context:** `nilaway` (Uber) statically traces nil flows across function boundaries — catches nil derefs before they hit production.

**What to do:**
1. Add to CI:
   ```yaml
   - name: nilaway
     run: go run go.uber.org/nilaway/cmd/nilaway@latest ./...
   ```
2. Add to `ship doctor`: run `nilaway ./...` and parse output for issues. Report as warnings (not errors) initially until existing codebase is clean.
3. Document in `docs/guides/01-ai-agent-guide.md` under "Nil Safety" section.

**Done when:** `nilaway` runs in CI. `ship doctor` surfaces nil issues as warnings.

---

### H03 — Audit and enforce value-type viewmodels

**Status:** `[ ] todo`
**Depends on:** H01 (recovery middleware should be in first)
**Files:** `app/web/viewmodels/`, all templ components

**Context:** The root cause of most nil panics in templ: domain model pointers flowing directly into templates. Viewmodels must be pure value types — no pointer fields.

**Convention:**
- Domain models (`db/gen/`, `framework/domain/`) may have pointers for nullable DB columns.
- Viewmodels (`app/web/viewmodels/`) must have **zero pointer fields**. Use `sql.NullString`, zero values, or custom `Option[T]` for optional data.
- Templ component signatures must accept viewmodel types (or primitives), never `*DomainModel`.
- Controllers own the domain → viewmodel transformation and all nil handling.

**What to do:**
1. Read all files in `app/web/viewmodels/`.
2. For each struct: replace any pointer field (`*string`, `*int`, `*SomeStruct`) with a value equivalent:
   - `*string` → `string` (empty string = absent)
   - `*int` → `int` (zero = absent), or `sql.NullInt64` if you need to distinguish zero from absent
   - `*NestedStruct` → `NestedStruct` (zero value struct)
3. For each templ component that accepts a `*DomainModel` directly: introduce a viewmodel and update the component signature.
4. Update all controllers that feed into those components to do the transformation.
5. Add a note in `docs/guides/01-ai-agent-guide.md` under "Nil Safety" codifying this as a permanent convention.

**Done when:** `grep -rn '\*[A-Z]' app/web/viewmodels/` returns no pointer fields. All affected templ components updated. Tests pass.

---

### H04 — Add nil-safe accessor methods to domain models

**Status:** `[ ] todo`
**Depends on:** nothing (parallel)
**Files:** `framework/domain/`, `db/gen/`

**Context:** For places where domain model pointers genuinely must be used (e.g., loading from DB before transformation), add nil-safe accessor methods. Go methods on nil pointer receivers are legal if they guard immediately.

**What to do:**
1. For every domain model struct that has pointer fields, add accessor methods:
   ```go
   func (u *User) DisplayName() string {
       if u == nil { return "" }
       if u.Name == nil { return "" }
       return *u.Name
   }
   ```
2. Add a shared helper in `framework/`:
   ```go
   func StringOr(s *string, def string) string {
       if s == nil { return def }
       return *s
   }
   ```
3. Replace all `*s` dereferences outside of viewmodel transformers with these safe accessors.

**Done when:** No bare `*ptr` dereferences exist outside of viewmodel transformer functions. `nilaway` passes cleanly on domain model files.

---

### H05 — Viewmodel constructor functions

**Status:** `[ ] todo`
**Depends on:** H03
**Files:** `app/web/viewmodels/`

**Context:** Viewmodels should always be initialized via constructors that guarantee all fields are set. This prevents "forgot to set a field" nil panics.

**What to do:**
1. For each viewmodel struct in `app/web/viewmodels/`, add a constructor:
   ```go
   func NewHomeFeedData(user User, items []FeedItem) HomeFeedData {
       if items == nil { items = []FeedItem{} }
       return HomeFeedData{User: user, Items: items}
   }
   ```
2. Update all controllers to use constructors instead of struct literals.
3. Convention: viewmodel struct literals (`HomeFeedData{...}`) are only allowed inside their own constructor. Everywhere else must use `NewHomeFeedData(...)`.

**Done when:** Every viewmodel has a constructor. Controllers use constructors. `go build` + tests pass.

---

### H06 — Route smoke tests for nil deref

**Status:** `[ ] todo`
**Depends on:** H03, H05
**Files:** `app/web/controllers/*_test.go`

**Context:** Each route test with zero-value / minimal data is a nil deref smoke test. If a template tries to dereference a nil, the test catches it before production.

**What to do:**
1. For every controller that does not already have a route test: add a minimal test that calls the route with zero-value data and asserts HTTP 200.
2. For existing tests: verify they pass zero-value viewmodels (not maximal/happy-path data only).
3. Follow the existing goquery test pattern in `app/web/controllers/*_test.go`.

**Done when:** Every public-facing route has at least one route test with minimal data. All tests pass.

---

## Group I — Single Binary Mode

> These four tasks unlock zero-dependency deployment. Do them together as a unit.

### I01 — Add SQLite DB adapter (CGO-free)

**Status:** `[ ] todo`
**Depends on:** G01 (config must be cleanenv-based to add `DB_DRIVER` env var cleanly)
**Files:** `app/foundation/container.go`, `go.mod`, new `framework/repos/db/sqlite.go`

**Context:** Use `modernc.org/sqlite` (pure Go, CGO-free — cross-compilation works) NOT `go-sqlite3` (requires CGO). Goose supports SQLite dialect. Bob supports SQLite.

**What to do:**
1. `go get modernc.org/sqlite`.
2. Add `DB_DRIVER` env var to Config struct (values: `postgres`, `sqlite`; default: `sqlite` for new projects, existing config keeps `postgres`).
3. In `app/foundation/container.go` DB init: switch on `c.Config.DBDriver`:
   - `sqlite`: open `modernc.org/sqlite` driver, connect to `./dbs/app.db` (path configurable via `DB_PATH` env var).
   - `postgres`: existing Postgres connection (unchanged).
4. Ensure Goose migration runner uses the correct dialect.
5. Ensure Bob query generation works against SQLite (may need a separate bobgen config).
6. Test: `DB_DRIVER=sqlite make dev` starts app with SQLite.

**Done when:** App boots with `DB_DRIVER=sqlite`. Migrations run. Basic CRUD works. No CGO required.

---

### I02 — Add Backlite as SQLite-backed jobs driver

**Status:** `[ ] todo`
**Depends on:** I01 (needs SQLite DB to be working)
**Files:** `modules/jobs/drivers/backlite/` (new), `config/config.go`, `app/foundation/container.go`

**Context:** Backlite (`github.com/mikestefanello/backlite`) uses SQLite as a job queue — same DB file, no Redis needed. Implements the existing `core.Jobs` interface.

**What to do:**
1. `go get github.com/mikestefanello/backlite`.
2. Create `modules/jobs/drivers/backlite/driver.go` implementing `core.Jobs` using Backlite's client.
3. Add `JOBS_DRIVER` env var to Config (values: `backlite`, `asynq`; default: `backlite`).
4. In `app/foundation/container.go` jobs init: switch on `JOBS_DRIVER`:
   - `backlite`: init Backlite client with the existing SQLite DB connection.
   - `asynq`: existing Asynq setup (unchanged).
5. Start Backlite dispatcher in `cmd/web/main.go` when jobs driver is Backlite (runs in-process, no separate worker needed).
6. Test: `JOBS_DRIVER=backlite make dev` — enqueue a test job, verify it executes.

**Done when:** Jobs work with `JOBS_DRIVER=backlite`. No Redis required. Backlite dispatcher runs in-process with the web server.

---

### I03 — Add Otter as in-memory cache adapter

**Status:** `[ ] todo`
**Depends on:** nothing (parallel with I01)
**Files:** `app/foundation/container.go`, new `framework/repos/cache/otter.go`, `go.mod`

**Context:** Otter (`github.com/maypok86/otter`) is a lockless in-memory cache (S3-FIFO eviction, very high throughput). Implements the existing `core.Cache` interface. Valid only for single-process deployment; use Redis for multi-process.

**What to do:**
1. `go get github.com/maypok86/otter`.
2. Create `framework/repos/cache/otter.go` implementing `core.Cache` with Otter as the backend. Support key/group/tag/expiration semantics matching the existing interface. Add the chainable builder API (see M04 section 1.3).
3. Add `CACHE_DRIVER` env var to Config (values: `otter`, `redis`; default: `otter`).
4. In container cache init: switch on `CACHE_DRIVER`.
5. Test: cache set/get/flush works with `CACHE_DRIVER=otter`.

**Done when:** `CACHE_DRIVER=otter` works. No Redis required for cache. Chainable builder API exposed.

---

### I04 — Wire single-binary mode as default + update docs

**Status:** `[ ] todo`
**Depends on:** I01, I02, I03
**Files:** `.env.example`, `Makefile`, `README.md`, `docs/guides/02-development-workflows.md`

**Context:** Make single-binary mode the default for new projects. `make run` should work with zero Docker.

**What to do:**
1. Set defaults in Config struct: `DB_DRIVER=sqlite`, `CACHE_DRIVER=otter`, `JOBS_DRIVER=backlite`.
2. Update `.env.example` to reflect these defaults.
3. Add `make run` target: no Docker, no infra, just `go run ./cmd/web`. Succeeds with single-binary defaults.
4. Update `README.md` Requirements section: remove Docker as hard requirement ("Docker required for Postgres/Redis; not needed for single-binary SQLite mode").
5. Update `docs/guides/02-development-workflows.md`: document single-binary vs standard modes.

**Done when:** `cp .env.example .env && make run` starts a working app with no Docker. Docs reflect two modes.

---

### I05 — In-memory test database (zero Docker for tests)

**Status:** `[ ] todo`
**Depends on:** I01
**Files:** `config/config.go`, `app/foundation/container.go`, all test files using DB

**Context:** When `APP_ENV=test`, the container should auto-connect to an in-memory SQLite DB and run migrations. Tests run instantly with no Docker. Integration tests (testing Postgres-specific behavior) remain Docker-based but are clearly separated.

**What to do:**
1. Add `APP_ENV` env var to Config (values: `development`, `test`, `production`).
2. In container DB init: if `APP_ENV=test`, use SQLite in-memory (`file::memory:?cache=shared&mode=memory`), run migrations.
3. Add `config.SwitchEnvironment(config.EnvTest)` helper (set `APP_ENV=test` before container init).
4. In all `TestMain` functions: call `config.SwitchEnvironment(config.EnvTest)` before `services.NewContainer()`.
5. Tag existing Docker-dependent tests as `//go:build integration` so `make test` skips them; `make test-integration` includes them.

**Done when:** `make test` passes with no Docker running. In-memory DB is used. Integration tests still work with Docker via `make test-integration`.

---

## Group J — Admin Panel Module

### J01 — Define AdminField and AdminResource type system (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `modules/admin/types.go` (new)

**Context:** The admin panel is reflection-based + Bob-backed. No Ent required. Modules register Go structs; the admin module uses `reflect` to discover fields, types, and tags, then describes them as `AdminField` slices. Templ components receive these slices and render the appropriate UI. Read `docs/roadmap/04-pagoda-and-dx-improvements.md` section 1.4.

**What to do:**
Create `modules/admin/types.go` with:
```go
type FieldType string
const (
    FieldTypeString   FieldType = "string"
    FieldTypeInt      FieldType = "int"
    FieldTypeBool     FieldType = "bool"
    FieldTypeTime     FieldType = "time"
    FieldTypeText     FieldType = "text"     // multiline
    FieldTypeEmail    FieldType = "email"
    FieldTypePassword FieldType = "password" // omit from list, hide in form
    FieldTypeReadOnly FieldType = "readonly"
)

type AdminField struct {
    Name        string
    Label       string    // human-readable, derived from field name
    Type        FieldType
    Value       any       // current value for forms
    Required    bool
    Sensitive   bool      // omit from list view
}

type AdminResource struct {
    Name       string     // e.g., "Post"
    PluralName string     // e.g., "Posts"
    TableName  string     // DB table name
    Fields     []AdminField
    IDField    string     // which field is the PK
}

type AdminRow map[string]any  // one row from DB list
```

**Done when:** Types file compiles. No other code changes yet.

---

### J02 — Implement reflection-based resource registration

**Status:** `[ ] todo`
**Depends on:** J01
**Files:** `modules/admin/registry.go` (new)

**Context:** `admin.Register[T]()` uses Go generics + reflection to inspect the struct type `T` and produce an `AdminResource` describing it.

**What to do:**
Create `modules/admin/registry.go`:
```go
var registry = map[string]AdminResource{}

type ResourceConfig struct {
    TableName   string
    ListFields  []string  // which fields appear in list view; empty = all non-sensitive
    ReadOnly    []string  // fields shown but not editable
    Sensitive   []string  // fields omitted from list, input type=password in form
}

func Register[T any](cfg ResourceConfig) {
    t := reflect.TypeOf(*new(T))
    // introspect t.Fields()
    // derive FieldType from field Kind + tags
    // build AdminResource and store in registry
}
```

Field type derivation rules:
- `string` → `FieldTypeString` (or `FieldTypeEmail` if tag `admin:"email"`, `FieldTypeText` if tag `admin:"text"`)
- `bool` → `FieldTypeBool`
- `int`, `int64` etc → `FieldTypeInt`
- `time.Time` → `FieldTypeTime`
- Field in `Sensitive` list → `FieldTypePassword`
- Field in `ReadOnly` list → `FieldTypeReadOnly`

**Done when:** `admin.Register[Post](cfg)` populates registry with correct `AdminResource`. Verified by unit test.

---

### J03 — Implement Bob-backed CRUD operations for admin

**Status:** `[ ] todo`
**Depends on:** J02, I01 (SQLite must work if testing with SQLite)
**Files:** `modules/admin/store.go` (new)

**Context:** The admin module must list, get, create, update, and delete records for any registered resource using Bob for type-safe SQL. Since the resource type is dynamic, use raw SQL with `database/sql` fallback for admin operations (Bob is used for app code; admin is introspection territory).

**What to do:**
1. Implement:
   ```go
   func List(ctx context.Context, db *sql.DB, res AdminResource, page, perPage int) ([]AdminRow, int, error)
   func Get(ctx context.Context, db *sql.DB, res AdminResource, id any) (AdminRow, error)
   func Create(ctx context.Context, db *sql.DB, res AdminResource, values map[string]any) error
   func Update(ctx context.Context, db *sql.DB, res AdminResource, id any, values map[string]any) error
   func Delete(ctx context.Context, db *sql.DB, res AdminResource, id any) error
   ```
2. Use parameterized queries (`?` for SQLite, `$1` for Postgres) — detect dialect from driver name.
3. `List` returns rows as `[]AdminRow` (map[string]any) and total count for pagination.

**Done when:** All 5 operations work against a test SQLite DB. Unit tests cover each.

---

### J04 — Build templ components for admin UI

**Status:** `[ ] todo`
**Depends on:** J01
**Files:** `modules/admin/views/web/` (new templ files)

**Context:** Templ components are **data-driven** — they receive `AdminResource` and `[]AdminField` at runtime and render the appropriate UI. The dynamic behavior is in the *data*, not in runtime template generation. A `switch` on `AdminField.Type` renders the correct input. This is fully compatible with templ's compiled approach.

**What to do:**
Create these templ components:

1. `admin_layout.templ` — admin shell: sidebar with resource links, main content area.
   ```templ
   // Renders: full-page admin shell with left sidebar listing all registered resources and top bar with "Admin" title
   templ AdminLayout(resources []AdminResource, content templ.Component) { ... }
   ```

2. `admin_list.templ` — list table for a resource.
   ```templ
   // Renders: paginated table of resource rows with column headers, edit/delete links per row, and an "Add new" button
   templ AdminList(res AdminResource, rows []AdminRow, pager Pager) { ... }
   ```

3. `admin_form.templ` — create/edit form.
   ```templ
   // Renders: create/edit form with one input per AdminField, type-appropriate input widget per field type
   templ AdminForm(res AdminResource, values map[string]any, errs map[string]string, csrfToken string) { ... }
   ```

4. `admin_field_input.templ` — single field input, switches on FieldType.
   ```templ
   // Renders: appropriate HTML input for the given field type (text, checkbox, number, datetime-local, textarea, password)
   templ AdminFieldInput(field AdminField) {
       switch field.Type {
       case FieldTypeString: <input type="text" ...>
       case FieldTypeBool:   <input type="checkbox" ...>
       case FieldTypeInt:    <input type="number" ...>
       case FieldTypeTime:   <input type="datetime-local" ...>
       case FieldTypeText:   <textarea ...></textarea>
       case FieldTypePassword: <input type="password" ...>
       case FieldTypeReadOnly: <input type="text" disabled ...>
       }
   }
   ```

5. `admin_delete_confirm.templ` — SweetAlert2 delete confirmation, or inline form.

Run `make templ-gen` after.

**Done when:** All 5 templ files exist and compile. `make templ-gen` succeeds.

---

### J05 — Wire admin routes

**Status:** `[ ] todo`
**Depends on:** J02, J03, J04
**Files:** `modules/admin/routes.go` (new), `modules/admin/module.go` (new)

**Context:** Admin routes are automatically generated for every registered resource. Protected by `middleware.RequireAdmin`.

**What to do:**
1. Create `modules/admin/module.go`:
   ```go
   func New() *AdminModule { ... }
   func (m *AdminModule) ID() string { return "admin" }
   func (m *AdminModule) Migrations() fs.FS { return nil }
   func (m *AdminModule) RegisterRoutes(r Router) error { ... }
   ```
2. Create `modules/admin/routes.go`. For each registered resource, register:
   ```
   GET    /admin/{resource}          → List handler
   GET    /admin/{resource}/new      → New form
   POST   /admin/{resource}          → Create handler
   GET    /admin/{resource}/{id}     → Edit form
   PUT    /admin/{resource}/{id}     → Update handler
   DELETE /admin/{resource}/{id}     → Delete handler
   ```
3. All admin routes wrapped in `middleware.RequireAdmin`.
4. Add link to admin in main nav (conditionally, if user is admin).

**Done when:** Visiting `/admin/posts` (assuming Post is registered) renders the list. CRUD works end-to-end. Non-admin users get 403.

---

### J06 — Embed Backlite queue monitor in admin panel (parallel after J05, I02)

**Status:** `[ ] todo`
**Depends on:** J05, I02
**Files:** `modules/admin/routes.go`

**Context:** Backlite provides an HTTP handler for monitoring queues. Embed it at `/admin/queues`.

**What to do:**
1. Read Backlite docs for the embedded monitor handler.
2. Mount Backlite's handler at `/admin/queues` in admin routes.
3. Add "Queue Monitor" link to admin sidebar.

**Done when:** `/admin/queues` shows task queue monitor when `JOBS_DRIVER=backlite`.

---

## Group K — DX Improvements

### K01 — Chainable redirect helper (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `framework/redirect/redirect.go` (new)

**Context:** Replace manual redirect calls with a chainable builder. Automatically handles HTMX redirects (`HX-Redirect` header) for boosted requests.

**What to do:**
```go
// Usage:
return redirect.New(ctx).Route("user_profile").Params(userID).Query(q).Go()

// Implementation:
type Redirect struct { ctx echo.Context; route string; params []any; query url.Values }
func New(ctx echo.Context) *Redirect
func (r *Redirect) Route(name string) *Redirect
func (r *Redirect) Params(params ...any) *Redirect
func (r *Redirect) Query(q url.Values) *Redirect
func (r *Redirect) Go() error  // detects HX-Request header, sets HX-Redirect if HTMX
```

**Done when:** `redirect.New(ctx).Route("home_feed").Go()` works in a controller. HTMX requests get `HX-Redirect` header. Non-HTMX requests get 302.

---

### K02 — Pagination utility (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `framework/pager/pager.go` (new), new templ component in `app/views/web/components/pager.templ`

**Context:** Standardize cursor/offset pagination. Controller gets a `Pager`, passes it to viewmodel, templ component renders prev/next links.

**What to do:**
1. Create `framework/pager/pager.go`:
   ```go
   type Pager struct { Page, PerPage, Total int }
   func New(ctx echo.Context, perPage int) Pager  // reads ?page= from query
   func (p Pager) Offset() int
   func (p Pager) Limit() int
   func (p Pager) HasNext() bool
   func (p Pager) HasPrev() bool
   func (p Pager) TotalPages() int
   ```
2. Create `app/views/web/components/pager.templ`:
   ```templ
   // Renders: prev/next pagination bar with page number and total pages indicator
   templ Pagination(p pager.Pager, baseURL string) { ... }
   ```

**Done when:** Controller can call `pager.New(ctx, 20)`, pass pager to viewmodel, and render `Pagination` component. Unit tests for offset/limit/HasNext/HasPrev.

---

### K03 — `ship routes` command (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `tools/cli/ship/internal/commands/routes.go` (new)

**Context:** Print a table of all registered routes. Inspect `app/router.go` via AST parsing. Also expose as MCP tool.

**What to do:**
1. Parse `app/router.go` AST to extract route registrations (method, path, handler, auth level).
2. Print as table:
   ```
   METHOD  PATH                      AUTH    HANDLER
   GET     /                         public  landing.Get
   POST    /user/register            public  register.Post
   GET     /auth/homeFeed            auth    home_feed.Get
   ```
3. Add `--json` flag.
4. Integrate as `ship_routes` MCP tool (see C05).

**Done when:** `ship routes` prints route table. `ship routes --json` outputs JSON array.

---

### K04 — `ship db:console` command (parallel)

**Status:** `[ ] todo`
**Depends on:** G01 (needs cleanenv config to read DB URL)
**Files:** `tools/cli/ship/internal/commands/db.go`

**Context:** Opens a raw DB shell. Reads active DB config and spawns `psql`, `mysql`, or `sqlite3` with the correct connection string.

**What to do:**
1. Read active `DB_DRIVER` from config.
2. Spawn the appropriate shell with the connection string from config.
3. Pass through stdin/stdout/stderr to the terminal.

**Done when:** `ship db:console` drops into an interactive DB shell.

---

### K05 — Built-in rate limiter middleware (parallel)

**Status:** `[ ] todo`
**Depends on:** I03 (Otter for in-memory rate limit state; Redis if scaled)
**Files:** `app/web/middleware/rate_limit.go` (new), `framework/repos/ratelimit/` (new)

**Context:** Per-IP and per-user rate limiting with configurable limits per route group.

**What to do:**
1. Create `framework/repos/ratelimit/ratelimit.go` with an interface backed by Otter (in-memory) or Redis.
2. Create `app/web/middleware/rate_limit.go` Echo middleware factory:
   ```go
   func RateLimit(store ratelimit.Store, max int, window time.Duration) echo.MiddlewareFunc
   ```
3. Apply to auth routes (e.g., 10 req/min on `/user/login`).
4. Returns 429 with `Retry-After` header on exceed.

**Done when:** Auth routes return 429 after exceeding the limit. Test covers this.

---

### K06 — Afero file system abstraction (parallel)

**Status:** `[ ] todo`
**Depends on:** G01 (needs `STORAGE_DRIVER` env var)
**Files:** `framework/repos/storage/`, `app/foundation/container.go`

**Context:** Replace MinIO-only storage with afero abstraction. `STORAGE_DRIVER=local` for dev/single-binary; `STORAGE_DRIVER=minio` for production.

**What to do:**
1. `go get github.com/spf13/afero`.
2. Add `STORAGE_DRIVER` env var (values: `local`, `minio`).
3. Wrap afero behind the existing `framework/core` storage interface (or create one).
4. `local`: afero `OsFs` rooted at `./uploads` (path configurable).
5. Tests: automatically use afero `MemMapFs` when `APP_ENV=test`.
6. Keep MinIO backend for production compatibility.

**Done when:** File uploads work with `STORAGE_DRIVER=local`. Tests use in-memory FS.

---

## Group F — Documentation

### F01 — Fix README inconsistencies (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing

**What to do:**
1. Read `README.md`.
2. Fix `pkg/` → `framework/` in the Repository Shape section.
3. Fix `pkg/repos/storage/storagerepo.go` reference to correct path.
4. Update Requirements: remove Docker as hard requirement; note it's only needed for Postgres/Redis mode.
5. Add brief description of single-binary mode once Group I tasks are done, or add a TODO note.

**Done when:** README has no stale `pkg/` references. Docker requirement is accurately described.

---

### F02 — Fix architecture doc: decouple from Asynq (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing (fix the doc now; implementation follows in Group I/C)

**What to do:**
1. Read `docs/architecture/01-architecture.md`.
2. Update Worker Runtime Flow section: replace hardcoded Asynq description with "jobs adapter — currently Asynq (Redis-backed); Backlite (SQLite-backed) supported for single-binary mode".
3. Update "Asynq handles background jobs" line at bottom to reflect adapter abstraction.

**Done when:** Architecture doc does not assume Asynq specifically. References adapter pattern.

---

### F03 — Update AI agent guide: add nil safety convention (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing

**What to do:**
Add a "Nil Safety" section to `docs/guides/01-ai-agent-guide.md`:
- Viewmodels must have zero pointer fields (value types only).
- Templ components accept viewmodel types, never `*DomainModel`.
- Controllers own domain → viewmodel transformation and all nil handling.
- `nilaway` runs in CI — new code must pass it.
- Recovery middleware is registered globally — panics return 500 but app stays up.

**Done when:** Section exists in the guide.

---

### F04 — Update docs index with all new roadmap docs (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing

**What to do:** Read `docs/00-index.md`. Verify M01–M04 are all listed. Add any missing entries.

**Done when:** Index references all four roadmap documents.

---

### F05 — Update workflows doc: config and single binary mode (parallel)

**Status:** `[ ] todo`
**Depends on:** G01, G02, I04

**What to do:**
1. Read `docs/guides/02-development-workflows.md`.
2. Add "Configuration" section: "Copy `.env.example` to `.env`. All config comes from env vars. No YAML for secrets."
3. Add "Single Binary Mode" section: "Set `DB_DRIVER=sqlite`, `CACHE_DRIVER=otter`, `JOBS_DRIVER=backlite` in `.env`. Run `make run`. No Docker needed."
4. Update Services and Infra section to clarify Redis/Postgres are optional.

**Done when:** Workflows doc accurately describes both single-binary and standard modes.

---

### F06 — Update scope analysis doc to reflect evolving architecture (parallel)

**Status:** `[ ] todo`
**Depends on:** nothing

**What to do:**
1. Read `docs/architecture/03-project-scope-analysis.md`.
2. Remove Viper reference (line ~121).
3. Update background task section to mention Backlite as an option.
4. Add entry for admin module once J01–J05 are planned.

**Done when:** Scope analysis doc has no Viper references. Reflects adapter-based jobs and planned admin module.

---

## Completion Tracker

```
Group A — Critical Fixes
[ ] A01  Container init bug
[ ] A02  ship doctor --json

Group B — JS Islands
[ ] B01  Vite config
[ ] B02  Islands runtime
[ ] B03  ThemeToggle migrated
[ ] B04  All components migrated
[ ] B05  Old esbuild removed

Group C — Module System
[ ] C01  Module interface
[ ] C02  Marker comments
[ ] C03  ship module:add
[ ] C04  ship module:remove
[ ] C05  MCP tools

Group D — Module Extraction (parallel after C01)
[ ] D01  modules/auth
[ ] D02  modules/profile
[ ] D03  modules/paidsubscriptions routes
[ ] D04  modules/notifications routes
[ ] D05  modules/pwa

Group E — App Split
[ ] E01  starter/ skeleton
[ ] E02  ship new uses starter/

Group F — Documentation (mostly parallel)
[ ] F01  README fix
[ ] F02  Architecture doc fix
[ ] F03  Agent guide: nil safety
[ ] F04  Docs index
[ ] F05  Workflows: config + single binary
[ ] F06  Scope analysis cleanup

Group G — Config: cleanenv + .env
[ ] G01  Replace Viper with cleanenv
[ ] G02  .env file loading
[ ] G03  Remove YAML config files
[ ] G04  ship config:validate command

Group H — Nil Safety
[ ] H01  Recovery middleware
[ ] H02  nilaway in CI + ship doctor
[ ] H03  Audit viewmodels (no pointer fields)
[ ] H04  Nil-safe domain model accessors
[ ] H05  Viewmodel constructors
[ ] H06  Route smoke tests

Group I — Single Binary Mode
[ ] I01  SQLite DB adapter (modernc, CGO-free)
[ ] I02  Backlite jobs driver
[ ] I03  Otter cache adapter
[ ] I04  Single-binary default + docs
[ ] I05  In-memory test DB

Group J — Admin Panel
[ ] J01  AdminField/AdminResource types
[ ] J02  Reflection-based registration
[ ] J03  Bob-backed CRUD operations
[ ] J04  Templ components (list, form, field input)
[ ] J05  Wire admin routes
[ ] J06  Backlite queue monitor in admin

Group K — DX Improvements (all parallel)
[ ] K01  Chainable redirect helper
[ ] K02  Pagination utility
[ ] K03  ship routes command
[ ] K04  ship db:console command
[ ] K05  Rate limiter middleware
[ ] K06  Afero file system abstraction
```

---

## Recommended Execution Order

Tasks are ordered by dependency. Tasks with no shared dependencies can run in parallel.

**Layer 1 — No dependencies, start immediately (all parallel):**
A01, A02, H01, H02, G01, F01, F02, F03, F04, C01, C02, J01, K01, K02, K03, K04

**Layer 2 — Depends on Layer 1 tasks (parallel within layer):**
- G02 → needs G01
- G03 → needs G01, G02
- G04 → needs G01
- H03 → needs H01 (recovery in place first)
- H04 → no dependency (can move to Layer 1)
- H05 → needs H03
- I01 → needs G01
- I03 → no hard dependency (can move to Layer 1)
- B01, B02 → no dependency (can move to Layer 1)
- J02 → needs J01
- F05 → needs G01, G02, I04
- F06 → no dependency (can move to Layer 1)

**Layer 3 — Depends on Layer 2:**
- I02 → needs I01 (SQLite adapter)
- I04 → needs I01, I02, I03
- I05 → needs I01
- H06 → needs H03, H05
- B03 → needs B01, B02
- C03 → needs C01, C02
- J03 → needs J02
- J04 → needs J01
- K05 → needs I03 (Otter)
- K06 → needs G01

**Layer 4 — Depends on Layer 3:**
- B04 → needs B03 (proven pattern)
- C04 → needs C03
- C05 → needs A02
- D01–D05 → needs C01 (module interface)
- J05 → needs J02, J03, J04
- E01 → needs D01, D02

**Layer 5 — Final cleanup:**
- B05 → needs B04 (all components migrated)
- J06 → needs J05, I02
- E02 → needs E01
