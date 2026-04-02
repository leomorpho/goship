# P1-02 Follow-Up Architecture — Inline Validation UX

## Why this improvement exists

P1-02 successfully introduced a lightweight starter-safe validation seam:
- reusable field-level `validation_error` shape
- consistent validation behavior across starter auth/account routes
- generated-app invalid-input proof

That solved the **contract** problem.
It did **not** yet solve the **browser UX** problem.

Today, HTML form posts that fail validation return JSON `400` payloads instead of re-rendering the form with inline field feedback.
That is acceptable as a foundation, but it is still below Rails/Laravel-grade productivity and polish.

## Objective

Preserve the new structured validation seam while adding starter-safe HTML form re-rendering with inline field errors and field value preservation.

## Product target

For browser-posted starter forms:
- invalid submission should re-render the same form
- previously entered values should be preserved where safe
- field-level validation messages should appear inline next to the relevant inputs
- success behavior should remain unchanged

For machine-facing surfaces:
- the structured JSON validation contract should remain available when explicitly requested
- do not regress the agent-readable validation seam introduced in P1-02

## Architectural direction

### 1. Keep one validation model, add two render paths
Do **not** replace the current validation shape.
Instead:
- keep `validationError` as the starter-local canonical field-error model
- add response negotiation or explicit HTML-vs-JSON branching
- route handlers should validate once, then choose the response rendering mode

This avoids duplicating validation logic.

### 2. Introduce starter-local form state structs
Add a tiny starter-local form view model for each auth/account form that needs re-rendering.
For example:
- register form state
- login form state
- settings form state
- password reset request form state
- password reset confirm form state
- delete-account form state

Each form state should carry only:
- submitted field values that are safe to echo back
- validation errors keyed by field
- page metadata already needed to render the form

Do **not** overbuild a generic meta-form engine yet.

### 3. Split rendering helpers by intent, not abstraction fantasy
Current rendering helpers are too string-template oriented for inline error UX.
The next step should likely be:
- keep `renderAuthPage` only if it stays useful for success/empty-state rendering
- evolve `renderSimpleFormPage` or replace it with a helper that can render:
  - field values
  - field errors
  - form-level errors if ever needed

A small starter-local rendering helper is preferable to a premature framework-wide form system.

### 4. Preserve safe-field echo rules
When re-rendering after validation failure:
- preserve text/email/date fields
- do **not** repopulate password fields
- preserve hidden safe values like `next` where needed

This should be explicit in the rendering layer, not accidental.

### 5. Keep auth failures distinct from validation failures
Maintain this distinction:
- validation failure -> `400` with inline HTML re-render for browser posts, JSON for machine-facing requests
- authentication/authorization failure -> existing `401`/redirect behavior

Do not collapse invalid credentials into the validation channel.
That current asymmetry is actually correct.

## Recommended implementation slices

### Slice A — HTML re-render contract for register/login/settings
1. add starter-local form state structs
2. add inline field-error rendering support
3. re-render register/login/settings on validation failure
4. preserve safe field values

### Slice B — HTML re-render contract for reset/delete flows
1. password reset request re-render
2. password reset confirm re-render
3. delete-account re-render
4. keep token/email handling safe and explicit

### Slice C — negotiation / dual-surface preservation
1. keep structured JSON validation output for machine-facing callers
2. choose HTML re-render for typical browser form posts
3. avoid implicit heuristics that are hard for agents to reason about

## Suggested response-shape rule

Prefer a simple rule like:
- if `Accept` includes `application/json`, return JSON validation payload
- otherwise return HTML re-render for starter form submissions

This is simple, conventional, and predictable.

## Test plan for the improvement

### Generated-app browser/HTTP proof to add
- register invalid submission re-renders with inline display-name/email/password errors
- login invalid submission re-renders with inline email/password errors
- settings invalid submission re-renders with inline display-name error
- password reset invalid submission re-renders with inline errors
- delete-account invalid submission re-renders with inline email-confirmation error
- JSON-request path still returns structured `validation_error` payloads

### Proof expectations
- same route re-render, no redirect on validation failure
- body includes field-specific validation markers
- non-password values persist where safe
- password fields do not echo back

## Boundaries / non-goals

Do not turn this into:
- a full framework-wide forms package
- a universal request DTO system
- a complete templ rewrite of all starter forms
- a full client-side validation system

This is specifically a **starter browser UX follow-up** to P1-02.

## Recommendation

This should happen **before or during P1-03**, not long after.

Why:
- CRUD scaffolds without inline validation UX will still feel below Rails/Laravel quality
- the current validation seam is good enough to extend now
- waiting too long risks baking JSON-only validation assumptions into the generator story
