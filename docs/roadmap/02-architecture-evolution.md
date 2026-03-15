# GoShip Architecture Evolution

**Status:** Active planning — tasks tracked in `03-atomic-tasks.md`
**Last updated:** 2026-03-08

This document captures the architectural direction for GoShip as a Rails/Laravel-inspired, LLM-forward Go framework. It is the reference for understanding *why* changes are being made, not just what they are.

---

## Vision

GoShip should be:

1. **Minimal core** — the framework provides just enough to wire a Go web app (routing, DI, config, DB, auth primitives, rendering). Everything else is opt-in.
2. **Self-contained modules** — each optional capability (notifications, payments, profile, PWA, etc.) ships as a module that brings its own schema, migrations, routes, and views, and can be installed/removed via CLI.
3. **LLM-forward** — the codebase structure, tooling, and conventions are optimized so that an LLM can scaffold, edit, and verify changes reliably with minimal human correction.
4. **Islands-first JS** — JS components are lazy-loaded per-page via an islands architecture. Each component is a proper JS/TS file with full tooling. No monolithic bundle.
5. **Single canonical app** — the GoShip repo contains one runtime app (`app/` + `cmd/`), and `ship new` scaffolds from CLI-embedded starter templates.

---

## 1. JS Islands Architecture

### Problem with Current Approach

The current setup:
- Bundles all Svelte components into one `svelte_bundle.js` (~all pages load all components)
- Mounts components via `window.renderSvelteComponent('Name', id, props)` global
- No code splitting, no lazy loading
- Svelte-only (wiring a React or Vue component requires significant plumbing)
- TypeScript/linting in Svelte files works, but the wiring to templ is a magic string

### Target Architecture: Data-Attribute Islands

**Runtime contract:**

In templ, declare a mount point:
```templ
<div data-island="ThemeToggle" data-props={ templ.JSONString(props) }></div>
```

A small runtime script (`islands-runtime.js`, ~30 lines) runs on page load and after every HTMX swap:
1. Finds all `[data-island]` elements that aren't mounted yet
2. Looks up the island name in a build manifest
3. Dynamically imports the island's JS chunk (framework-agnostic)
4. Calls the island's exported `mount(el, props)` function

**Island file contract** (each island is a standalone file):
```
frontend/islands/ThemeToggle.svelte     → exports mount(el, props)
frontend/islands/EmojiPicker.tsx        → exports mount(el, props)
frontend/islands/MapView.vue            → exports mount(el, props)
```

Each framework adapter (Svelte, React, Preact, Vue) is ~3 lines to implement `mount`. The runtime doesn't care which framework the island uses.

**Build tool: Vite**

Switch from esbuild to Vite:
- Vite handles per-island code splitting natively via Rollup
- `@sveltejs/vite-plugin-svelte`, `@vitejs/plugin-react`, `@vitejs/plugin-vue` all maintained
- Generates a manifest (`islands-manifest.json`) mapping island names → hashed chunk URLs
- HMR during development
- Tailwind CSS via Vite plugin (replaces standalone Tailwind CLI invocation)

**HTMX compatibility:**
- Islands runtime re-scans after `htmx:afterSettle` event
- Islands are idempotent: `el.__island_mounted` flag prevents double-mount
- No conflict with HTMX swapping regions that contain islands — the island remounts cleanly

**What changes in templ:**
- Remove `script` blocks that call `renderSvelteComponent()`
- Replace with `<div data-island="..." data-props="...">` mount points
- Remove import of `svelte_bundle.js` from core layout
- Add import of `islands-runtime.js` (small, loads once)

### Migration Path

1. Build Vite config with island splitting
2. Write `islands-runtime.js`
3. Write per-framework `mount()` adapter conventions
4. Migrate existing Svelte components one by one (ThemeToggle, EmojiPicker, etc.)
5. Remove old esbuild setup and global registry

---

## 2. Module System Evolution

### What a Module Should Be

A self-contained Go package that provides:

```
modules/mymodule/
├── module.go          # Module ID, config schema, dependency declarations
├── service.go         # Business logic (exported API)
├── store.go           # Storage interface
├── store_sql.go       # SQL implementation
├── routes.go          # Route registration (optional)
├── views/             # Templ templates (optional)
│   └── web/
├── db/                # Migrations
│   └── migrations/
└── *_test.go
```

`module.go` exports a `Module` struct conforming to a framework interface:

```go
type Module interface {
    ID() string
    Configure(cfg any) error
    Migrations() []string          // paths to migration files
    RegisterRoutes(router Router) error  // optional
}
```

### Module Installation: `ship module:add <name>`

When you run `ship module:add notifications`, the CLI:
1. Adds the import to `go.mod`
2. Adds the module to `config/modules.yaml`
3. Inserts wiring into `app/foundation/container.go` at a marker comment
4. Inserts route registration into `app/router.go` at a marker comment
5. Runs `db:migrate` if the module has migrations

When you run `ship module:remove notifications`, it reverses all of the above.

This is the same marker-comment pattern already used for route wiring in `app/router.go`.

### What Should Become a Module

| Currently in `app/` | Becomes module | Notes |
|---|---|---|
| `web/controllers/login.go`, `register.go`, `logout.go`, `forgot_password.go` | `modules/auth` | Auth flows + views. Auth primitives stay in framework. |
| `profile/`, `web/controllers/profile.go`, `profile_photo.go`, `upload_photo.go` | `modules/profile` | User profile domain, photo management, settings |
| `web/controllers/preferences.go` (490 lines) | `modules/preferences` | Wraps notifications + profile settings. Depends on `notifications` module. |
| `web/controllers/push_notifs.go` | Merge into `modules/notifications` | Already a module; route handler should move in |
| `web/controllers/email_subscribe.go` | Merge into `modules/emailsubscriptions` | Already a module; route handler should move in |
| PWA service worker, manifest, install prompt | `modules/pwa` | Brings own JS, manifest template, install route |
| `web/controllers/home_feed.go` | Stays in app (app-specific) or `modules/feed` | Depends on product |
| `web/controllers/landing.go`, `about.go`, `contact.go` | Stays in app (app-specific) | Landing is always app-specific |
| `web/controllers/payments.go` | Merge into `modules/paidsubscriptions` | Already a module; route handler should move in |

### What Stays in Framework Core

- HTTP routing adapter (Echo wrapper)
- DI container infrastructure
- Config management
- DB connection + migration runner
- Session handling
- Auth primitives (token generation, password hashing) — not auth flows
- Middleware infrastructure (CSRF, logging, error handling)
- Templ rendering pipeline
- Static asset serving
- Background job infrastructure (adapter interface)
- Healthcheck endpoint

---

## 3. Single-App Repository Model

### Decision

GoShip now uses a single-app repository model.

### Target Structure

```
cmd/
├── web/          # Shared web process entrypoint
├── worker/       # Shared worker process entrypoint
└── seed/         # Shared seeder

app/              # Canonical runtime app for this repository
```

`ship new` template source lives inside the CLI module:

```
tools/cli/ship/internal/templates/starter/testdata/scaffold/
```

This keeps runtime concerns separate from scaffold-template concerns while preserving deterministic, offline `ship new` behavior.

---

## 4. MCP Expansion

### Current State

3 tools: `ship_help`, `docs_search`, `docs_get` — read-only, documentation-only.

### Why MCP Matters

The highest-value pattern for LLM development is: **act → verify → fix**, autonomously. The LLM writes code, calls `ship_doctor` via MCP, gets structured errors, fixes them — without human intervention.

This loop requires MCP tools that provide:
- **Verification**: does my change break anything structurally?
- **Inspection**: what routes/schema/modules exist, so I don't create conflicts?
- **Action**: can I scaffold without needing shell access?

### Tools to Add

| Tool | Input | Output | Value |
|---|---|---|---|
| `ship_doctor` | — | JSON: `{issues: [{type, file, detail}]}` | LLM self-validates after changes |
| `ship_routes` | — | JSON: route inventory (method, path, auth, handler) | LLM avoids route conflicts |
| `ship_schema` | — | JSON: DB tables + columns + types | LLM writes correct migrations |
| `ship_modules` | — | JSON: installed modules + their routes/config | LLM knows what's available |
| `ship_scaffold` | model name + fields | Runs `make:scaffold`, returns generated file paths | LLM creates resources |
| `ship_make_migration` | migration name | Creates migration file, returns path | LLM creates schema changes |

All outputs should be JSON with consistent `{ok: bool, data: any, errors: [string]}` envelope.

### Integration Recommendation

Make `ship doctor` produce machine-readable JSON output (flag: `--json`). The MCP tool just calls it and passes through. This way the CLI and MCP stay in sync automatically.

---

## 5. Container Initialization Fix

### Current Bug

`app/foundation/container.go` has these commented out:
```go
// c.initCache()
// c.initNotifier()
// c.initTasks()
```

But runtime code in several places assumes these are initialized and calls `.Close()` on them at shutdown — risk of nil pointer panics.

### Fix

The container should use a capability-based initialization pattern:

```go
func NewContainer() *Container {
    // ...
    if c.Config.Cache.Enabled {
        c.initCache()
    }
    if c.Config.Notifications.Enabled {
        c.initNotifier()
    }
    if c.Config.Jobs.Enabled {
        c.initTasks()
    }
}
```

All `.Close()` calls must nil-check or use the adapter pattern (`c.CoreCache` is already the right abstraction — use it exclusively).

---

## 6. LLM-Forward Conventions

Beyond the `data-*` UI attribute convention (tracked in `docs/ui/convention.md`), the following make the codebase LLM-reliable:

### Marker Comments for Code Generation

The router already uses `// ship:routes:auth:start` markers. Extend this pattern:
- `// ship:container:start` / `// ship:container:end` — container wiring insertion points
- `// ship:module:start` / `// ship:module:end` — module registration
- `// ship:routes:public:start` etc. — route group markers

These allow `ship module:add` to insert code without regex guessing.

### One Canonical Path Per Concern

Already the GoShip philosophy. Enforce it in `ship doctor`:
- No controller outside `app/web/controllers/`
- No route registration outside `app/router.go`
- No business logic in controllers (must delegate to service/domain layer)

### Structured Error Output from CLI

All CLI commands should support `--json` flag for machine-readable output. MCP tools consume these directly.
