# UI Agent Context: LLM-Friendly UI Convention

Make it easy for any LLM to navigate and edit the Go/HTMX/Templ UI in this repository.

Convention spec: `docs/ui/convention.md`
Playwright MCP setup: `MCP_TOOLS.md`

---

## Dependency Graph

```
T01 (convention spec)        ← DONE: docs/ui/convention.md
├── T02 (style guide)
├── T04 (agent guide)        ← DONE
├── T06 (data-component)
│   └── T08 (data-slot/action) ← BLOCKED on T06
└── T10 (templ comments + routes)

T12 (Playwright MCP)         ← DONE: MCP_TOOLS.md
└── T13 (agent guide visual workflow) ← DONE
```

---

## Tasks

---

### T02 — Write Style Guide

**Status:** `[ ] todo`

**Context:**
GoShip uses Tailwind CSS + DaisyUI with two custom themes: `lightmode` and `darkmode`.
Theme config: `frontend/tailwind.config.js`. CSS source: `app/styles/styles.css`.
LLMs need a reference for what tokens like `accent`, `base-200`, `primary` look like before touching a template.

**What to do:**
Write `docs/ui/style-guide.md` with:

1. **Theme tokens table** — for each DaisyUI token (`primary`, `secondary`, `accent`, `neutral`, `base-100/200/300`, `info`, `success`, `warning`, `error`):
   - Light theme hex value + visual description
   - Dark theme hex value + visual description
   Pull exact values from `frontend/tailwind.config.js`.

2. **Typography** — fonts in use (Playfair Display for branding, system sans for body), where each is applied.

3. **Dark mode** — mechanism: `data-theme="lightmode"` or `data-theme="darkmode"` on HTML root, set by Alpine.js, persisted to localStorage.

4. **Responsive breakpoints** — `lg:` is the primary desktop breakpoint. Mobile is default. `md:` is used occasionally.

5. **Recurring layout patterns** — describe the 3-4 layout patterns that appear across many pages (page container, card, feed-item). Show actual Tailwind class strings — do not abstract them.

6. **HTMX swap patterns** — list common `hx-swap` values and when each is appropriate.

7. **Component libraries** — available: DaisyUI, Flowbite, Alpine.js, HTMX. Policy: prefer DaisyUI first, Flowbite second, custom Tailwind last.

Also add a reference to `docs/ui/style-guide.md` from `docs/00-index.md` under the UI section.

**Done when:** `docs/ui/style-guide.md` exists covering all 7 sections and is linked from `docs/00-index.md`.

---

### T04 — Update Agent Guide with UI Convention ✓

**Status:** `[x] done` → `docs/guides/01-ai-agent-guide.md`

---

### T06 — Apply `data-component` to Templ Component Roots

**Status:** `[ ] todo`

**Context:**
Templ components live in `app/views/web/components/` and `app/views/web/layouts/`.
Each exported templ function needs `data-component="<kebab-case-name>"` on its outermost HTML element.
See `docs/ui/convention.md` section 1 for the naming rule.

**What to do:**
For every exported templ function in `app/views/web/components/` and `app/views/web/layouts/`:
1. Identify the outermost HTML element.
2. Add `data-component="<kebab-case-function-name>"` to it.
3. If the outermost element is a Go fragment (no single root), wrap in `<div data-component="...">` only if it does not break layout. If wrapping would break layout (e.g., a `<tr>`), skip and leave a comment.

Do NOT touch:
- Unexported (lowercase) templ functions.
- Generated `*_templ.go` files.
- Page templates in `app/views/web/pages/` — handled in T10.

After editing, run `make templ-gen` to regenerate `*_templ.go` and confirm no compile errors.

**Done when:** All exported components in `components/` and `layouts/` have `data-component` on their root element and the project compiles cleanly.

---

### T08 — Apply `data-slot` and `data-action` to Components

**Status:** `[ ] todo — BLOCKED on T06`

**Context:**
After `data-component` marks every component root (T06), this task adds `data-slot` and `data-action` to meaningful sub-elements. Goal is not exhaustive — only elements an LLM would plausibly need to target independently.

**What to do:**
For each component in `app/views/web/components/`, read it and add:
- `data-slot="<role>"` to named sub-elements with distinct identity.
- `data-action="<verb-noun>"` to interactive elements where intent is ambiguous.

Priority targets:
- `navbar.templ`: slots `home-link`, `notification-bell`, `notification-count`, `profile-menu`, `theme-toggle`; actions `toggle-theme`, `open-profile-menu`
- `bottom_nav.templ`: slots for each nav item
- `forms.templ`: slots `error-message`, `field-wrapper`
- `messages.templ`: slots `flash-message`, `flash-text`
- `loading.templ`: slot `spinner`
- `payments.templ`: slots `plan-card`, `subscribe-button`

After edits, run `make templ-gen` and verify compile.

**Done when:** All components have meaningful `data-slot` and `data-action` annotations per `docs/ui/convention.md` and the project compiles.

---

### T10 — Add Inline Visual Comments to Templ Functions

**Status:** `[ ] todo`

**Context:**
Exported templ functions in `app/views/web/` currently have no standard description comment.
Convention: two lines immediately above each exported function — `// Renders:` then `// Route(s):`.
See `docs/ui/convention.md` section 4 for the full rule.

**What to do:**
Add both comment lines above every exported templ function in:
- `app/views/web/components/`
- `app/views/web/layouts/`
- `app/views/web/pages/`

For `// Renders:`, describe what the user sees, not how it works.

For `// Route(s):`, read `app/router.go` to trace which handler renders which templ component:
- Page components: use the exact route pattern (e.g., `// Route(s): /`)
- Layout/base components: `// Route(s): all pages` or `// Route(s): all authenticated pages (via base layout)`
- HTMX partials with their own endpoint: use that endpoint's route
- Sub-components embedded inside another templ with no direct route: `// Route(s): embedded in <ParentName>`

Do NOT run `make templ-gen` for comment-only changes — comments in `.templ` files do not affect generated output.

**Done when:** Every exported templ function in `app/views/web/` has both `// Renders:` and `// Route(s):` comments.

---

### T12 — Playwright MCP Setup ✓

**Status:** `[x] done` → `MCP_TOOLS.md`

---

### T13 — Agent Guide: Visual Discovery Workflow ✓

**Status:** `[x] done` → `docs/guides/01-ai-agent-guide.md`

---

## Completion Checklist

```
[x] T01  Convention spec          → docs/ui/convention.md
[ ] T02  Style guide              → docs/ui/style-guide.md (exists but may be incomplete)
[x] T04  Agent guide UI section   → docs/guides/01-ai-agent-guide.md
[ ] T06  data-component roots     → app/views/web/components/ + layouts/
[ ] T08  data-slot / data-action  ← BLOCKED on T06
[ ] T10  Templ comments + routes  → app/views/web/ (components, layouts, pages)
[x] T12  Playwright MCP           → MCP_TOOLS.md
[x] T13  Agent guide visual workflow → docs/guides/01-ai-agent-guide.md
```

To resume: T02, T06, T10 are all unblocked and independent of each other. T08 requires T06 first.
