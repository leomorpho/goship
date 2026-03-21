# GoShip — Modules & Capabilities

> Status note: `01-framework-plan.md` is the canonical roadmap. This file is supporting execution guidance.

**Status:** Active planning — tasks pickup-ready for any LLM agent
**Last updated:** 2026-03-08

**Reference docs (read before picking up any task):**
- `docs/roadmap/02-architecture-evolution.md` — module system design
- `docs/roadmap/03-atomic-tasks.md` — M03 task list (groups A–K, includes module interfaces)
- `docs/roadmap/06-dx-and-infrastructure.md` — M06 (mailer, slog, etc.)
- `docs/guides/01-ai-agent-guide.md` — conventions, safe change workflow

**Stack context:** Go 1.24, Echo v4, Templ, HTMX, Bob ORM, cleanenv config,
`ship module:add` for installation (M03 C03), module interface from M03 C01.

**Module structure (every module follows this layout):**
```
modules/<name>/
├── module.go          # ID(), Configure(), Migrations() — implements core.Module
├── service.go         # Business logic, exported API
├── store.go           # Storage interface
├── store_sql.go       # SQL implementation via Bob
├── routes.go          # Route registration — implements core.RoutableModule (optional)
├── views/web/         # Templ templates
├── db/migrations/     # SQL migration files
├── CLAUDE.md          # Agent context for this module
└── *_test.go
```

**Task format:** Self-contained. Full context, exact files, "done when" criterion.
Mark `[x]` before starting any dependent task.

**Group prefix:** W–AD (continues from M06's S–V).

---

## Key File Map (read before touching any task)

| Concern | File / Fact |
|---------|-------------|
| Module interface | `framework/core/interfaces.go` → `type Module interface { ID() string; Migrations() fs.FS }` |
| RoutableModule interface | `framework/core/interfaces.go` → `type RoutableModule interface { Module; RegisterRoutes(r Router) error }` |
| Core Router type | `framework/core/interfaces.go` → `type Router interface { Group, GET, POST, PUT, DELETE }` |
| Canonical module example | `modules/auth/module.go` — read this as the reference implementation |
| Auth module routes | `modules/auth/routes.go` — full route registration example |
| Modules directory | `modules/` — list it before creating new modules to avoid duplicating existing ones |
| Container wiring | `app/foundation/container.go` → `NewContainer()` with `// ship:container:start` / `// ship:container:end` marker at line ~95 |
| Router wiring | `app/router.go` → `// ship:routes:auth:start/end`, `// ship:routes:public:start/end` markers |
| Core interfaces (Mailer) | `framework/core/interfaces.go` → `core.Mailer`, `core.MailMessage`, `core.MailAddress` already defined |
| Existing mailer impl | `framework/repos/mailer/` — SMTP + Resend drivers already exist |
| Config struct | `config/config.go` → add new config fields as sub-structs (e.g., `OAuth OAuthConfig`) |
| App controllers | `app/web/controllers/` — for any app-layer route handlers |
| App views | `app/views/` — for app-layer templ components |
| Migrations dir | `db/migrate/migrations/` (not `db/migrations/`) — this is the actual path used by goose |
| Templ generate | `make templ-gen` |
| Test commands | `make test` (unit), `make test-integration` (Docker), `make e2e` (Playwright) |
| Ship verify | `ship verify` or `make verify` — run after every change |

---

## Group W — Auth Capabilities

### W01 — OAuth / Social Login module

**Status:** `[ ] todo`
**Depends on:** M03 C01 (module interface), M03 D01 (auth module must exist)
**Files:** `modules/auth/oauth.go` (new), `modules/auth/routes.go` (update),
`modules/auth/db/migrations/` (new: oauth_accounts table), `config/config.go`,
`go.mod` (`golang.org/x/oauth2`)

**Context:** Nearly every app needs social login. Password-only auth is a barrier to sign-up.
The OAuth module extends the existing `modules/auth` — it does not replace it. A user can have
multiple auth methods: email/password AND GitHub login linked to the same account.

**Database schema:**
```sql
-- modules/auth/db/migrations/00002_oauth_accounts.sql
CREATE TABLE oauth_accounts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,          -- "github", "google", "discord"
    provider_id TEXT NOT NULL,          -- the provider's user ID
    email       TEXT,                   -- email from provider (may differ from user.email)
    token       TEXT,                   -- encrypted access token
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, provider_id)
);
CREATE INDEX idx_oauth_accounts_user_id ON oauth_accounts(user_id);
```

**Provider configuration:**
```go
// config/config.go
type OAuthConfig struct {
    GitHub struct {
        ClientID     string `env:"OAUTH_GITHUB_CLIENT_ID"`
        ClientSecret string `env:"OAUTH_GITHUB_CLIENT_SECRET"`
    }
    Google struct {
        ClientID     string `env:"OAUTH_GOOGLE_CLIENT_ID"`
        ClientSecret string `env:"OAUTH_GOOGLE_CLIENT_SECRET"`
    }
    Discord struct {
        ClientID     string `env:"OAUTH_DISCORD_CLIENT_ID"`
        ClientSecret string `env:"OAUTH_DISCORD_CLIENT_SECRET"`
    }
}
// A provider is enabled if its ClientID is non-empty.
```

**Flow:**
```
GET  /auth/oauth/:provider          → redirect to provider's authorization URL
GET  /auth/oauth/:provider/callback → exchange code for token, upsert user + oauth_account,
                                      create session, redirect to home
```

**Account linking logic (critical):**
1. Provider returns email. Query `users` for that email.
2. If user exists: link the oauth_account to existing user (merge). Log them in.
3. If user doesn't exist: create new user from provider data (name, email, avatar). Create oauth_account. Log them in.
4. If provider_id already in oauth_accounts: just log them in (returning user).

**What to do:**
1. Add `golang.org/x/oauth2` to `go.mod`.
2. Create `modules/auth/oauth.go`:
   - `OAuthProvider` interface with `Name()`, `Config() *oauth2.Config`, `FetchUser(token) (*OAuthUser, error)`
   - Implement GitHub, Google, Discord providers
   - `OAuthService` with `HandleCallback(ctx, provider, code) (*User, error)` containing linking logic
3. Create migration `00002_oauth_accounts.sql`.
4. Add routes to `modules/auth/routes.go`:
   - `GET /auth/oauth/:provider` → generate state (CSRF), store in session, redirect
   - `GET /auth/oauth/:provider/callback` → validate state, exchange code, call `HandleCallback`
5. Update the login page templ view to show "Continue with GitHub / Google" buttons
   (only for providers that are enabled in config — check ClientID is non-empty).
6. Token storage: encrypt the access token at rest using the app's secret key.
7. Add OAuth config to `.env.example`.

**Done when:** GitHub OAuth login flow works end-to-end: clicking "Continue with GitHub"
redirects to GitHub, callback creates/links user, session is created, user lands on home.
Account linking works (same email = same user). CSRF state validation prevents open redirect.

---

### W02 — Two-Factor Authentication (TOTP) module

**Status:** `[ ] todo`
**Depends on:** M03 D01 (auth module), W01 is not required (2FA is independent of OAuth)
**Files:** `modules/2fa/` (new module), `modules/auth/db/migrations/` (2fa columns),
`go.mod` (`github.com/pquerna/otp`)

**Context:** TOTP (RFC 6238) — Time-based One-Time Passwords. Compatible with Google Authenticator,
Authy, 1Password, Bitwarden, and any standard TOTP app. Installable via `ship module:add 2fa`.
Once installed, users can optionally enable 2FA from their profile. Admins can enforce it.

**Database additions (migration on auth module's users table):**
```sql
ALTER TABLE users ADD COLUMN totp_secret TEXT;         -- encrypted TOTP secret, null = not enabled
ALTER TABLE users ADD COLUMN totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN totp_backup_codes TEXT;   -- JSON array of hashed backup codes
```

**Module structure:**
```
modules/2fa/
├── module.go           # implements core.Module
├── service.go          # GenerateSecret, ValidateCode, Enable, Disable, UseBackupCode
├── store.go            # storage interface
├── store_sql.go        # Bob SQL implementation
├── routes.go           # /profile/2fa/* routes
└── views/web/
    ├── setup.templ     # QR code + manual key + verification form
    ├── verify.templ    # "Enter your 6-digit code" during login
    └── backup_codes.templ  # Show backup codes after enabling
```

**Auth flow integration:**
After primary auth (password or OAuth) succeeds:
1. If `user.totp_enabled == true`: do NOT create session yet. Instead redirect to `/auth/2fa/verify`.
2. Store pending user ID in a short-lived signed cookie (5 minutes).
3. `POST /auth/2fa/verify` with code → validate → create full session → redirect home.
4. If code is a backup code (starts with `BK-`): validate against hashed list, invalidate used code.

**Setup flow (from profile settings):**
1. `GET /profile/2fa/setup` → generate TOTP secret, show QR code (as SVG, no external service),
   show manual entry key.
2. `POST /profile/2fa/verify-setup` → user enters first code to confirm they have the app working.
3. On success: save encrypted secret to DB, set `totp_enabled = true`, show backup codes.
4. `GET /profile/2fa/backup-codes` → regenerate backup codes (invalidates old ones).

**What to do:**
1. Add `github.com/pquerna/otp` to `go.mod`.
2. Create `modules/2fa/` with all files above.
3. Create `modules/2fa/service.go`:
   - `GenerateSecret(issuer, accountName string) (secret, qrCodeSVG string, err error)`
   - `ValidateCode(secret, code string) bool` — uses totp.Validate from pquerna/otp
   - `GenerateBackupCodes() []string` — generates 10 codes like `BK-XXXX-XXXX`
   - `HashBackupCode(code string) string` — bcrypt hash for storage
4. Encrypt TOTP secret at rest using app secret key (same encryption helper used for OAuth tokens).
5. Integrate auth flow: after `POST /login` success, check `totp_enabled`, redirect if true.
6. Create setup views with QR code rendered as inline SVG (no external deps).

**Done when:** User can enable 2FA from profile. QR code works with Google Authenticator.
Login with 2FA enabled requires the 6-digit code. Backup codes work. Invalid codes are rejected.
`ship module:add 2fa` installs the module. `ship verify` passes.

---

## Group X — AI Integration Module

### X01 — `modules/ai` core: provider interface + Anthropic adapter

**Status:** `[ ] todo`
**Depends on:** M03 C01 (module interface), U01 (SSE from M06 for streaming)
**Files:** `modules/ai/` (new module), `go.mod` (anthropic-sdk-go)

**Context:** GoShip is LLM-forward — it should make building LLM features trivial. `modules/ai`
provides a provider-agnostic interface for text completion, with Anthropic as the primary adapter.
Every app built on GoShip should be able to add AI features in minutes, not days.

**Core provider interface:**
```go
// modules/ai/provider.go
type Provider interface {
    Complete(ctx context.Context, req Request) (*Response, error)
    Stream(ctx context.Context, req req Request) (<-chan Token, error)
}

type Request struct {
    Model       string
    System      string
    Messages    []Message
    MaxTokens   int
    Temperature float32
    Schema      any    // if set, requests structured JSON output bound to this type
    Tools       []Tool // function calling tools
}

type Message struct {
    Role    string // "user" or "assistant"
    Content string
}

type Response struct {
    Content    string
    InputTokens  int
    OutputTokens int
    Model      string
    FinishReason string
}

type Token struct {
    Content string
    Done    bool
    Error   error
}
```

**Anthropic adapter** (`modules/ai/drivers/anthropic/`):
```go
// Uses github.com/anthropics/anthropic-sdk-go
type AnthropicDriver struct {
    Client *anthropic.Client
    DefaultModel string
}

func (d *AnthropicDriver) Complete(ctx context.Context, req ai.Request) (*ai.Response, error) {
    // Map ai.Request → anthropic.MessageNewParams
    // Return ai.Response from anthropic.Message
}

func (d *AnthropicDriver) Stream(ctx context.Context, req ai.Request) (<-chan ai.Token, error) {
    // Use anthropic streaming API
    // Fan tokens out to channel
}
```

**Structured output:**
If `req.Schema != nil`, instruct the model to output JSON matching the schema. Use Anthropic's
tool-use with a single tool that forces structured output. Unmarshal response into `req.Schema`
using `encoding/json`. Return error if JSON is invalid.

**Config:**
```go
type AIConfig struct {
    Driver  string `env:"AI_DRIVER" env-default:"anthropic"` // anthropic, openai, openrouter
    Anthropic struct {
        APIKey       string `env:"ANTHROPIC_API_KEY"`
        DefaultModel string `env:"ANTHROPIC_DEFAULT_MODEL" env-default:"claude-haiku-4-5-20251001"`
    }
}
```

**Model constants:**
```go
// modules/ai/models.go
const (
    // Anthropic
    ClaudeOpus4    = "claude-opus-4-6"
    ClaudeSonnet4  = "claude-sonnet-4-6"
    ClaudeHaiku4   = "claude-haiku-4-5-20251001"
)
```

**What to do:**
1. Add `github.com/anthropics/anthropic-sdk-go` to `go.mod`.
2. Create `modules/ai/` with `module.go`, `provider.go` (interface), `models.go` (constants).
3. Create `modules/ai/drivers/anthropic/driver.go` — Anthropic adapter implementation.
4. Create `modules/ai/service.go` — wraps provider with rate limiting, logging, error handling.
5. Wire in `module.go` via config.
6. Add `AIConfig` to `config/config.go`.
7. Write unit tests using a mock provider (no real API calls in tests).

**Done when:** `container.AI.Complete(ctx, ai.Request{Model: ai.ClaudeHaiku4, Messages: [...]})`
returns a response. Streaming returns a channel of tokens. Structured output unmarshals correctly.
Mock provider works for testing.

---

### X02 — OpenAI and OpenRouter adapters

**Status:** `[ ] todo`
**Depends on:** X01 (AI interface must exist)
**Files:** `modules/ai/drivers/openai/driver.go` (new), `modules/ai/drivers/openrouter/driver.go` (new),
`config/config.go` (update AIConfig), `go.mod`

**Context:** OpenAI is the most common LLM provider. OpenRouter is a unified API gateway that
provides access to 200+ models (Claude, GPT-4, Gemini, Llama, Mistral, etc.) through a single
OpenAI-compatible endpoint — you configure one API key and can switch models by name. Supporting
OpenRouter means supporting virtually every major model with zero additional adapters.

**OpenAI adapter** (`modules/ai/drivers/openai/`):
```go
// Uses github.com/sashabaranov/go-openai — the standard Go OpenAI client
type OpenAIDriver struct {
    Client *openai.Client
    DefaultModel string
}
// Maps ai.Request → openai.ChatCompletionRequest
// Streaming: openai.CreateChatCompletionStream → fan to ai.Token channel
```

**OpenRouter adapter** (`modules/ai/drivers/openrouter/`):
OpenRouter exposes an OpenAI-compatible API — same endpoints, same request/response format.
The only differences: base URL is `https://openrouter.ai/api/v1`, and there are optional headers
(`HTTP-Referer`, `X-Title`) for attribution in OpenRouter's dashboard.

```go
// OpenRouter reuses the OpenAI adapter with a custom base URL:
type OpenRouterDriver struct {
    *openai.OpenAIDriver  // embed and override base URL
}

func NewOpenRouterDriver(apiKey string) *OpenRouterDriver {
    client := openai.NewClientWithConfig(openai.ClientConfig{
        BaseURL: "https://openrouter.ai/api/v1",
        AuthToken: apiKey,
        HTTPClient: &http.Client{Timeout: 60 * time.Second},
    })
    return &OpenRouterDriver{OpenAIDriver: &openai.OpenAIDriver{Client: client}}
}
```

**OpenRouter model names** (pass through as-is to the API):
```go
// modules/ai/models.go — add OpenRouter model name constants
const (
    // Via OpenRouter
    ORClaudeOpus4   = "anthropic/claude-opus-4-6"
    ORClaudeHaiku4  = "anthropic/claude-haiku-4-5-20251001"
    ORGPTo4Mini     = "openai/gpt-4o-mini"
    ORGeminiFlash   = "google/gemini-flash-1.5"
    ORLlama3370B    = "meta-llama/llama-3.3-70b-instruct"
)
```

**Config additions:**
```go
type AIConfig struct {
    Driver string `env:"AI_DRIVER" env-default:"anthropic"`
    // ...Anthropic config (from X01)...
    OpenAI struct {
        APIKey       string `env:"OPENAI_API_KEY"`
        DefaultModel string `env:"OPENAI_DEFAULT_MODEL" env-default:"gpt-4o-mini"`
    }
    OpenRouter struct {
        APIKey       string `env:"OPENROUTER_API_KEY"`
        DefaultModel string `env:"OPENROUTER_DEFAULT_MODEL" env-default:"anthropic/claude-haiku-4-5-20251001"`
        SiteURL      string `env:"OPENROUTER_SITE_URL"`  // optional, for attribution
        SiteName     string `env:"OPENROUTER_SITE_NAME"` // optional, for attribution
    }
}
```

**What to do:**
1. Add `github.com/sashabaranov/go-openai` to `go.mod`.
2. Create `modules/ai/drivers/openai/driver.go` — OpenAI adapter implementing `ai.Provider`.
3. Create `modules/ai/drivers/openrouter/driver.go` — wraps OpenAI adapter with OpenRouter base URL.
4. Update driver selection in `modules/ai/module.go` to handle `openai` and `openrouter` drivers.
5. Update `config/config.go` with OpenAI and OpenRouter config sections.
6. Update `.env.example` with all AI provider variables.
7. Add model constants to `modules/ai/models.go`.
8. Unit tests with mock HTTP server (record/replay pattern — no live API calls).

**Done when:** Setting `AI_DRIVER=openrouter` + `OPENROUTER_API_KEY=...` routes all AI calls
through OpenRouter. Model constants work as model identifiers. OpenAI driver works with OpenAI
directly. All three drivers implement `ai.Provider` and pass the same interface tests.

---

### X03 — AI streaming via SSE + HTMX

**Status:** `[ ] todo`
**Depends on:** X01 (AI provider), SSE infrastructure (see W03 — SSE module)
**Files:** `modules/ai/stream_handler.go` (new), example in `app/web/controllers/`

**Context:** Streaming LLM responses to the browser is the standard AI UX pattern — users see
tokens appear as they're generated. HTMX + SSE is the natural GoShip approach: a server-sent
event stream that appends tokens to a target element. No custom JavaScript required.

**Stream endpoint pattern:**
```go
// modules/ai/stream_handler.go
// StreamCompletion writes an SSE stream of AI tokens to the response.
// Compatible with HTMX hx-ext="sse" extension.
func StreamCompletion(ctx context.Context, w http.ResponseWriter, req ai.Request, provider ai.Provider) error {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

    tokens, err := provider.Stream(ctx, req)
    if err != nil { return err }

    flusher, ok := w.(http.Flusher)
    if !ok { return errors.New("streaming not supported") }

    for token := range tokens {
        if token.Error != nil { break }
        fmt.Fprintf(w, "data: %s\n\n", token.Content)
        flusher.Flush()
        if token.Done { break }
    }
    fmt.Fprintf(w, "event: done\ndata: \n\n")
    flusher.Flush()
    return nil
}
```

**Templ usage:**
```templ
// HTMX polls SSE endpoint, appending each token to #response
<div hx-ext="sse" sse-connect="/ai/stream?prompt={prompt}">
    <div id="response" sse-swap="message"></div>
</div>
```

**What to do:**
1. Create `modules/ai/stream_handler.go` with `StreamCompletion` helper.
2. Create an example controller in `app/web/controllers/ai_demo.go` demonstrating the pattern.
3. Create a simple demo page templ template using the HTMX SSE pattern.
4. Register the demo route behind auth in `app/router.go` (dev/demo only, easy to remove).
5. Document the SSE streaming pattern in a new `docs/guides/06-ai-module.md`.

**Done when:** A browser request to the stream endpoint receives tokens one by one via SSE.
HTMX appends tokens to the target element in real time. The stream ends with a `done` event.

---

### X04 — AI conversation history persistence

**Status:** `[ ] todo`
**Depends on:** X01 (AI provider), T02 (slog)
**Files:** `modules/ai/db/migrations/` (new tables), `modules/ai/conversation_store.go` (new),
`modules/ai/conversation_service.go` (new)

**Context:** Stateless AI calls are sufficient for one-off completions. For chatbot-style features,
conversation history must be persisted — the user's messages and AI responses stored in DB,
sent as context with each new message.

**Database schema:**
```sql
-- modules/ai/db/migrations/00001_ai_conversations.sql
CREATE TABLE ai_conversations (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model      TEXT NOT NULL,
    title      TEXT,           -- auto-generated from first message, or user-set
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ai_messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,       -- "user" or "assistant"
    content         TEXT NOT NULL,
    input_tokens    INTEGER,             -- for assistant messages: token usage
    output_tokens   INTEGER,
    model           TEXT,               -- model used for this response
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_ai_messages_conversation ON ai_messages(conversation_id, created_at);
```

**Conversation service:**
```go
type ConversationService struct { store ConversationStore; provider ai.Provider }

func (s *ConversationService) SendMessage(ctx context.Context, convID int64, userMessage string) (<-chan ai.Token, error) {
    // 1. Load conversation history from DB
    // 2. Append user message to history, save to DB
    // 3. Call provider.Stream with full history as context
    // 4. Collect response tokens, save assistant message to DB when stream ends
    // 5. Return token channel for streaming to client
}
```

**What to do:**
1. Create migration files for `ai_conversations` and `ai_messages`.
2. Create `modules/ai/conversation_store.go` (interface) and `conversation_store_sql.go` (Bob impl).
3. Create `modules/ai/conversation_service.go` with `SendMessage`, `ListConversations`, `GetHistory`.
4. Wire into module in `module.go`.
5. Write tests using in-memory SQLite test DB.

**Done when:** A conversation persists across requests. `SendMessage` loads history, sends to AI,
saves response. Token/cost data is recorded per message. Tests pass.

---

## Group Y — Data Patterns

### Y01 — Domain events system

**Status:** `[ ] todo`
**Depends on:** nothing (standalone framework addition)
**Files:** `framework/events/` (new package), `framework/events/types/` (new),
`app/foundation/container.go` (wire event bus)

**Context:** Modules need to react to things that happen in other modules without direct imports
(which would create circular dependencies). Domain events solve this: the auth module publishes
`UserLoggedIn` — the audit module subscribes and records it — without auth importing audit or
vice versa. Both modules only import `framework/events`.

**Event bus:**
```go
// framework/events/bus.go
type Bus struct { mu sync.RWMutex; handlers map[string][]HandlerFunc }

type HandlerFunc func(ctx context.Context, event any) error

func (b *Bus) Publish(ctx context.Context, event any) error {
    // Get all handlers for reflect.TypeOf(event).String()
    // Call each handler synchronously (same goroutine)
    // Return first error, or nil
}

func Subscribe[T any](b *Bus, handler func(ctx context.Context, event T) error) {
    typeName := reflect.TypeOf(*new(T)).String()
    b.handlers[typeName] = append(b.handlers[typeName], func(ctx context.Context, e any) error {
        return handler(ctx, e.(T))
    })
}
```

**Event types** (defined in `framework/events/types/` — shared, no module imports):
```go
// framework/events/types/auth.go
type UserRegistered struct { UserID int64; Email string; At time.Time }
type UserLoggedIn   struct { UserID int64; IP string; At time.Time }
type UserLoggedOut  struct { UserID int64; At time.Time }
type PasswordChanged struct { UserID int64; At time.Time }

// framework/events/types/subscription.go
type SubscriptionCreated struct { UserID int64; Plan string; At time.Time }
type SubscriptionCancelled struct { UserID int64; At time.Time }
```

**Async variant** — for non-critical side effects, route through jobs module:
`PublishAsync(...)` now enqueues an `AsyncEnvelope` under the explicit job name
`framework.events.publish`; the worker-side bridge decodes that envelope and republishes the
supported shared event type into the local bus.

**What to do:**
1. Create `framework/events/bus.go` with `Bus`, `Publish`, `Subscribe[T]`.
2. Create `framework/events/types/` with common event types for auth, subscriptions, profile.
3. Add `EventBus *events.Bus` to container. Initialize in `NewContainer`.
4. Update `modules/auth` to publish `UserRegistered`, `UserLoggedIn`, `UserLoggedOut` on relevant actions.
5. Document the pattern in a new `docs/guides/07-domain-events.md`.
6. Create `ship make:event TypeName` CLI command: scaffolds a new event type in `framework/events/types/`.

**Done when:** Auth module publishes events. A subscriber registered in `app/foundation/container.go`
receives them. `Subscribe[events.UserLoggedIn]` works with type inference. Async variant enqueues
correctly. Tests verify publish → handler called.

---

### Y02 — Soft deletes convention

**Status:** `[ ] todo`
**Depends on:** nothing (parallel)
**Files:** `framework/softdelete/` (new package), `db/queries/` (update query patterns),
`tools/cli/ship/internal/commands/make_migration.go` (update scaffold)

**Context:** Many resources need to be "deleted" without actually removing the row — for audit
history, recovery, or cascading effects. The `deleted_at DATETIME` pattern is universal.
Without a framework convention, each developer implements it differently.

**Convention:**
1. Any table supporting soft deletes has a `deleted_at DATETIME` column (nullable).
2. `deleted_at IS NULL` = active record; `deleted_at IS NOT NULL` = soft deleted.
3. All Bob queries for soft-deletable resources must scope to `deleted_at IS NULL` by default.
4. Hard delete is explicit: `store.HardDelete(ctx, id)`.

**Framework helpers:**
```go
// framework/softdelete/softdelete.go
const Column = "deleted_at"

// SoftDeleteClause returns a Bob WHERE clause for non-deleted records
func SoftDeleteClause() bob.Expression {
    return bob.Raw("deleted_at IS NULL")
}

// IsDeleted checks if a struct with DeletedAt field is soft-deleted
func IsDeleted(v any) bool {
    rv := reflect.ValueOf(v)
    f := rv.FieldByName("DeletedAt")
    if !f.IsValid() { return false }
    return !f.IsNil()  // *time.Time: non-nil = deleted
}
```

**Migration scaffold addition:**
```
ship make:migration AddSoftDelete --table posts
```
Generates:
```sql
ALTER TABLE posts ADD COLUMN deleted_at DATETIME;
CREATE INDEX idx_posts_deleted_at ON posts(deleted_at);
```

**`ship doctor` check:**
Any table with a `deleted_at` column where queries in `db/queries/` do NOT include
`deleted_at IS NULL` in the WHERE clause → warning. (Static analysis of SQL files.)

**What to do:**
1. Create `framework/softdelete/` package with helpers.
2. Update `ship make:migration` to support `--soft-delete` flag: generates `deleted_at` column.
3. Add `ship doctor` check: warn if SQL query on a soft-delete table lacks the IS NULL filter.
4. Create `ship make:restore <Model>` command: generates a restore handler + route.
5. Update admin panel (M03 J tasks) to show a "Trash" view with restore/hard-delete actions.
6. Document in `docs/guides/` as a pattern guide.

**Done when:** `ship make:migration AddSoftDeleteToOrders --table orders` generates correct SQL.
Doctor warns on unfiltered queries. Framework helpers compile and have tests. Admin panel shows
trash view (if admin module is installed).

---

### Y03 — Feature flags module

**Status:** `[ ] todo`
**Depends on:** M03 C01 (module interface), M03 I03 (Otter cache — for caching flag values)
**Files:** `modules/flags/` (new module)

**Context:** Feature flags enable: staged rollouts (enable for 10% of users), A/B testing,
instant rollback without deploy (toggle off), and safe LLM-generated feature development
(ship behind a flag, verify, then enable globally).

**Database schema:**
```sql
-- modules/flags/db/migrations/00001_feature_flags.sql
CREATE TABLE feature_flags (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    key         TEXT NOT NULL UNIQUE,   -- "new_checkout_flow"
    enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    rollout_pct INTEGER NOT NULL DEFAULT 0,    -- 0-100: percentage of users
    user_ids    TEXT,                          -- JSON array of specific user IDs
    description TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Service API:**
```go
type FlagService struct { store FlagStore; cache core.Cache }

// Enabled returns true if the flag is on for the given context.
// Checks: global enabled → user-specific → rollout percentage.
func (s *FlagService) Enabled(ctx context.Context, key string, userID ...int64) bool {
    flag := s.cache.Get(key) // check cache first
    if flag == nil {
        flag = s.store.Find(ctx, key)
        s.cache.Set(key, flag, 5*time.Minute)
    }
    if !flag.Enabled { return false }
    if len(userID) > 0 {
        if flag.IsUserTargeted(userID[0]) { return true }
        return flag.InRollout(userID[0])  // deterministic hash: same user always gets same bucket
    }
    return flag.Rollout == 100
}
```

**Rollout determinism:** use `hash(userID + flagKey) % 100 < rollout_pct`. Same user always
sees the same variant — no flickering between requests.

**Templ helper:**
```go
// In controllers: pass flag state to viewmodel
type CheckoutPage struct {
    UseNewCheckout bool
}
// In controller:
page.UseNewCheckout = container.Flags.Enabled(ctx, "new_checkout_flow", userID)
```

**What to do:**
1. Create `modules/flags/` with standard module structure.
2. Implement `FlagService` with `Enabled`, `Create`, `Update`, `Delete`.
3. Implement rollout percentage with deterministic hashing.
4. Cache flag values in Otter (5-minute TTL) to avoid per-request DB queries.
5. Admin panel integration: list flags, toggle enabled, edit rollout %, set target user IDs.
6. `ship make:flag <key> --description "..."` creates a DB seed for the flag.
7. Wire into container.

**Done when:** `container.Flags.Enabled(ctx, "my_flag")` returns correct value based on DB.
Cache prevents repeated DB reads. Rollout is deterministic (same user, same result).
Toggle in admin panel takes effect within cache TTL. `ship verify` passes.

---

### Y04 — Audit log module

**Status:** `[ ] todo`
**Depends on:** Y01 (domain events — audit log subscribes to events)
**Files:** `modules/auditlog/` (new module)

**Context:** Every SaaS app eventually needs to know: who changed what, when, from where.
Audit logs are especially important for compliance (GDPR, SOC2, HIPAA) and debugging production
incidents. The audit module subscribes to domain events — no manual instrumentation in business logic.

**Database schema:**
```sql
-- modules/auditlog/db/migrations/00001_audit_logs.sql
CREATE TABLE audit_logs (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER REFERENCES users(id) ON DELETE SET NULL,
    action        TEXT NOT NULL,        -- "user.login", "post.created", "subscription.cancelled"
    resource_type TEXT,                 -- "post", "user", "subscription"
    resource_id   TEXT,                 -- the ID of the affected resource
    changes       TEXT,                 -- JSON: {"before": {...}, "after": {...}}
    ip_address    TEXT,
    user_agent    TEXT,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id, created_at);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
```

**Service:**
```go
type AuditService struct { store AuditStore }

func (s *AuditService) Record(ctx context.Context, action, resourceType, resourceID string, changes any) error {
    userID := auth.UserIDFromContext(ctx)   // from session middleware
    ip := ipFromContext(ctx)               // from request middleware
    return s.store.Insert(ctx, AuditLog{
        UserID: userID, Action: action,
        ResourceType: resourceType, ResourceID: resourceID,
        Changes: toJSON(changes), IP: ip,
    })
}
```

**Event subscriptions (wired in module.go):**
```go
events.Subscribe[types.UserLoggedIn](bus, func(ctx context.Context, e types.UserLoggedIn) error {
    return auditSvc.Record(ctx, "user.login", "user", strconv.FormatInt(e.UserID, 10), nil)
})
events.Subscribe[types.PasswordChanged](bus, func(ctx context.Context, e types.PasswordChanged) error {
    return auditSvc.Record(ctx, "user.password_changed", "user", strconv.FormatInt(e.UserID, 10), nil)
})
```

**Admin panel integration:** List audit logs per user and per resource. Filterable by action type and date range. Non-editable (append-only).

**What to do:**
1. Create `modules/auditlog/` with standard module structure.
2. Implement `AuditService.Record`.
3. In `module.go`, subscribe to all relevant domain event types from `framework/events/types/`.
4. Admin panel: `GET /admin/audit-logs` with user/action/resource filters.
5. Provide `AuditService.Record` as a public API for manual instrumentation in app code.

**Done when:** Login event automatically creates an audit log entry. Admin panel shows filterable
log. `Record` can be called manually from controllers. Append-only (no update/delete routes).

---

## Group Z — API & Real-time

### Z01 — First-class SSE (Server-Sent Events) support

**Status:** `[ ] todo`
**Depends on:** nothing (parallel)
**Files:** `framework/sse/` (new package), `framework/pubsub/` (integrate with existing pubsub)

**Context:** HTMX's `hx-ext="sse"` extension makes SSE the natural real-time primitive for GoShip.
Without a framework abstraction, SSE is verbose: manually set headers, cast to `http.Flusher`,
handle disconnects, implement fan-out. The framework should make this one function call.

**Design intent — real-time collaborative UX, not local-first:**
GoShip's target is apps that feel live and collaborative while keeping the server as the source
of truth. The pattern: client writes via HTMX form → server applies → SSE broadcasts the change
to all subscribers → other clients' UIs update without a page reload. This covers the 80% of
"real-time" use cases without the complexity of offline writes or CRDT conflict resolution.

True local-first (offline writes, conflict resolution, client-owned data) is a different
architecture that requires a dedicated sync layer (InstantDB, ElectricSQL). GoShip doesn't
try to compete there. If a GoShip app needs local-first for a specific feature, the recommended
pattern is: GoShip handles auth, business logic, and server rendering; a separate InstantDB
namespace handles the collaborative/offline feature.

**The Hub is the core primitive** — design it with broadcast-to-all-subscribers as the primary
use case, not an afterthought. Topic naming convention: `<resource>:<id>` (e.g. `post:42`,
`user:7`, `room:chat-general`). A single broadcast on `post:42` updates every open browser
tab viewing that post simultaneously.

**Framework package:**
```go
// framework/sse/sse.go

type Stream struct {
    w       http.ResponseWriter
    flusher http.Flusher
    ctx     context.Context
}

// New creates an SSE stream from an Echo context. Sets required headers.
func New(c echo.Context) (*Stream, error) {
    flusher, ok := c.Response().Writer.(http.Flusher)
    if !ok { return nil, errors.New("streaming unsupported") }
    c.Response().Header().Set("Content-Type", "text/event-stream")
    c.Response().Header().Set("Cache-Control", "no-cache")
    c.Response().Header().Set("Connection", "keep-alive")
    c.Response().Header().Set("X-Accel-Buffering", "no")
    return &Stream{w: c.Response().Writer, flusher: flusher, ctx: c.Request().Context()}, nil
}

// Send sends a named event with data.
func (s *Stream) Send(event, data string) error {
    if s.ctx.Err() != nil { return s.ctx.Err() } // client disconnected
    fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, data)
    s.flusher.Flush()
    return nil
}

// SendMessage sends a generic "message" event (HTMX default).
func (s *Stream) SendMessage(data string) error { return s.Send("message", data) }

// Wait blocks until client disconnects.
func (s *Stream) Wait() { <-s.ctx.Done() }
```

**Topic-based fan-out (integrates with pubsub adapter):**
```go
// framework/sse/hub.go
type Hub struct { mu sync.RWMutex; topics map[string][]chan string }

func (h *Hub) Subscribe(topic string) (ch chan string, unsubscribe func()) { ... }
func (h *Hub) Publish(topic string, data string) { ... }
```

**Usage in a controller:**
```go
func (c *NotifController) Stream(ctx echo.Context) error {
    stream, err := sse.New(ctx)
    if err != nil { return err }

    userID := auth.UserID(ctx)
    ch, unsub := container.SSEHub.Subscribe(fmt.Sprintf("user:%d", userID))
    defer unsub()

    for {
        select {
        case msg := <-ch:
            stream.SendMessage(msg)
        case <-ctx.Request().Context().Done():
            return nil
        }
    }
}
```

**What to do:**
1. Create `framework/sse/sse.go` with `Stream` struct and methods.
2. Create `framework/sse/hub.go` with topic-based `Hub`. The Hub must support:
   - `Subscribe(topic string) (ch chan string, unsubscribe func())` — per-connection subscription
   - `Publish(topic string, data string)` — broadcast to ALL subscribers on the topic (this is
     the primary use case: one write → all open browser tabs for that resource update)
   - `PublishHTML(topic string, component templ.Component)` — render a templ component and
     broadcast the HTML string. This is the natural HTMX integration: server renders the updated
     fragment, pushes it via SSE, HTMX swaps it in on every subscribed client.
3. Add `SSEHub *sse.Hub` to container, initialize in `NewContainer`.
4. Create example: a shared counter page where incrementing on one tab updates all other tabs.
   This demonstrates the intended GoShip real-time pattern end-to-end.
5. Write tests: verify headers are set, verify Send writes correct SSE format, verify Publish
   delivers to all current subscribers, verify unsubscribe removes the channel, verify Wait
   returns on context cancellation (no goroutine leak).

**Done when:** A controller can stream SSE events in 5 lines of code. `Hub.Publish("post:42", html)`
delivers to every open connection subscribed to `post:42`. Unsubscribed connections receive nothing.
Client disconnect cleanup works (no goroutine leak). The shared counter example works in a browser.

---

### Z02 — JSON API pattern and typed response helpers

**Status:** `[ ] todo`
**Depends on:** M05 O01 (route contracts)
**Files:** `framework/api/` (new package), owning module/controller DTO/response packages

**Context:** Many GoShip apps need both HTML views (for the web UI) and JSON responses (for
mobile clients or third-party integrations). The pattern should be: one handler, two representations,
zero duplication. Route contracts already define the data shape — JSON is just another rendering.

**Response envelope:**
```go
// framework/api/response.go
type Response[T any] struct {
    Data   T             `json:"data"`
    Meta   *Meta         `json:"meta,omitempty"`
    Errors []APIError    `json:"errors,omitempty"`
}

type Meta struct {
    Page    int `json:"page,omitempty"`
    PerPage int `json:"per_page,omitempty"`
    Total   int `json:"total,omitempty"`
}

type APIError struct {
    Field   string `json:"field,omitempty"`
    Message string `json:"message"`
    Code    string `json:"code"`
}

// OK sends a successful JSON response.
func OK[T any](c echo.Context, data T) error {
    return c.JSON(http.StatusOK, Response[T]{Data: data})
}

// Fail sends an error JSON response.
func Fail(c echo.Context, status int, errors ...APIError) error {
    return c.JSON(status, Response[struct{}]{Errors: errors})
}
```

**Content negotiation helper:**
```go
// framework/api/negotiate.go
// IsAPIRequest returns true if the client prefers JSON.
func IsAPIRequest(c echo.Context) bool {
    accept := c.Request().Header.Get("Accept")
    return strings.Contains(accept, "application/json") ||
           strings.HasPrefix(c.Path(), "/api/")
}
```

**Dual-response controller pattern:**
```go
func (pc *PostController) Show(c echo.Context) error {
    post, err := pc.store.Find(ctx, id)
    if err != nil { return err }

    if api.IsAPIRequest(c) {
        return api.OK(c, contracts.PostResponse{ID: post.ID, Title: post.Title})
    }
    return render(c, views.PostShow(toPostVM(post)))
}
```

**API versioning:** Route groups in `app/router.go`:
```go
v1 := e.Group("/api/v1")  // ship:routes:api:v1:start
// ship:routes:api:v1:end
```

**What to do:**
1. Create `framework/api/response.go` with typed response helpers.
2. Create `framework/api/negotiate.go` with `IsAPIRequest`.
3. Create `framework/api/errors.go` with common error constructors (`NotFound`, `Unauthorized`, `Validation`).
4. Add API route group markers to `app/router.go`.
5. Update `docs/guides/` with a "Building an API" guide.
6. `ship doctor` check: API routes should return JSON (warn if API-prefixed route renders templ).

**Done when:** `api.OK(c, data)` sends correctly enveloped JSON. `IsAPIRequest` correctly detects
API clients. Common error responses use the standard error format. `ship verify` passes.

---

### Z03 — OpenAPI spec generation from route contracts

**Status:** `[x] removed`
**Depends on:** none
**Files:** n/a

**Context:** The prior `ship api:spec` and `app/contracts`-driven OpenAPI flow was intentionally
removed during the app-minimalization cleanup stream.

**Current direction:**
1. Keep request DTO ownership local to controllers/modules.
2. Keep core `ship` surface focused on runtime/dev tooling.
3. Revisit OpenAPI generation only if a new dedicated design is approved.

---

## Group AA — Testing Infrastructure

### AA01 — Test data factory system

**Status:** `[ ] todo`
**Depends on:** M03 I04 (in-memory test DB must exist)
**Files:** `framework/factory/` (new package), `tests/factories/` (new directory),
`tests/factories/user_factory.go` (example)

**Context:** Every test that needs database state requires boilerplate setup. Agents writing tests
copy this boilerplate inconsistently, creating fragile tests. A factory system provides: typed
default values, trait overrides, DB insertion, and sequence generation. This is the single
highest-impact testing improvement for LLM agent reliability.

**Framework package:**
```go
// framework/factory/factory.go
type Factory[T any] struct {
    defaults  func() T
    afterBuild []func(*T)
}

func New[T any](defaults func() T) *Factory[T] {
    return &Factory[T]{defaults: defaults}
}

// Build creates a T with defaults, applying overrides. Does NOT insert to DB.
func (f *Factory[T]) Build(overrides ...func(*T)) T {
    v := f.defaults()
    for _, fn := range overrides { fn(&v) }
    for _, fn := range f.afterBuild { fn(&v) }
    return v
}

// Create builds and inserts into DB. Returns the created record (with DB-assigned ID).
func (f *Factory[T]) Create(t testing.TB, db *sql.DB, overrides ...func(*T)) T {
    t.Helper()
    v := f.Build(overrides...)
    // insert v into DB using reflection to build INSERT statement
    // set ID field from LastInsertId
    return v
}

// Sequence generates unique values per test run.
var seq = atomic.Int64{}
func Sequence(prefix string) string { return fmt.Sprintf("%s-%d", prefix, seq.Add(1)) }
```

**Example factories:**
```go
// tests/factories/user_factory.go
package factories

var User = factory.New(func() db.User {
    return db.User{
        Name:     "Test User",
        Email:    factory.Sequence("user") + "@example.com",
        Password: "$2a$10$...",  // pre-hashed "password"
        Role:     "member",
        CreatedAt: time.Now(),
    }
})

// traits
func WithAdminRole(u *db.User) { u.Role = "admin" }
func WithEmail(email string) func(*db.User) { return func(u *db.User) { u.Email = email } }
```

**Usage in tests:**
```go
func TestAdminRoute(t *testing.T) {
    db := testdb.Open(t)  // in-memory SQLite
    admin := factories.User.Create(t, db, factories.WithAdminRole)
    regular := factories.User.Create(t, db)

    // test admin-only route with admin user...
}
```

**What to do:**
1. Create `framework/factory/factory.go` with generic `Factory[T]` struct.
2. Implement `Build`, `Create` (with reflection-based INSERT), `Sequence`.
3. Create `tests/factories/` directory.
4. Create `tests/factories/user_factory.go` with User factory + common traits.
5. Create `ship make:factory User` CLI command: scaffolds a new factory file from the model type.
6. Document in `docs/guides/` as a testing guide.

**Done when:** `factories.User.Create(t, db)` inserts a user and returns it with its DB ID set.
Overrides work: `factories.User.Create(t, db, factories.WithAdminRole)` creates an admin.
Sequence ensures unique emails across test runs. `go test ./...` passes.

---

### AA02 — Typed HTTP test helpers

**Status:** `[ ] todo`
**Depends on:** AA01 (factory system), M03 I04 (in-memory test DB)
**Files:** `framework/testutil/` (new package), `framework/testutil/http.go`, `framework/testutil/auth.go`

**Context:** Integration tests that make HTTP requests to the Echo app are verbose: build request,
set headers, handle CSRF, parse response. Agents writing tests get the boilerplate wrong.
Test helpers provide a fluent API that handles the plumbing.

**Test server helper:**
```go
// framework/testutil/http.go
type TestServer struct {
    Server    *httptest.Server
    Container *foundation.Container
    t         testing.TB
}

func NewTestServer(t testing.TB) *TestServer {
    // Create container with in-memory test DB
    // Create Echo app, register all routes
    // Wrap in httptest.Server
}

func (s *TestServer) Get(path string, opts ...RequestOpt) *TestResponse {
    // Build GET request, apply opts, execute, return response
}

func (s *TestServer) PostForm(path string, form url.Values, opts ...RequestOpt) *TestResponse {
    // Build POST request with form body + CSRF token, execute
}

func (s *TestServer) AsUser(userID int64) RequestOpt {
    // Returns an option that sets an auth session cookie for userID
}
```

**Response helper:**
```go
type TestResponse struct {
    *http.Response
    t testing.TB
}

func (r *TestResponse) AssertStatus(code int) *TestResponse {
    r.t.Helper()
    if r.StatusCode != code { r.t.Errorf("expected status %d, got %d", code, r.StatusCode) }
    return r
}

func (r *TestResponse) AssertRedirectsTo(path string) *TestResponse { ... }
func (r *TestResponse) AssertContains(text string) *TestResponse { ... }
func (r *TestResponse) AssertJSON(v any) *TestResponse { ... }  // unmarshal JSON body into v
```

**Usage:**
```go
func TestLogin(t *testing.T) {
    s := testutil.NewTestServer(t)
    user := factories.User.Create(t, s.Container.DB)

    s.PostForm("/login", url.Values{
        "email":    {user.Email},
        "password": {"password"},
    }).AssertRedirectsTo("/auth/homeFeed")

    s.Get("/auth/homeFeed", s.AsUser(user.ID)).AssertStatus(200)
}
```

**What to do:**
1. Create `framework/testutil/http.go` with `TestServer`, `TestResponse`.
2. Create `framework/testutil/auth.go` with session cookie helper (`AsUser`).
3. Create `framework/testutil/csrf.go` with CSRF token injection for POST tests.
4. Wire up the test server to create a real container with in-memory DB.
5. Update existing tests in `tests/e2e/` or `app/` to use the new helpers as examples.
6. Document in testing guide.

**Done when:** A test can make authenticated POST requests to the app in 5 lines. CSRF is
handled automatically. `AssertStatus`, `AssertRedirectsTo`, `AssertContains` work correctly.
`go test ./...` passes.

---

## Group AB — Internationalization

### AB01 — i18n module

**Status:** `[ ] todo`
**Depends on:** M03 C01 (module interface)
**Files:** `modules/i18n/` (new module), `locales/en.yaml` (new), `go.mod` (go-i18n)

**Context:** i18n is painful to retrofit. Adding it early means the framework is ready for it
even if individual apps don't use it initially. The convention is explicit and LLM-friendly:
all user-visible strings that aren't app-specific content go through `t("key")`.

**Library:** `github.com/nicksnyder/go-i18n/v2` — standard, well-maintained, struct-tag based.

**Module structure:**
```
modules/i18n/
├── module.go
├── middleware.go     # detect language from Accept-Language header / user profile / cookie
├── service.go        # T(ctx, key, data...) → translated string
└── views/
    └── language_switcher.templ

locales/
├── en.yaml           # English (source of truth)
└── fr.yaml           # French (example)
```

**Locale file format:**
```yaml
# locales/en.yaml
auth:
  login:
    title: "Sign in to your account"
    submit: "Sign in"
    email_placeholder: "your@email.com"
  errors:
    invalid_credentials: "Invalid email or password"
```

**Usage in controllers:**
```go
// Controller: pass translated strings to viewmodel
page.Title = container.I18n.T(ctx, "auth.login.title")
```

**Templ helper:**
```templ
// In templ files — I18n is passed as a function via context or viewmodel
<h1>{ i18n.T(ctx, "auth.login.title") }</h1>
```

**Language detection middleware (priority order):**
1. `lang` query parameter (`?lang=fr`) — for testing
2. User profile language preference (if authenticated)
3. `lang` cookie (set when user switches language)
4. `Accept-Language` HTTP header
5. Default to `en`

**CLI support:**
```
ship make:locale fr          → creates locales/fr.yaml with all en.yaml keys, values empty
ship i18n:missing            → lists keys present in en.yaml but missing in fr.yaml
ship i18n:unused             → lists keys in locale files not referenced in .go/.templ files
```

**What to do:**
1. Add `github.com/nicksnyder/go-i18n/v2` to `go.mod`.
2. Create `modules/i18n/` with module, middleware, service.
3. Create `locales/en.yaml` with initial keys for auth, navigation, and common UI strings.
4. Create `locales/fr.yaml` as an example translation.
5. Add language detection middleware.
6. Add `ship make:locale`, `ship i18n:missing`, `ship i18n:unused` CLI commands.
7. Wire into container.

**Done when:** `container.I18n.T(ctx, "auth.login.title")` returns the correct translation for
the detected language. `Accept-Language: fr` header returns French strings. `ship i18n:missing`
lists untranslated keys. `ship verify` passes.

---

## Execution Order

**Layer 0 (no dependencies — run in parallel):**
- W01 (OAuth — needs auth module from M03 D01)
- W02 (2FA — needs auth module from M03 D01)
- X01 (AI core)
- Y01 (domain events)
- Y02 (soft deletes)
- Z01 (SSE)
- Z02 (JSON API)
- AA01 (factory system — needs M03 I04)
- AB01 (i18n)

**Layer 1:**
- X02 (OpenAI + OpenRouter — needs X01)
- X03 (AI streaming — needs X01 + Z01)
- X04 (conversation history — needs X01)
- Y03 (feature flags — needs M03 I03 for Otter cache)
- Y04 (audit log — needs Y01 domain events)
- Z03 removed from core roadmap (legacy `api:spec` flow retired)
- AA02 (HTTP test helpers — needs AA01)

**Layer 2:**
- X03 full — needs Z01 complete

**External dependencies (must be done before starting):**
- M03 C01 (module interface) before any module tasks
- M03 D01 (auth module) before W01, W02
- M03 I03 (Otter) before Y03
- M03 I04 (in-memory test DB) before AA01, AA02
