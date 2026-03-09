# UI Agent Convention

Specification for `data-*` attribute annotations and inline visual comments
in Go/Templ/HTMX UI code in this repository.

---

## 1. `data-component` Rule

Add `data-component` to the **root HTML element** of every exported templ component function.

**Value:** the templ function name converted to kebab-case.

```templ
// Navbar → data-component="navbar"
templ Navbar(page *ui.Page) {
  <nav data-component="navbar" class="...">
    ...
  </nav>
}

// AnswerEmojiReactions → data-component="answer-emoji-reactions"
templ AnswerEmojiReactions(...) {
  <div data-component="answer-emoji-reactions" class="...">
    ...
  </div>
}
```

**Rules:**
- One `data-component` per component — on the root element only, never on children.
- Never used in CSS selectors or for styling.
- If the component has no single root element (Go fragment), wrap in a `<div data-component="...">` only if it does not break layout (e.g., wrapping a `<tr>` would break a table — skip in that case and leave a comment noting it).
- Unexported (lowercase) templ helpers do not need this attribute.
- Page templates (in `pages/`) may also carry this attribute using the same rule.

**Purpose:** allows any LLM or test to locate a component's rendered root in one grep:
```
grep -r 'data-component="navbar"'
```

---

## 2. `data-slot` Rule

Add `data-slot` to **named sub-elements** within a component that have distinct identity — elements an LLM or test would need to target independently.

**Value:** a kebab-case description of the element's role within the component.

```templ
templ Navbar(page *ui.Page) {
  <nav data-component="navbar" class="...">
    <a data-slot="home-link" href="...">...</a>
    <button data-slot="notification-bell" ...>
      <span data-slot="notification-count">{ count }</span>
    </button>
    <div data-slot="profile-menu" ...>...</div>
    <button data-slot="theme-toggle" ...>...</button>
  </nav>
}
```

**Rules:**
- Only add where the element has a meaningful, distinct role within the component. Not every element needs one.
- Ask: "Would an LLM be asked to change this element specifically?" If yes, add `data-slot`. If no, skip.
- Values should be nouns or noun-phrases describing role, not appearance. Use `notification-count` not `red-badge`.
- Never used in CSS selectors or for styling.
- A `data-slot` element may also carry `data-action` (see below) — they are not mutually exclusive.

---

## 3. `data-action` Rule

Add `data-action` to **interactive elements** (buttons, links, form triggers) where the intent is not immediately obvious from surrounding HTMX or Alpine attributes alone.

**Value:** a verb-noun in kebab-case describing what the interaction does.

```templ
<button
  data-action="toggle-theme"
  @click="toggleTheme()"
  class="...">
  ...
</button>
```

**Rules:**
- Skip if the element already has a clear `hx-post`, `hx-get`, or `@click` that makes the intent self-evident. Only add when there is genuine ambiguity.
- Use active verb-noun form: `open-emoji-picker`, `submit-answer`, `toggle-theme`, `delete-answer`.
- Never used in CSS selectors or for styling.

---

## 4. Inline Visual Comment Rule

Every **exported** templ component function gets two comment lines immediately above it: `// Renders:` and `// Route(s):`.

**Format:**
```go
// Renders: <plain English description of what the user sees>
// Route(s): <comma-separated route patterns where this component appears>
templ ComponentName(...) {
```

**Examples:**
```go
// Renders: sticky top nav with logo, home icon, notification bell with unread count badge, profile dropdown, and theme toggle button
// Route(s): all authenticated pages (via base layout)
templ Navbar(page *ui.Page) {

// Renders: mobile bottom navigation bar with icons for home, notifications, and profile
// Route(s): all authenticated pages (via base layout)
templ BottomNav(page *ui.Page) {

// Renders: inline validation error list below a form field, styled as red warning text
// Route(s): embedded in form components
templ FormFieldErrors(errs []string) {

// Renders: full landing page with hero section, feature highlights, pricing table, and FAQ accordion
// Route(s): /
templ LandingPage(page *ui.Page) {
```

**`// Renders:` rules:**
- Describes what the **user sees** — not implementation, not props, not HTMX wiring.
- One line only. If it needs more, the component is likely doing too much.
- For unexported (lowercase) templ helpers, the comment is optional but encouraged.
- Do not duplicate the function name — describe the visual output.

**`// Route(s):` rules:**
- Use the same route pattern syntax as `app/router.go` (e.g., `/settings/plan`).
- For layout or base components rendered on every page: `// Route(s): all pages` or `// Route(s): all authenticated pages (via base layout)`.
- For HTMX partial/fragment components rendered from a swap endpoint: use the endpoint route.
- For deeply nested sub-components embedded within a parent templ function with no direct route: `// Route(s): embedded in <ParentComponentName>`.
- Multiple routes: comma-separated on one line.
- Keep it accurate. When a route changes or a component moves to a new page, update this comment alongside the code change.

---

## 5. What These Attributes Are NOT For

- **Not for styling.** CSS selectors must never target `data-component`, `data-slot`, or `data-action`.
- **Not a replacement for `data-testid`.** Playwright tests may continue to use `data-testid`. These attributes are complementary — `data-component`/`data-slot` are permanent semantic annotations; `data-testid` is test-scoped.
- **Not exhaustive.** Do not annotate every element. The target is: component root + meaningful slots + ambiguous actions. Over-annotation creates noise.

---

## 6. Quick Reference

| Attribute | Where | Value format | Required? |
|---|---|---|---|
| `data-component` | Root element of exported templ function | kebab-case function name | Yes, all exported components |
| `data-slot` | Named sub-elements with distinct role | kebab-case role noun | Only where independently targetable |
| `data-action` | Interactive elements with ambiguous intent | kebab-case verb-noun | Only where HTMX/Alpine attributes are insufficient |
| `// Renders:` | Line above exported templ function | Plain English visual description | Yes, all exported functions |
| `// Route(s):` | Line below `// Renders:` | Route pattern(s) from router.go | Yes, all exported functions |
