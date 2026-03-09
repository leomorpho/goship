# GoShip — DX & Core Infrastructure

**Status:** Active planning — tasks pickup-ready for any LLM agent
**Last updated:** 2026-03-08

**Reference docs (read before picking up any task):**
- `docs/roadmap/02-architecture-evolution.md` — architecture overview
- `docs/roadmap/03-atomic-tasks.md` — M03 task list (groups A–K)
- `docs/roadmap/05-llm-dx-agent-friendly.md` — M05 task list (groups L–R)
- `docs/guides/01-ai-agent-guide.md` — conventions, safe change workflow

**Stack context:** Go 1.24, Echo v4, Templ, HTMX, Bob ORM, cleanenv config,
ship CLI (`tools/cli/ship/`), Vite frontend, Overmind for process management.

**Task format:** Self-contained. Full context, exact files, "done when" criterion.
Mark `[x]` before starting any dependent task.

**Group prefix:** S–V (continues from M05's L–R).

---

## Group S — Developer Workflow

### S01 — Add `ship dev` unified development command

**Status:** `[ ] todo`
**Depends on:** nothing
**Files:** `Procfile.dev` (new), `tools/cli/ship/internal/commands/dev.go` (new),
`tools/cli/ship/internal/cli/cli.go`, `Makefile`

**Context:** Running GoShip in development currently requires 4–5 separate terminal windows:
`templ generate --watch`, `air` (Go live reload), `pnpm --prefix frontend run dev` (Vite HMR),
`go run cmd/worker/main.go`. Each process has its own output, and agents don't know which to
restart after which kind of change. `ship dev` runs all processes as a single multiplexed stream.

**Implementation approach:** Use Overmind (`github.com/DarthSim/overmind`) or `goreman` to read
`Procfile.dev`. Overmind is preferred: it supports per-process restart, colored output by default,
and is a single static binary.

**`Procfile.dev` content:**
```
web: air -c .air.toml
worker: go run ./cmd/worker/main.go
vite: pnpm --prefix frontend run dev
templ: templ generate --watch --proxy="http://localhost:8080"
```

**What to do:**
1. Create `Procfile.dev` at repo root with the content above.
2. Verify `.air.toml` exists and is configured correctly (if not, create with standard defaults:
   watch `app/`, `config/`, `cmd/`, exclude `tmp/`, build to `tmp/main`).
3. Create `tools/cli/ship/internal/commands/dev.go`:
   - Check if `overmind` is in PATH. If not, check `goreman`. If neither, print install instructions and exit 1.
   - Exec: `overmind start -f Procfile.dev` (replaces current process, inherits stdio).
4. Register `dev` command in `cli.go`.
5. Add `Makefile` target `dev` that calls `ship dev` (convenience alias).
6. Document in `docs/guides/02-development-workflows.md`: "Run `ship dev` to start all processes."

**Done when:** `ship dev` starts all four processes with merged colored output. Killing the command
(Ctrl+C) stops all child processes cleanly. Works from repo root.

---

### S02 — Generate GitHub Actions CI/CD workflows in `ship new`

**Status:** `[ ] todo`
**Depends on:** nothing (parallel with S01)
**Files:** `.github/workflows/ci.yml` (new), `.github/workflows/deploy.yml` (new),
`.github/workflows/security.yml` (new), `.github/dependabot.yml` (new),
`tools/cli/ship/internal/commands/new.go` (update scaffold)

**Context:** Every new GoShip project has a CI gap for weeks after creation — developers add CI
manually and inconsistently. `ship new myapp` should generate working GitHub Actions workflows
from day one. CI should be green on the first push.

**`ci.yml` — runs on every push and PR:**
```yaml
name: CI
on: [push, pull_request]
jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - uses: actions/setup-node@v4
        with: { node-version: '22' }
      - run: go install github.com/a-h/templ/cmd/templ@latest
      - run: pnpm install --prefix frontend
      - run: ship verify --skip-tests  # templ gen + build + doctor
      - run: go test ./...
```

**`deploy.yml` — runs on push to main:**
```yaml
name: Deploy
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: webfactory/ssh-agent@v0.9.0
        with: { ssh-private-key: '${{ secrets.DEPLOY_KEY }}' }
      - run: gem install kamal
      - run: kamal deploy
```

**`security.yml` — weekly vulnerability scan:**
```yaml
name: Security
on:
  schedule: [{ cron: '0 9 * * 1' }]
jobs:
  govulncheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest
      - run: govulncheck ./...
```

**`dependabot.yml`:**
```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule: { interval: weekly }
  - package-ecosystem: npm
    directory: /frontend
    schedule: { interval: weekly }
  - package-ecosystem: github-actions
    directory: /
    schedule: { interval: weekly }
```

**What to do:**
1. Create these four files as templates in `tools/cli/ship/internal/templates/github/`.
2. Update the `ship new` command to copy them into the new project's `.github/` directory.
3. Add a note in the `ship new` output: "GitHub Actions workflows created. Add DEPLOY_KEY secret
   to enable deployment."
4. These files should also exist in the GoShip repo itself (dogfooding).

**Done when:** `ship new myapp` creates all four workflow files. CI workflow runs `ship verify`
correctly on first push (assuming ship is installed on the runner).

---

## Group T — Core Infrastructure

### T01 — Multi-process SQLite safety (WAL mode + connection pool)

**Status:** `[ ] todo`
**Depends on:** M03 I01 (SQLite adapter must exist first)
**Files:** `framework/repos/sql/sqlite_adapter.go` (new or update), `framework/repos/sql/connection.go`

**Context:** SQLite under concurrent HTTP load produces `"database is locked"` errors without
specific configuration. This is a silent killer for single-binary mode — the app appears to work
in development (low concurrency) but fails under any real load. These settings are mandatory,
not optional.

**Required settings (applied at connection open time):**
```go
// Applied via SQLite pragma statements immediately after opening the DB
pragmas := []string{
    "PRAGMA journal_mode=WAL",       // Write-Ahead Logging: readers don't block writers
    "PRAGMA synchronous=NORMAL",     // Safe with WAL, faster than FULL
    "PRAGMA busy_timeout=5000",      // Wait up to 5s before returning SQLITE_BUSY
    "PRAGMA foreign_keys=ON",        // Enforce FK constraints
    "PRAGMA cache_size=-64000",      // 64MB page cache
    "PRAGMA temp_store=MEMORY",      // Temp tables in memory
}
```

**Connection pool pattern:**
- Use a single `*sql.DB` with `SetMaxOpenConns(1)` for **write** operations (SQLite allows one writer)
- Use a separate `*sql.DB` with multiple connections for **read** operations
- OR: use `modernc.org/sqlite`'s WAL mode with `_txlock=immediate` for write transactions

**What to do:**
1. Read the existing SQLite adapter implementation (from M03 I01).
2. Apply all pragma statements immediately after `sql.Open`.
3. Implement the read/write pool separation or `_txlock=immediate` write transactions.
4. Add a test: spin up the SQLite adapter, run 50 concurrent goroutines each doing a write.
   Verify zero "database is locked" errors.
5. Document the settings and rationale in the adapter file as comments.

**Done when:** 50 concurrent writes to SQLite via the adapter produce zero lock errors.
All pragma settings are applied on connection open. Test passes.

---

### T02 — Integrate `slog` structured logging into framework

**Status:** `[ ] todo`
**Depends on:** nothing (parallel)
**Files:** `framework/logging/` (new package), `framework/middleware/logging.go` (update),
`app/foundation/container.go` (wire logger), `config/config.go` (log level config)

**Context:** GoShip currently uses whatever logging each component calls independently. `log/slog`
is in Go's standard library since 1.21 — no external dependency. Structured logging is essential
for production debugging: log lines are JSON objects that can be queried, filtered, and correlated
by request ID.

**Logger setup:**
```go
// Development: human-readable colored output
// Production: JSON lines to stdout (captured by log aggregator)
func NewLogger(env string, level slog.Level) *slog.Logger {
    if env == "production" {
        return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
    }
    return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
```

**Request ID middleware (update existing or create):**
```go
func RequestID() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            id := c.Request().Header.Get("X-Request-ID")
            if id == "" { id = uuid.New().String() }
            c.Set("request_id", id)
            c.Response().Header().Set("X-Request-ID", id)
            // Add to context for slog
            ctx := context.WithValue(c.Request().Context(), logKeyRequestID, id)
            c.SetRequest(c.Request().WithContext(ctx))
            return next(c)
        }
    }
}
```

**Request logging middleware:**
```go
// Logs: method, path, status, latency, request_id, user_id (if authenticated)
// Format in dev: "GET /login 200 3.2ms req=abc123"
// Format in prod: {"method":"GET","path":"/login","status":200,"latency_ms":3,"request_id":"abc123"}
```

**Config additions:**
```go
type Config struct {
    // ...existing fields...
    Log struct {
        Level  string `env:"LOG_LEVEL" env-default:"info"`  // debug, info, warn, error
        Format string `env:"LOG_FORMAT" env-default:"text"` // text (dev) or json (prod)
    }
}
```

**What to do:**
1. Create `framework/logging/logger.go` with `NewLogger(cfg Config) *slog.Logger`.
2. Create `framework/logging/context.go`: `FromContext(ctx) *slog.Logger` and `WithLogger(ctx, logger)`.
3. Update `app/foundation/container.go`: initialize logger in `NewContainer`, store as `c.Logger`.
4. Update request logging middleware to use slog.
5. Add request ID middleware if not present.
6. Replace any `log.Println` / `fmt.Printf` in framework code with `slog` calls.
7. Add log level and format to config struct.

**Done when:** All framework log output goes through slog. Dev output is text, prod is JSON.
Every log line from the request middleware includes `request_id`. `LOG_LEVEL=debug` enables
verbose output. `go build ./...` passes.

---

### T03 — Security headers middleware

**Status:** `[ ] todo`
**Depends on:** nothing (parallel)
**Files:** `framework/middleware/security_headers.go` (new), `app/router.go` (add to middleware stack),
`config/config.go` (CSP config)

**Context:** Without security headers, GoShip apps score C or below on securityheaders.com.
These headers prevent XSS, clickjacking, MIME sniffing, and other attacks. They should be
default-on — developers shouldn't have to add them. The only configurable part is CSP, since
Vite HMR in development needs `'unsafe-eval'` and websocket connections.

**Headers to set:**
```
X-Content-Type-Options: nosniff
X-Frame-Options: SAMEORIGIN
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=()
X-XSS-Protection: 0  (deprecated, explicitly disable to prevent IE bugs)

# In production:
Strict-Transport-Security: max-age=31536000; includeSubDomains

# CSP — configurable, with safe defaults:
Content-Security-Policy: default-src 'self'; script-src 'self' 'nonce-{random}'; ...
```

**Nonce-based CSP approach:**
- Generate a random nonce per request, store in context
- Pass nonce to templ layout via context: `layout.templ` reads `middleware.CSPNonce(ctx)`
- `<script nonce={nonce}>` in layout
- Eliminates need for `'unsafe-inline'` in script-src

**Config:**
```go
type SecurityConfig struct {
    Headers struct {
        Enabled bool   `env:"SECURITY_HEADERS_ENABLED" env-default:"true"`
        HSTS    bool   `env:"SECURITY_HEADERS_HSTS" env-default:"false"` // enable in prod
        CSP     string `env:"SECURITY_HEADERS_CSP"`                       // override full CSP
    }
}
```

**What to do:**
1. Create `framework/middleware/security_headers.go` with a `SecurityHeaders(cfg)` middleware function.
2. Implement nonce generation (crypto/rand, base64-encoded, 16 bytes).
3. Store nonce in Echo context: `c.Set("csp_nonce", nonce)`.
4. Export `CSPNonce(c echo.Context) string` helper.
5. Update `app/router.go`: add `SecurityHeaders(cfg.Security)` to the global middleware stack,
   before route handlers, after recover/logger.
6. Update the base layout templ file to use `CSPNonce` on script tags.
7. In development config: relax CSP to allow Vite HMR websocket (`ws://localhost:5173`).
8. Add `security` section to `config/config.go`.

**Done when:** All responses include the security headers. CSP nonce is set per-request.
Dev config allows Vite HMR without CSP violations. `curl -I http://localhost:8080/` shows all
headers. securityheaders.com scan on staging returns A grade.

---

### T04 — Expand health check endpoint

**Status:** `[ ] todo`
**Depends on:** nothing (parallel)
**Files:** `app/web/controllers/health.go` (update), `framework/health/` (new package)

**Context:** The current `/health` endpoint returns a simple 200 OK. Kubernetes, Render, Fly.io,
and other platforms distinguish between liveness (is the process alive?) and readiness (are
dependencies ready?). GoShip should support both, with structured JSON output.

**Target API:**
```
GET /health       → liveness: always 200 if process is running
GET /health/ready → readiness: 503 if any critical dependency is down
```

**Response shape:**
```json
{
  "status": "ok",
  "version": "1.4.2",
  "uptime": "3h22m14s",
  "checks": {
    "db": {"status": "ok", "latency_ms": 2},
    "cache": {"status": "ok"},
    "jobs": {"status": "ok", "queue_depth": 7}
  }
}
```

**Health check interface:**
```go
// framework/health/health.go
type Checker interface {
    Name() string
    Check(ctx context.Context) CheckResult
}

type CheckResult struct {
    Status    string         `json:"status"`  // "ok" or "error"
    LatencyMs int64          `json:"latency_ms,omitempty"`
    Error     string         `json:"error,omitempty"`
    Extra     map[string]any `json:"extra,omitempty"`
}
```

**What to do:**
1. Create `framework/health/` package with `Checker` interface and `Registry`.
2. Create DB checker: pings DB with 2s timeout, measures latency.
3. Create cache checker: set/get a test key.
4. Create jobs checker: returns queue depth (adapter-specific).
5. Each installed module can register its own checker via `health.Register(checker)`.
6. Update `app/web/controllers/health.go`:
   - `GET /health` → 200 always (process is up), minimal JSON `{"status":"ok"}`
   - `GET /health/ready` → runs all checkers, 200 if all ok, 503 if any fail
7. Expose version (from `go build -ldflags "-X main.version=..."`) and uptime in response.
8. These routes must be public (no auth) — add before auth middleware in router.

**Done when:** `GET /health/ready` returns full JSON with all checker results. Returns 503 if
DB is unreachable. Returns 200 with correct latencies when all systems are healthy.

---

## Group U — Email System

### U01 — Email provider interface and adapters

**Status:** `[ ] todo`
**Depends on:** nothing (parallel)
**Files:** `framework/core/interfaces.go` (add Mailer interface), `modules/mailer/` (new),
`modules/mailer/drivers/smtp/`, `modules/mailer/drivers/resend/`, `config/config.go`

**Context:** GoShip currently has no email sending abstraction. Email is universal — every app
needs it for auth flows (password reset, verification), notifications, and transactional messages.
The mailer interface must be swappable: SMTP for self-hosted, Resend/SendGrid for production.

**Core interface:**
```go
// framework/core/interfaces.go
type Mailer interface {
    Send(ctx context.Context, msg Email) error
    SendBulk(ctx context.Context, msgs []Email) error
}

type Email struct {
    To      []Address
    CC      []Address
    From    Address
    Subject string
    HTML    string  // rendered by templ
    Text    string  // plain text fallback
    ReplyTo *Address
}

type Address struct {
    Name  string
    Email string
}
```

**Drivers to implement:**

*SMTP driver* (`modules/mailer/drivers/smtp/`):
```go
type SMTPDriver struct { /* host, port, user, pass, tls config */ }
func (d *SMTPDriver) Send(ctx context.Context, msg core.Email) error { ... }
```
Use `net/smtp` from stdlib or `github.com/wneessen/go-mail` (better TLS support, actively maintained).

*Resend driver* (`modules/mailer/drivers/resend/`):
```go
type ResendDriver struct { APIKey string }
// POST https://api.resend.com/emails with JSON body
```
Resend has a Go SDK: `github.com/resend/resend-go/v2`.

*Log driver* (for dev/test — prints email to stdout, never sends):
```go
type LogDriver struct { Logger *slog.Logger }
func (d *LogDriver) Send(ctx context.Context, msg core.Email) error {
    d.Logger.Info("email sent", "to", msg.To, "subject", msg.Subject)
    return nil
}
```

**Config:**
```go
type MailConfig struct {
    Driver   string `env:"MAIL_DRIVER" env-default:"log"` // log, smtp, resend
    FromName string `env:"MAIL_FROM_NAME" env-default:"GoShip App"`
    FromAddr string `env:"MAIL_FROM_ADDRESS"`
    SMTP struct {
        Host string `env:"MAIL_SMTP_HOST"`
        Port int    `env:"MAIL_SMTP_PORT" env-default:"587"`
        User string `env:"MAIL_SMTP_USER"`
        Pass string `env:"MAIL_SMTP_PASS"`
        TLS  bool   `env:"MAIL_SMTP_TLS" env-default:"true"`
    }
    Resend struct {
        APIKey string `env:"MAIL_RESEND_API_KEY"`
    }
}
```

**What to do:**
1. Add `Mailer` interface and `Email`/`Address` types to `framework/core/interfaces.go`.
2. Create `modules/mailer/` with `module.go` (implements `core.Module`).
3. Create SMTP, Resend, and Log drivers.
4. Wire driver selection in module based on `config.Mail.Driver`.
5. Register mailer in `app/foundation/container.go` at `// ship:container:start` marker.
6. Add `MailConfig` to `config/config.go`.
7. Update `.env.example` with mail variables.

**Done when:** `container.Mailer.Send(ctx, email)` sends an email via the configured driver.
Log driver outputs to stdout in test/dev. SMTP and Resend drivers compile and have unit tests
(mock the network call). `ship verify` passes.

---

### U02 — Templ-based email templates

**Status:** `[ ] todo`
**Depends on:** U01 (mailer interface must exist)
**Files:** `app/views/email/` (new), `modules/mailer/render.go` (new)

**Context:** Email templates need to be reliable — missing variables cause broken emails, not
just broken pages. Templ's compile-time checking applies to email templates too. The rendering
pipeline converts a templ component into an HTML string for the mailer.

**Email template convention:**
```
app/views/email/
├── layout.templ          # shared email layout (header, footer, brand colors)
├── welcome.templ         # Welcome to {AppName}
├── password_reset.templ  # Reset your password
├── verify_email.templ    # Verify your email address
└── notifications/
    └── digest.templ      # Daily digest email
```

**Rendering helper:**
```go
// modules/mailer/render.go
func RenderEmail(ctx context.Context, component templ.Component) (html string, text string, err error) {
    var buf bytes.Buffer
    if err := component.Render(ctx, &buf); err != nil {
        return "", "", err
    }
    html = buf.String()
    text = stripHTML(html)  // simple HTML → plain text for fallback
    return html, text, nil
}
```

**Usage in controllers/services:**
```go
html, text, _ := mailer.RenderEmail(ctx, views.PasswordResetEmail(views.PasswordResetEmailData{
    UserName:  user.Name,
    ResetLink: resetURL,
    ExpiresIn: "1 hour",
}))
container.Mailer.Send(ctx, core.Email{
    To:      []core.Address{{Name: user.Name, Email: user.Email}},
    Subject: "Reset your password",
    HTML:    html,
    Text:    text,
})
```

**What to do:**
1. Create `app/views/email/layout.templ`: base email layout with inline CSS, responsive 600px
   container, brand header, footer with unsubscribe link slot.
2. Create `welcome.templ`, `password_reset.templ`, `verify_email.templ` using the layout.
3. Create `modules/mailer/render.go` with `RenderEmail` helper.
4. Create `app/commands/send_test_email.go` as a CLI command to send a test email
   (verifies mailer config is correct).
5. Run `templ generate` to verify no syntax errors.

**Done when:** Each email template renders to valid HTML. `RenderEmail` returns non-empty HTML
and text strings. The templates follow `docs/ui/convention.md` where applicable.

---

### U03 — Email preview routes (development only)

**Status:** `[ ] todo`
**Depends on:** U02 (email templates must exist)
**Files:** `app/web/controllers/mail_preview.go` (new), `app/router.go` (dev-only routes)

**Context:** The standard workflow for email development is: change template → send test email →
check inbox → repeat. This is slow and noisy. Email preview routes render the template directly
in the browser — instant feedback, zero email sending.

**Routes (development environment only):**
```
GET /dev/mail                    → lists all available email templates
GET /dev/mail/welcome            → renders welcome email in browser
GET /dev/mail/password-reset     → renders password reset email
GET /dev/mail/verify-email       → renders verify email
```

**Controller:**
```go
// app/web/controllers/mail_preview.go
type MailPreviewController struct { container *foundation.Container }

func (c *MailPreviewController) Index(ctx echo.Context) error {
    // renders a list of all preview routes
}

func (c *MailPreviewController) Welcome(ctx echo.Context) error {
    html, _, _ := mailer.RenderEmail(ctx.Request().Context(), views.WelcomeEmail(views.WelcomeEmailData{
        UserName: "Preview User",
        AppName:  c.container.Config.App.Name,
    }))
    return ctx.HTML(200, html)
}
```

**Router registration (dev only):**
```go
// app/router.go — inside a config guard
if container.Config.App.Environment == "development" {
    dev := e.Group("/dev")
    mailPreview := &controllers.MailPreviewController{container}
    dev.GET("/mail", mailPreview.Index)
    dev.GET("/mail/:template", mailPreview.Show)
}
```

**What to do:**
1. Create `app/web/controllers/mail_preview.go` with one handler per email template.
2. Add dev-only route group to `app/router.go` guarded by `config.App.Environment == "development"`.
3. The index handler renders an HTML page listing all preview URLs as clickable links.
4. Document in `docs/guides/02-development-workflows.md`: "Visit /dev/mail to preview email templates."

**Done when:** `GET /dev/mail/welcome` renders the welcome email in the browser. Routes do not
exist in production (verified by checking env guard). Index page lists all available previews.

---

## Group V — Scheduling & CLI Commands

### V01 — Cron job scheduling convention

**Status:** `[ ] todo`
**Depends on:** M03 A01 (container init), jobs module must be wired
**Files:** `app/schedules/schedules.go` (new), `app/foundation/container.go` (wire scheduler),
`cmd/worker/main.go` (start scheduler), `go.mod` (add robfig/cron)

**Context:** Background jobs in GoShip are one-off (enqueued on demand). Many apps need periodic
jobs: send daily digest at 9am, sync external data every 5 minutes, clean up expired sessions
nightly. `robfig/cron` (`github.com/robfig/cron/v3`) is the standard Go cron library.

**Convention:**
```go
// app/schedules/schedules.go
// This file registers all periodic scheduled jobs.
// Each schedule enqueues a job via the jobs adapter — it does NOT run logic inline.
// ship:schedules:start
func Register(s *cron.Cron, container *foundation.Container) {
    // Daily report at 9am UTC
    s.AddFunc("0 9 * * *", func() {
        container.Jobs.Enqueue(context.Background(), jobs.DailyReportJob{})
    })
    // Cleanup expired sessions every hour
    s.AddFunc("0 * * * *", func() {
        container.Jobs.Enqueue(context.Background(), jobs.CleanupSessionsJob{})
    })
}
// ship:schedules:end
```

**Key rule:** Schedules only enqueue jobs — they never contain business logic. This keeps the
cron goroutine fast and ensures all work goes through the job queue with its retry/monitoring.

**What to do:**
1. Add `github.com/robfig/cron/v3` to `go.mod`.
2. Create `app/schedules/schedules.go` with a `Register` function and ship:schedules markers.
3. Update `app/foundation/container.go`: add `Scheduler *cron.Cron` field, initialize in
   `NewContainer` with `cron.New(cron.WithSeconds())`, call `schedules.Register(c.Scheduler, c)`.
4. Update `cmd/worker/main.go`: call `container.Scheduler.Start()` after job workers start,
   and `container.Scheduler.Stop()` in the shutdown hook.
5. Create `ship make:schedule` CLI command: generates a named schedule entry inside the
   `// ship:schedules:start` / `:end` markers.
6. Update `docs/guides/05-jobs-module.md` with a Scheduled Jobs section.

**Done when:** Worker process starts the cron scheduler. A schedule registered in `schedules.go`
fires at the correct time and enqueues a job. `ship make:schedule DailyReport --cron "0 9 * * *"`
inserts a new entry at the marker. `go build ./...` passes.

---

### V02 — App-level CLI commands convention

**Status:** `[ ] todo`
**Depends on:** M03 A01 (container init)
**Files:** `app/commands/` (new directory), `app/commands/example.go` (scaffold example),
`cmd/cli/main.go` (new entrypoint), `tools/cli/ship/internal/commands/make_command.go` (new)

**Context:** Apps need custom scripts: import CSV data, backfill a new column, generate a report,
send a one-off email batch. Go's standard approach is `cmd/scripts/main.go` with duplicate
container initialization. GoShip should have a convention: `app/commands/` contains typed command
structs that get full DI container access, registered and run via `ship run:command`.

**Command interface:**
```go
// framework/command/command.go
type Command interface {
    Name() string
    Description() string
    Run(ctx context.Context, args []string) error
}
```

**Example command:**
```go
// app/commands/send_digest.go
type SendDigestCommand struct { Container *foundation.Container }

func (c *SendDigestCommand) Name() string        { return "send:digest" }
func (c *SendDigestCommand) Description() string { return "Send daily digest email to all subscribers" }
func (c *SendDigestCommand) Run(ctx context.Context, args []string) error {
    users, err := c.Container.Store.Users().FindSubscribed(ctx)
    if err != nil { return err }
    for _, u := range users {
        c.Container.Jobs.Enqueue(ctx, jobs.SendDigestJob{UserID: u.ID})
    }
    fmt.Printf("Enqueued digest for %d users\n", len(users))
    return nil
}
```

**Command runner entrypoint:**
```go
// cmd/cli/main.go
func main() {
    container := foundation.NewContainer()
    registry := command.NewRegistry(container)
    registry.Register(&commands.SendDigestCommand{Container: container})
    // ship:commands:start
    // ship:commands:end

    if err := registry.Run(os.Args[1:]); err != nil {
        log.Fatal(err)
    }
}
```

**ship CLI integration:**
```
ship run:command send:digest
ship run:command send:digest -- --dry-run
```
`ship run:command` shells out to `go run cmd/cli/main.go <command-name> <args>`.

**What to do:**
1. Create `framework/command/` package with `Command` interface and `Registry`.
2. Create `cmd/cli/main.go` with command runner.
3. Create `app/commands/` directory with `example.go` showing the pattern.
4. Create `app/commands/send_test_email.go` (useful: tests mailer config is working).
5. Add `ship run:command <name>` to ship CLI: execs `go run cmd/cli/main.go <name>`.
6. Add `ship make:command <Name>` to ship CLI: scaffolds a new command file with the interface
   implemented and registered at the `// ship:commands:start` marker.
7. Document in `docs/guides/02-development-workflows.md`.

**Done when:** `ship run:command send:digest` runs the command with full DI access.
`ship make:command BackfillUserStats` creates a new command file. `go build ./cmd/cli/` passes.

---

## Execution Order

**Layer 0 (no dependencies — run in parallel):**
- S01 (ship dev), S02 (GitHub Actions)
- T02 (slog), T03 (security headers), T04 (health check)
- U01 (mailer interface)

**Layer 1:**
- T01 (SQLite safety — needs M03 I01 complete first)
- U02 (email templates — needs U01)
- V01 (scheduling — needs container init from M03 A01)
- V02 (CLI commands — needs container init from M03 A01)

**Layer 2:**
- U03 (email previews — needs U02)

**External dependencies (from M03 that must be done first):**
- M03 A01 before V01, V02
- M03 I01 before T01
