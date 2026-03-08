# GoShip Atomic Task List

**Reference:** `docs/roadmap/02-architecture-evolution.md` — read this first for context on *why*.

Tasks are grouped by area. Within each group, items are ordered by dependency.
Mark `[x]` when done before starting dependents.
Tasks marked `(parallel)` can run concurrently with other parallel tasks in the same group.

---

## Group A — Critical Bug Fixes (do first, unblocks everything)

### A01 — Fix container initialization for optional services

**Status:** `[ ] todo`
**Files:** `app/foundation/container.go`
**Depends on:** nothing
**Context:**
`initCache()`, `initNotifier()`, `initTasks()` are commented out in `NewContainer()`. Shutdown code calls `.Close()` on potentially nil fields. This is a latent nil-pointer panic.

**What to do:**
1. Read `app/foundation/container.go` fully.
2. Uncomment `c.initCache()`, `c.initNotifier()`, `c.initTasks()` — but wrap each in a config guard:
   ```go
   if c.Config.Cache.Enabled { c.initCache() }
   ```
3. For each guarded service, update the shutdown logic to nil-check before calling `.Close()`:
   ```go
   if c.Cache != nil { c.Cache.Close() }
   ```
4. Verify `Config` struct has appropriate `Enabled` booleans; add them if not.
5. Run `go build ./...` and `make test-unit`.

**Done when:** No commented-out init calls remain. Shutdown is nil-safe. `go build ./...` passes.

---

### A02 — Add `--json` output flag to `ship doctor`

**Status:** `[ ] todo`
**Files:** `tools/cli/ship/internal/commands/doctor.go` (or equivalent)
**Depends on:** nothing
**Context:**
`ship doctor` currently outputs human-readable text. For MCP integration and LLM self-validation, it needs a machine-readable JSON mode.

**What to do:**
1. Read the doctor command implementation.
2. Add a `--json` flag.
3. When `--json` is set, output:
   ```json
   {"ok": true, "issues": [{"type": "string", "file": "string", "detail": "string", "severity": "error|warning"}]}
   ```
4. Exit code: 0 if no errors (warnings allowed), 1 if any errors.
5. Existing non-JSON output must be unchanged when flag is absent.

**Done when:** `ship doctor --json` outputs valid JSON matching the schema above. Existing output unchanged.

---

## Group B — JS Islands Architecture

### B01 — Set up Vite build config with island splitting (parallel)

**Status:** `[ ] todo`
**Files:** `frontend/vite.config.ts` (new), `frontend/package.json`
**Depends on:** nothing
**Context:**
Replace esbuild with Vite. Vite splits each island in `frontend/islands/` into its own chunk. Read `docs/roadmap/02-architecture-evolution.md` section 1 for full design.

**What to do:**
1. Read `frontend/build.mjs`, `frontend/package.json`, `frontend/tailwind.config.js`.
2. Add Vite and plugins to `package.json` devDependencies:
   - `vite`, `@sveltejs/vite-plugin-svelte`, `@vitejs/plugin-react` (optional, for future), `vite-plugin-tailwindcss` or use postcss
3. Create `frontend/vite.config.ts`:
   - Entry point: glob all files in `frontend/islands/` as separate chunks
   - Output: `app/static/islands/[name]-[hash].js`
   - Also output: `app/static/islands-manifest.json` mapping island name → hashed URL
   - Also build: `frontend/javascript/vanilla/main.js` → `app/static/vanilla_bundle.js`
   - Tailwind via postcss config (reuse existing `tailwind.config.js`)
4. Create `frontend/islands/` directory with a `.gitkeep`.
5. Update `Makefile` targets: replace esbuild build commands with `vite build` and `vite` for dev.
6. Keep old `build.mjs` until all Svelte components are migrated (Group B03+).

**Done when:** `make js-build` runs Vite successfully. Output files appear in `app/static/`. Old esbuild build still works via a separate target.

---

### B02 — Write the islands runtime (parallel)

**Status:** `[ ] todo`
**Files:** `frontend/javascript/vanilla/islands-runtime.js` (new), `app/views/web/components/core.templ`
**Depends on:** nothing (can write runtime before Vite is set up)
**Context:**
Small vanilla JS script that discovers `[data-island]` elements, fetches the manifest, dynamically imports island chunks, and calls their `mount(el, props)`. Must re-run after HTMX swaps.

**What to do:**
1. Create `frontend/javascript/vanilla/islands-runtime.js`:
   ```js
   // Fetches /static/islands-manifest.json once, caches it
   // On DOMContentLoaded and after htmx:afterSettle:
   //   querySelectorAll('[data-island]:not([data-island-mounted])')
   //   For each: mark data-island-mounted="true", import chunk, call mount(el, props)
   // props = JSON.parse(el.dataset.props || '{}')
   // Log warning if island name not in manifest
   ```
2. Include `islands-runtime.js` in `app/views/web/components/core.templ` JS block (after HTMX loads).
3. Do NOT remove existing `svelte_bundle.js` include yet — both coexist during migration.

**Done when:** Script exists. When included in a page with a `[data-island]` element, it correctly imports and mounts the island. Can be tested manually with a simple test island.

---

### B03 — Create island adapter convention + migrate ThemeToggle (parallel)

**Status:** `[ ] todo`
**Files:** `frontend/islands/ThemeToggle.svelte` (new), `app/views/web/components/theme_toggle.templ`
**Depends on:** B01, B02
**Context:**
First real island migration. ThemeToggle is a good candidate — small, self-contained.
The island convention: each island file exports `mount(el, props)`. For Svelte:
```js
export function mount(el, props) {
  new MyComponent({ target: el, props })
}
```

**What to do:**
1. Read the current Svelte `ThemeToggle` component and its templ mounting code.
2. Create `frontend/islands/ThemeToggle.svelte` with the component logic + the `mount` export.
3. Update `app/views/web/components/theme_toggle.templ`:
   - Replace `<div id={id}>` + `@initThemeToggle(id)` script block
   - With `<div data-island="ThemeToggle" data-props={ templ.JSONString(props) }></div>`
4. Delete the old templ `script` block for this component.
5. Run `make templ` to regenerate Go files.
6. Test in dev: theme toggle should still work.

**Done when:** ThemeToggle works via the islands runtime. Old `renderSvelteComponent('ThemeToggle', ...)` call is gone.

---

### B04 — Migrate remaining Svelte components to islands

**Status:** `[ ] todo`
**Files:** All Svelte components in `frontend/javascript/svelte/`, corresponding templ files
**Depends on:** B03 (use ThemeToggle as the proven pattern)
**Context:**
Migrate each remaining Svelte component one at a time. Current components (from `main.js` registry):
`MultiSelectComponent`, `PhotoUploader`, `SingleSelect`, `PhoneNumberPicker`, `PwaInstallButton`, `PwaSubscribePush`, `NotificationPermissions`.

**What to do:**
For each component:
1. Read the Svelte source + the templ mounting code.
2. Create `frontend/islands/{ComponentName}.svelte` with `mount` export.
3. Update the templ file to use `data-island` pattern.
4. Remove the old `script` block from templ.
5. Test manually or via Playwright.

Do components one by one. Each is a separate commit.

**Done when:** All components migrated. `window.renderSvelteComponent` is no longer called anywhere. `svelte_bundle.js` is no longer included in `core.templ`.

---

### B05 — Remove old esbuild setup

**Status:** `[ ] todo`
**Files:** `frontend/build.mjs`, `frontend/javascript/svelte/main.js`, `Makefile`
**Depends on:** B04 (all components migrated)

**What to do:**
1. Delete `frontend/build.mjs`.
2. Delete `frontend/javascript/svelte/main.js` (registry file).
3. Remove esbuild from `package.json` devDependencies.
4. Update all `Makefile` targets that referenced old build commands.
5. Run `make js-build` and confirm clean build.

**Done when:** No esbuild references remain. `make js-build` uses Vite exclusively.

---

## Group C — Module System

### C01 — Define the Module interface in framework (parallel)

**Status:** `[ ] todo`
**Files:** `framework/core/interfaces.go` or new `framework/core/module.go`
**Depends on:** nothing
**Context:**
Add a `Module` interface to the framework that all installable modules must implement. Read `docs/roadmap/02-architecture-evolution.md` section 2 for the design.

**What to do:**
1. Read `framework/core/interfaces.go`.
2. Add to the framework (new file or existing):
   ```go
   type Module interface {
       ID() string
       Migrations() fs.FS  // embedded migration files, nil if none
   }

   type RoutableModule interface {
       Module
       RegisterRoutes(r Router) error
   }
   ```
   Where `Router` is a minimal interface wrapping Echo group registration.
3. Do NOT change any existing modules yet — this is additive.
4. Run `go build ./...`.

**Done when:** Interface defined, project compiles.

---

### C02 — Add marker comments to `app/router.go` and `app/foundation/container.go` (parallel)

**Status:** `[ ] todo`
**Files:** `app/router.go`, `app/foundation/container.go`
**Depends on:** nothing
**Context:**
Marker comments are the insertion points for `ship module:add`. They already exist for routes but need to be standardized and extended.

**What to do:**
1. Read both files.
2. In `app/router.go`, ensure these markers exist at correct insertion points:
   ```go
   // ship:routes:public:start
   // ship:routes:public:end
   // ship:routes:auth:start
   // ship:routes:auth:end
   // ship:routes:external:start
   // ship:routes:external:end
   ```
3. In `app/foundation/container.go`, add:
   ```go
   // ship:container:start
   // ship:container:end
   ```
   At the location where module service initialization should be inserted.
4. Do NOT change any logic — only add/normalize comments.

**Done when:** Markers exist in both files. `go build ./...` passes.

---

### C03 — Implement `ship module:add` CLI command

**Status:** `[ ] todo`
**Files:** `tools/cli/ship/internal/cli/cli.go`, new `tools/cli/ship/internal/commands/module.go`
**Depends on:** C01, C02
**Context:**
`ship module:add <name>` should install a module by: adding import, updating `config/modules.yaml`, inserting wiring at marker comments in container + router. Read `docs/roadmap/02-architecture-evolution.md` section 2 for full spec.

Supported modules for initial implementation: `notifications`, `paidsubscriptions`, `emailsubscriptions`, `jobs`, `pwa` (once extracted).

**What to do:**
1. Read existing `make:controller` or `make:scaffold` generators to understand the code-insertion pattern used (marker comments + text replacement).
2. Create `tools/cli/ship/internal/commands/module.go` with:
   - `module:add <name>` subcommand
   - For each known module: define what wiring lines to insert at each marker
   - Insert import into container file
   - Insert service init into container at `ship:container:start` marker
   - Insert route registration into router at appropriate `ship:routes:*:start` marker
   - Update `config/modules.yaml`
3. Register the command in `cli.go`.
4. Add `--dry-run` flag that shows what would change without writing.

**Done when:** `ship module:add notifications --dry-run` outputs correct diff. `ship module:add notifications` correctly wires the module (verify by reading modified files).

---

### C04 — Implement `ship module:remove` CLI command

**Status:** `[ ] todo`
**Files:** `tools/cli/ship/internal/commands/module.go`
**Depends on:** C03
**Context:**
Reverse of C03. Removes wiring inserted by `ship module:add`.

**What to do:**
1. For each module, define the lines to remove (must match exactly what C03 inserts).
2. Remove from container, router, modules.yaml.
3. Do NOT remove DB migrations (data safety — user must handle that manually). Print a reminder.
4. Add `--dry-run` flag.

**Done when:** `ship module:remove notifications` correctly removes wiring. Compile check passes.

---

### C05 — Add `ship_modules` and `ship_routes` tools to MCP server (parallel)

**Status:** `[ ] todo`
**Files:** `tools/mcp/ship/`
**Depends on:** A02 (for `ship_doctor`), nothing else
**Context:**
Expand MCP from 3 read-only tools to include inspection and validation tools. See `docs/roadmap/02-architecture-evolution.md` section 4.

**What to do:**
1. Read `tools/mcp/ship/` fully.
2. Add tools:
   - `ship_doctor`: runs `ship doctor --json`, returns parsed JSON
   - `ship_routes`: parses `app/router.go` to extract route inventory, returns JSON array of `{method, path, auth, handler}`
   - `ship_modules`: reads `config/modules.yaml` + scans `modules/` directory, returns installed modules + available modules as JSON
3. Each tool: clear name, description, input schema (if any), output schema documented in tool definition.
4. Test each tool returns valid JSON.

**Done when:** Three new MCP tools work and return valid JSON. Existing tools unchanged.

---

## Group D — Module Extraction

> Each extraction is independent. Do them in any order, but do C01–C02 first.

### D01 — Extract auth controllers into `modules/auth` (parallel)

**Status:** `[ ] todo`
**Files:** `app/web/controllers/login.go`, `register.go`, `logout.go`, `forgot_password.go`, new `modules/auth/`
**Depends on:** C01
**Context:**
Auth flows (login, register, logout, password reset) are currently app-level controllers. They should be a self-contained module. Auth *primitives* (token generation, password hashing) stay in framework.

**What to do:**
1. Read all four controller files + their views in `app/views/web/pages/` and `app/views/web/components/auth.templ`.
2. Create `modules/auth/`:
   - `module.go` — implements `Module` interface, ID: `"auth"`
   - `routes.go` — registers `/login`, `/register`, `/logout`, `/forgot-password`, `/reset-password`
   - `service.go` — thin wrapper delegating to framework auth primitives
   - `views/` — move auth templ files here
3. Update `app/router.go` to call `authModule.RegisterRoutes(...)` instead of directly registering routes.
4. Delete the four original controller files (or keep as thin shims temporarily).
5. Run `go build ./...` and `make test-unit`.

**Done when:** Auth routes work via the module. Old controllers deleted. Tests pass.

---

### D02 — Extract profile into `modules/profile` (parallel)

**Status:** `[ ] todo`
**Files:** `app/profile/`, `app/web/controllers/profile.go`, `profile_photo.go`, `upload_photo.go`, new `modules/profile/`
**Depends on:** C01
**Context:**
User profile management (view, edit, photo upload) + the profile domain logic in `app/profile/`.

**What to do:**
1. Read all profile files and their views.
2. Create `modules/profile/`:
   - `module.go` — ID: `"profile"`
   - `service.go` — wraps existing profile domain logic
   - `store.go` / `store_sql.go` — any profile-specific DB queries
   - `routes.go` — `/profile/*`, `/uploadPhoto/*`
   - `views/` — profile templ files
   - `db/migrations/` — any profile-specific schema (if separable)
3. Wire via `modules/profile` in router. Delete originals.
4. Run `go build ./...` and `make test-unit`.

**Done when:** Profile routes work via module. Tests pass.

---

### D03 — Move paidsubscriptions route handler into module (parallel)

**Status:** `[ ] todo`
**Files:** `app/web/controllers/payments.go`, `modules/paidsubscriptions/routes.go` (new)
**Depends on:** C01
**Context:**
`modules/paidsubscriptions` already has service + store. The route handler (`payments.go`) is still in `app/web/controllers/`. Move it into the module.

**What to do:**
1. Read `app/web/controllers/payments.go` and `modules/paidsubscriptions/`.
2. Create `modules/paidsubscriptions/routes.go` with the handler logic (moved from controller).
3. Have the module implement `RoutableModule` and register its own routes.
4. Delete `app/web/controllers/payments.go`.
5. Update `app/router.go` to call `paidsubscriptionsModule.RegisterRoutes(...)`.
6. Run `go build ./...` and tests.

**Done when:** Payments routes work via module. Old controller deleted.

---

### D04 — Move notifications route handlers into module (parallel)

**Status:** `[ ] todo`
**Files:** `app/web/controllers/notifications.go`, `push_notifs.go`, `modules/notifications/routes.go` (new)
**Depends on:** C01
**Context:**
Same pattern as D03. `notifications.go` (249 lines) and `push_notifs.go` (407 lines) belong in the notifications module.

**What to do:**
1. Read both controllers and `modules/notifications/`.
2. Create `modules/notifications/routes.go`.
3. Move handler logic. Implement `RoutableModule`.
4. Delete old controllers.
5. Update router.
6. Run `go build ./...` and tests.

**Done when:** Notification routes work via module. Old controllers deleted.

---

### D05 — Create `modules/pwa` (parallel)

**Status:** `[ ] todo`
**Files:** `app/web/controllers/install_app.go`, PWA-related templ components, service worker, manifest, new `modules/pwa/`
**Depends on:** C01
**Context:**
PWA support (service worker registration, manifest.json, install prompt, push subscription) is scattered across the app. Extract into a self-contained module.

**What to do:**
1. Find all PWA-related files (service worker, manifest, install_app controller, pwa_install templ component, push notif subscription).
2. Create `modules/pwa/`:
   - `module.go` — ID: `"pwa"`
   - `routes.go` — `/install-app`, serve manifest, service worker
   - `views/` — PWA install UI components
   - Static assets: manifest template, service worker JS
3. Wire via module. Delete originals.

**Done when:** PWA install flow works via module. Old files deleted.

---

## Group E — App Split: Landing vs Starter

### E01 — Create `starter/` skeleton app (parallel)

**Status:** `[ ] todo`
**Files:** new `starter/` directory
**Depends on:** D01, D02 (so auth + profile modules exist to include)
**Context:**
Create a minimal app skeleton that becomes the template for `ship new`. It should include auth, profile, home feed — nothing else. No landing page, no payments, no push notifications.

**What to do:**
1. Create `starter/` mirroring the `app/` structure but minimal:
   - `starter/router.go` — only public routes (landing placeholder) + auth module + profile module
   - `starter/foundation/container.go` — minimal container (DB, auth, mail only)
   - `starter/views/web/pages/home_feed.templ` — simple home feed page
   - `starter/views/web/pages/landing.templ` — minimal landing placeholder
2. Ensure it compiles standalone: `cd starter && go build ./...`
3. Document in `starter/README.md`: "This is the minimal GoShip starter. Add modules with `ship module:add`."

**Done when:** `starter/` compiles. Contains only auth + profile + home feed. No payments/push/etc.

---

### E02 — Wire `ship new` to use `starter/` as template

**Status:** `[ ] todo`
**Files:** `tools/cli/ship/internal/commands/new.go`
**Depends on:** E01
**Context:**
`ship new myapp` should scaffold from the `starter/` skeleton, not the full app.

**What to do:**
1. Read the current `ship new` implementation.
2. Update it to copy/template from `starter/` (either embedded in CLI binary or fetched from a known URL).
3. Replace placeholder names (module name, package paths) in generated files.
4. Print post-install instructions: `cd myapp && ship module:add <module> && make dev`.

**Done when:** `ship new testapp` generates a working minimal app based on starter.

---

## Group F — Documentation Updates

### F01 — Update `docs/00-index.md` with new roadmap docs (parallel)

**Status:** `[ ] todo`
**Files:** `docs/00-index.md`
**Depends on:** nothing

**What to do:**
Read `docs/00-index.md`. Add entries for:
- `M02` — `roadmap/02-architecture-evolution.md`
- `M03` — `roadmap/03-atomic-tasks.md` (this file)

**Done when:** Index references both new roadmap documents.

---

### F02 — Update `docs/architecture/01-architecture.md` to reflect module system (parallel)

**Status:** `[ ] todo`
**Files:** `docs/architecture/01-architecture.md`
**Depends on:** C01 (so the module interface exists to document)

**What to do:**
1. Read the current architecture doc.
2. Add a "Module System" section explaining:
   - The `Module` and `RoutableModule` interfaces
   - How modules are installed (`ship module:add`)
   - What a module can bring (routes, views, migrations)
   - List of available modules with one-line descriptions
3. Update the request flow diagram if it exists to show module route registration.

**Done when:** Architecture doc accurately describes the module system.

---

### F03 — Update `docs/reference/01-cli.md` with new commands (parallel)

**Status:** `[ ] todo`
**Files:** `docs/reference/01-cli.md`
**Depends on:** C03, C04, A02 (so commands exist before documenting)

**What to do:**
Add entries for:
- `ship module:add <name>` — description, options, examples
- `ship module:remove <name>` — description, warnings (no auto-migration rollback)
- `ship doctor --json` — new flag, output schema

**Done when:** CLI reference accurately documents new commands.

---

### F04 — Update `docs/reference/02-mcp.md` with new MCP tools (parallel)

**Status:** `[ ] todo`
**Files:** `docs/reference/02-mcp.md`
**Depends on:** C05

**What to do:**
Document the 3 new MCP tools (`ship_doctor`, `ship_routes`, `ship_modules`) with input schema, output schema, and example responses.

**Done when:** MCP reference covers all 6 tools (3 existing + 3 new).

---

## Completion Tracker

```
Group A — Critical Fixes
[ ] A01  Container initialization bug fixed
[ ] A02  ship doctor --json flag added

Group B — JS Islands
[ ] B01  Vite config set up
[ ] B02  Islands runtime written
[ ] B03  ThemeToggle migrated (first island)
[ ] B04  All Svelte components migrated
[ ] B05  Old esbuild setup removed

Group C — Module System
[ ] C01  Module interface defined in framework
[ ] C02  Marker comments added to router + container
[ ] C03  ship module:add implemented
[ ] C04  ship module:remove implemented
[ ] C05  MCP tools: ship_doctor, ship_routes, ship_modules

Group D — Module Extraction (all parallel after C01)
[ ] D01  Auth → modules/auth
[ ] D02  Profile → modules/profile
[ ] D03  Payments handler → modules/paidsubscriptions
[ ] D04  Notifications handler → modules/notifications
[ ] D05  PWA → modules/pwa

Group E — App Split
[ ] E01  starter/ skeleton created
[ ] E02  ship new uses starter/

Group F — Documentation
[ ] F01  docs/00-index.md updated
[ ] F02  Architecture doc updated with module system
[ ] F03  CLI reference updated
[ ] F04  MCP reference updated
```

---

## Recommended Execution Order

1. **A01, A02** — fix critical bugs first (parallel)
2. **B01, B02, C01, C02, F01** — foundation work (all parallel)
3. **B03, C03** — first island + module:add (B03 needs B01+B02; C03 needs C01+C02)
4. **B04, C04, C05, D01, D02, D03, D04, D05, F02** — main body of work (mostly parallel)
5. **B05, E01** — cleanup and starter (B05 needs B04; E01 needs D01+D02)
6. **E02, F03, F04** — final wiring (E02 needs E01; F03 needs C03+C04+A02; F04 needs C05)
