# Pagoda Inspirations & DX Improvements

**Reference:** `docs/roadmap/02-architecture-evolution.md` for broader context.
**Last updated:** 2026-03-08

This document captures ideas sourced from two places:
1. [Pagoda](https://github.com/mikestefanello/pagoda) — the Go starter kit GoShip originally branched from, which has diverged significantly but still has patterns worth adopting.
2. Rails / Laravel — the gold standards for web framework DX.

Each item includes a clear rationale, the pagoda/framework precedent, and whether it's a small lift or architectural change.

---

## Part 1 — What to Pull from Pagoda

### 1.1 Single-Binary Deployment Mode

**The gap:** GoShip requires Postgres + Redis + Docker just to run locally. Pagoda runs as a single binary with no external dependencies — SQLite for data, Backlite (SQLite-backed queue) for jobs, Otter (in-memory) for cache.

**Why it matters:** For indie devs, internal tools, self-hosted apps, and the GoShip starter itself — requiring a full stack is a huge barrier. `ship new myapp && make run` should produce a working app with zero external dependencies.

**The architecture:** GoShip's adapter interfaces (`core.Store`, `core.Cache`, `core.Jobs`) were designed for exactly this. They just need SQLite/Otter/Backlite adapters.

**Three deployment modes to support:**

| Mode | DB | Cache | Jobs | Use case |
|---|---|---|---|---|
| **Single binary** | SQLite (`modernc.org/sqlite`) | Otter (in-memory) | Backlite (SQLite queue) | Starter, indie, self-hosted |
| **Standard** | Postgres | Redis | Asynq (Redis) | Production SaaS |
| **Scaled** | Postgres | Redis | Asynq + separate worker process | High-traffic, horizontal scale |

Config selects the mode. `ship new` defaults to single-binary. Upgrading to standard is `ship adapter:set db=postgres cache=redis jobs=asynq`.

**Important:** Use `modernc.org/sqlite` (CGO-free, pure Go) not `go-sqlite3` (requires CGO). CGO breaks cross-compilation and complicates single-binary distribution.

**Pagoda precedent:** Full single-binary since v1. Stores data in `./dbs/` directory. `make run` is the only command needed.

---

### 1.2 Backlite — SQLite-Backed Job Queue

**The gap:** GoShip's `jobs` module uses Asynq which requires Redis. In single-binary mode there is no Redis.

**What Backlite provides:**
- SQLite as the job queue backend (same DB, no extra infra)
- Worker pool with configurable goroutine count
- Delayed jobs (execute after duration or at specific time)
- Retry with configurable attempts
- Task monitoring UI (embeds into admin panel via HTTP handler)
- Automatic schema install on startup

**Fit:** Add as a new driver under `modules/jobs/drivers/backlite/`. The existing `core.Jobs` interface defines the contract — Backlite implements it.

**Pagoda precedent:** Written by pagoda's author specifically for this use case. Actively maintained.

**Config selection:**
```yaml
jobs:
  driver: backlite  # or: asynq
  backlite:
    goroutines: 10
    release_after: 30s
    cleanup_interval: 1h
```

---

### 1.3 Otter — In-Memory Cache Adapter

**The gap:** GoShip's cache adapter requires Redis. In single-binary mode there is no Redis.

**What Otter provides:**
- Lockless in-memory cache using S3-FIFO eviction
- Very high throughput (benchmarks beat Ristretto)
- No external dependencies
- Works perfectly for single-process deployment

**Fit:** Add as a new `CoreCache` adapter. The existing `core.Cache` interface defines the contract.

**Limitation:** In-memory cache does not share state across multiple processes. Only valid for single-binary mode. When scaling to multiple web processes, swap to Redis adapter.

**Pagoda precedent:** Pagoda's default cache since they dropped Redis. Has `CacheStore` interface to swap Redis back in when needed.

**Important note on pagoda's cache API:**
Pagoda exposes a chainable builder API over the cache:
```go
c.Cache.Set().Key("k").Tags("tag1").Expiration(time.Hour).Data(myData).Save(ctx)
c.Cache.Get().Group("g").Key("k").Fetch(ctx)
c.Cache.Flush().Tags("tag1", "tag2").Execute(ctx)
```
This is significantly more ergonomic than a raw `Get(key)/Set(key, val, ttl)` interface. GoShip should adopt this API shape for the `core.Cache` interface.

---

### 1.4 Admin Panel (Auto-Generated from Ent Schema)

**The gap:** GoShip has no admin panel. Pagoda auto-generates one from the Ent schema using Ent's extension API — every entity type gets list/create/edit/delete UI automatically. It also embeds the Backlite queue monitor in the admin.

**Why it matters for DX:** This is a major productivity win. Every internal tool, SaaS, or early-stage product needs to manage data. Without an admin panel, every team builds bespoke tooling. Rails has ActiveAdmin; Laravel has Filament. Go has nothing mainstream — this could be GoShip's differentiator.

**How pagoda does it:**
1. A custom Ent extension (`ent/admin/extension.go`) generates flat structs and handler code for each entity type during `make ent-gen`.
2. A single `admin.go` handler serves all entity routes dynamically.
3. The UI uses gomponents (pagoda's rendering engine) to build forms dynamically from the Ent graph data structure.

**How it works without Ent (GoShip uses Bob):**
Pagoda's approach requires Ent's schema graph. GoShip uses Bob. Instead, use Go reflection + generics:

```go
// Register any Go struct type with the admin module
admin.Register[Post](admin.Config{
    TableName:  "posts",
    ListFields: []string{"title", "published_at"},
    Sensitive:  []string{"internal_notes"},
})
```

`admin.Register[T]()` uses `reflect.TypeOf(*new(T))` at runtime to enumerate exported fields, derive their types, and build a slice of `AdminField` descriptors. The admin module then drives all CRUD via raw SQL through `database/sql` (not Bob's codegen, since the resource type is dynamic).

**The UI — templ components, data-driven:**
GoShip uses templ, not gomponents. Templ is compiled, so templates cannot be generated at runtime — but they CAN be fully dynamic through data. The admin templ components receive `[]AdminField` at runtime and `switch` on field type to render the appropriate input:

```templ
templ AdminFieldInput(field AdminField) {
    switch field.Type {
    case "string":  <input type="text" name={field.Name} value={field.StringValue}>
    case "bool":    <input type="checkbox" name={field.Name} checked?={field.BoolValue}>
    case "int":     <input type="number" name={field.Name} value={field.IntValue}>
    case "time":    <input type="datetime-local" ...>
    case "text":    <textarea name={field.Name}>{ field.StringValue }</textarea>
    }
}
```

The dynamic behavior is in the **data** (field descriptors derived from reflection), not the template. This works perfectly with templ's compiled model.

**No Ent. No Ent extension. No Ent for admin-only.** Pure reflection + Bob runtime queries + templ components.

---

### 1.5 Afero — File System Abstraction

**The gap:** GoShip's file storage uses MinIO (S3-compatible), which requires infrastructure even locally.

**What afero provides:** A `fs.FS`-compatible abstraction with backends for local OS, GCS, SFTP, in-memory (for tests), and more. Swap backends without changing application code.

**Fit:** Add as an alternative to MinIO for local development and single-binary mode. In-memory backend for tests means no MinIO in CI.

**Pagoda precedent:** Default file system is local OS. Tests use in-memory automatically.

**Config selection:**
```yaml
storage:
  driver: local   # or: minio, gcs
  local:
    path: ./uploads
```

---

### 1.6 Chainable Redirect Helper

**The gap:** Redirects in GoShip are manual `c.Redirect(http.StatusFound, url)` calls with manual URL construction.

**Pagoda's pattern:**
```go
return redirect.New(ctx).
    Route("user_profile").
    Params(userID).
    Query(queryParams).
    Go()
```
Automatically handles HTMX redirects (sets `HX-Redirect` header for boosted requests). Type-safe route names. Chainable.

**Fit:** Small addition to `framework/htmx` or a new `framework/redirect` package. High DX value for low effort.

---

### 1.7 In-Memory Test Database

**The gap:** GoShip's integration tests require Docker + Postgres. This slows CI and makes tests harder to run locally.

**Pagoda's pattern:** When `config.EnvTest` is set, the container auto-connects to an in-memory SQLite database and runs migrations. Tests start instantly.

**GoShip adaptation:**
- `config.SwitchEnvironment(config.EnvTest)` sets env before container init.
- Container init: if env is test, use SQLite in-memory for DB, in-memory for cache, sync (no-op) for jobs.
- No Docker required for unit or route-level tests.
- Integration tests (testing actual Postgres behavior) remain Docker-based but are clearly separated.

**Pagoda precedent:** Enabled fast, zero-infrastructure test runs. A game-changer for iteration speed.

---

## Part 2 — Rails / Laravel DX Ideas

### 2.1 `ship console` — Interactive Database Session

**Rails:** `rails console` opens an IRB session connected to the live DB.
**Laravel:** `php artisan tinker` opens a PsySH REPL.

**Go reality:** This doesn't translate. Go compiles to machine code — there's no interpreter that can load your live app's types, run queries, and inspect results the way Ruby/PHP can. REPLs like `gore` or `yaegi` exist but they can't import your compiled app and have very limited stdlib/package support. Not worth pursuing.

**Practical Go-native alternative:** `ship db:console` (see 2.4) drops you into a raw DB shell. For structured data inspection, the admin panel (1.4) covers most real use cases. For one-off scripts, `cmd/scripts/` convention with access to the container is the Go way.

---

### 2.2 `ship routes` — Route Table

**Rails:** `rails routes` prints a table of all routes (verb, path, name, handler).
**Laravel:** `php artisan route:list`.

**GoShip proposal:** `ship routes` parses `app/router.go` (or inspects the running Echo instance) and prints:

```
METHOD  PATH                              AUTH    HANDLER
GET     /                                 public  landing.Get
POST    /user/register                    public  register.Post
GET     /auth/homeFeed                    auth    home_feed.Get
POST    /auth/payments/checkout           auth    payments.CreateCheckoutSession
...
```

**Value:** LLMs and devs can audit routes without reading the router file. Also useful as an MCP tool.

**Implementation:** Parse router.go at compile time (AST) or add Echo's `Routes()` method output to the CLI at runtime.

---

### 2.3 Auto-CRUD Admin Panel (see 1.4)

**Rails:** ActiveAdmin, Administrate.
**Laravel:** Filament, Nova.

Already covered in section 1.4 — the pagoda implementation is the direct model.

---

### 2.4 `ship db:console` — Direct DB Shell

**Rails:** `rails db` opens a psql/mysql/sqlite3 shell connected to the configured database.
**Laravel:** `php artisan db`.

**GoShip proposal:** `ship db:console` reads the active DB config and spawns `psql`, `mysql`, or `sqlite3` with the correct connection string.

**Value:** Fast data inspection without remembering connection strings.

---

### 2.5 Built-in Rate Limiter

**Rails:** `throttle` via rack-attack.
**Laravel:** `throttle` middleware.

**GoShip proposal:** A configurable rate-limiting middleware in the framework. Echo has `echo.IPRateLimit()` but it's minimal. A proper implementation:
- Per-IP and per-user limits
- Configurable per route group
- Backed by in-memory (Otter) for single binary, Redis for scaled
- Returns 429 with `Retry-After` header

---

### 2.6 DB-Backed Sessions (Optional)

**Rails:** `ActiveRecord::SessionStore`.
**GoShip current:** Cookie-only sessions (Gorilla sessions).

**Problem:** Cookie sessions work fine for single-server. Scaling to multiple web processes with cookie sessions requires sticky sessions at the load balancer.

**Proposal:** Add a DB-backed session store implementation as an option. The Gorilla sessions ecosystem has existing implementations for Postgres and SQLite. Config-selectable:
```yaml
session:
  store: cookie   # or: db
```

---

### 2.7 First-Class `.env` Support

**Rails/Laravel:** `.env` files for local secrets via dotenv.

**GoShip current:** YAML config with environment variable overrides via viper. Works, but the ergonomics for secrets (DB password, Stripe key) is cumbersome.

**Proposal:** Load `.env` at startup before viper config resolution. `.env` variables map to the same `GOSHIP_*` prefixed env var names already supported. `.env.example` committed; `.env` gitignored.

**Value:** Standard pattern every developer expects. Makes `ship new myapp` produce a project that works immediately after editing `.env`.

**Chosen library: `cleanenv` (`github.com/ilyakaznacheev/cleanenv`)**

Wins over `envconfig` (Kelsey Hightower):
- Built-in `.env` file loading (no separate godotenv needed)
- Auto-generates help/usage text for `ship config:validate`
- `required` and `env-default` tags built-in
- One dependency replaces Viper + godotenv

Config struct pattern:
```go
type Config struct {
    DatabaseURL string `env:"DATABASE_URL,required"`
    SecretKey   string `env:"SECRET_KEY,required"`
    Port        int    `env:"PORT" env-default:"8080"`
    RedisURL    string `env:"REDIS_URL"`
}

func Load() (*Config, error) {
    cfg := &Config{}
    _ = cleanenv.ReadConfig(".env", cfg) // load .env if present, ignore if absent
    return cfg, cleanenv.ReadEnv(cfg)    // overlay actual env vars
}
```

---

### 2.8 Pagination as First-Class

**Pagoda has it:** A `Pager` utility for cursor/offset pagination with page size, current page, total pages, and a `HasPages()` check.

**GoShip status:** Manual pagination in each controller.

**Proposal:** Add a `framework/pager` package:
```go
p := pager.New(ctx, 20)  // 20 per page
results, err := db.Query().Limit(p.Limit()).Offset(p.Offset())...
page.Pager = p
```
Templ component renders prev/next links automatically from `page.Pager`.

---

## Part 3 — Priority Matrix

| Item | Value | Effort | Priority |
|---|---|---|---|
| **1.2 Backlite driver** | Very High | Medium | P0 |
| **1.3 Otter cache adapter** | Very High | Low | P0 |
| **1.1 SQLite DB adapter** | Very High | Medium | P0 |
| **1.7 In-memory test DB** | High | Low | P0 |
| **1.6 Chainable redirect** | Medium | Low | P1 |
| **2.2 `ship routes`** | High | Low | P1 |
| **2.7 `.env` support** | High | Low | P1 |
| **2.8 Pagination utility** | Medium | Low | P1 |
| **1.4 Admin panel** | Very High | High | P2 |
| **1.5 Afero file system** | Medium | Medium | P2 |
| **2.4 `ship db:console`** | Medium | Low | P2 |
| **2.5 Rate limiter** | Medium | Medium | P2 |
| **2.6 DB sessions** | Low | Medium | P3 |
| **2.1 `ship console`** | N/A | N/A | ❌ Not viable in Go |

**P0 = unlocks single-binary deployment. Do these together as a unit.**

---

## Part 4 — Single Binary Release Checklist

For GoShip to support `ship new myapp && make run` with zero external dependencies, the following must all be done:

```
[ ] SQLite adapter for core.Store (modernc.org/sqlite, CGO-free)
[ ] Backlite driver for modules/jobs
[ ] Otter adapter for core.Cache
[ ] In-memory test DB (SQLite in EnvTest)
[ ] Goose SQLite dialect support verified
[ ] config/application.yaml: default to single-binary mode
[ ] ship new: scaffold with single-binary defaults
[ ] Makefile: make run works without docker-compose
[ ] docs: "single binary" getting-started guide
```

When all boxes are checked, GoShip can legitimately claim: **one binary, zero dependencies, production-ready**.

---

## Part 4 — Nil Safety: Eliminating Nil Deref Panics

Go + templ nil dereference panics are the most common runtime crash class. The fix is architectural, not defensive.

### Root causes
1. Domain model pointers (`*User`, `*string`) flowing directly into templ components
2. Uninitialized nested structs in viewmodels
3. Optional DB columns as `*string` instead of `sql.NullString`

### The architecture fix: value-type viewmodels

Create a hard boundary between domain models and viewmodels:

- **Domain models** (`db/gen/`, `framework/domain/`) — pointers allowed for nullable DB columns
- **Viewmodels** (`app/web/viewmodels/`) — **zero pointer fields**. All value types, fully initialized
- **Templ components** — accept viewmodel types or primitives only. Never `*DomainModel`
- **Controllers** — own the domain → viewmodel transformation. All nil handling happens here

```go
// Domain model — pointer fields OK
type User struct { Name *string }

// Viewmodel — value type only
type UserCardVM struct { DisplayName string }  // empty string = absent, never nil

// Controller transforms
func toUserCardVM(u *User) UserCardVM {
    return UserCardVM{DisplayName: stringOr(u.Name, "")}
}
```

### Nil-safe domain accessors

```go
func (u *User) DisplayName() string {
    if u == nil || u.Name == nil { return "" }
    return *u.Name
}
```

Go methods on nil pointer receivers are legal if they nil-guard at entry.

### Enforcement

- **`nilaway`** (Uber) in CI — statically traces nil flows across function boundaries
- **`middleware.Recover()`** as first middleware — panics return 500, app stays alive
- **Route smoke tests** with zero-value data — nil deref shows up in test, not production
- **Viewmodel constructors** — `NewUserCardVM(u *User) UserCardVM` guarantees all fields set

### Priority

Add to `app/web/viewmodels/` convention: no pointer fields, ever. This is a permanent architectural rule, enforced by nilaway in CI.
