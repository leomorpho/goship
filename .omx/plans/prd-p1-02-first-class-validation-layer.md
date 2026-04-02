# PRD — P1-02 First-Class Validation Layer

## Objective

Turn starter validation from scattered `http.Error(...)` checks into a coherent starter-safe validation layer that raises app-building leverage and sets up stronger generators.

## Problem

Today the generated starter mostly validates by:
- `ParseForm`
- checking one or two required fields inline
- returning raw string errors via `http.Error`

That is enough to reject obviously bad input.
It is not enough to feel framework-native.

Without a reusable validation layer:
- auth/account forms stay repetitive
- future CRUD scaffolds will repeat the same glue
- errors are not structured consistently
- HTML and JSON behavior cannot converge cleanly

## Product Goal

A fresh starter app should provide a small but real validation contract that:
- makes form validation defaults reusable
- keeps errors consistent
- reduces handwritten validation glue
- becomes the foundation for P1-03 and P3-01

## Scope

### In scope
- starter-safe validation helpers/primitives on the generated-app path
- reusable field-error shape
- consistent validation behavior for current starter auth/account forms
- generated-app proof for validation failures on starter auth/account flows
- starter contract/docs alignment if visible behavior changes

### Out of scope
- full framework-wide validation unification
- deep localization/i18n of all validation errors
- complete request DTO framework for every surface
- admin/resource scaffold generation
- advanced CSRF/form-state UX improvements beyond validation behavior

## Current Truth Baseline

Current starter auth/account routes validate inline in:
- `tools/cli/ship/internal/templates/starter/testdata/scaffold/cmd/web/main.go`

Examples today:
- register requires email/password
- password reset request requires email
- password reset confirm requires email/token/password
- delete-account checks email confirmation match
- failures usually return raw strings with `http.Error`

## Target Validation Contract

### Behavioral expectations
- starter forms use one consistent validation path for field-level failures
- validation failures are structured by field, not just generic strings
- starter HTML responses expose validation failure details predictably
- where JSON is already a machine-facing surface, validation shape should be reusable there too if practical

### Minimum starter fields to cover in this slice
- register:
  - display name required
  - email required
  - password required
- login:
  - email required
  - password required
- settings:
  - display name required
- password reset request:
  - email required
- password reset confirm:
  - email required
  - token required
  - password required
- delete-account:
  - email confirmation required and must match current user

## Delivery Slices

### Slice A — starter validation primitives
1. introduce small reusable validation error shape
2. add helpers for required-field validation and common starter rendering
3. avoid over-abstracting beyond current starter needs

### Slice B — auth/account route adoption
1. convert existing starter auth/account routes to use the validation helpers
2. keep behavior readable and starter-local
3. preserve current success paths

### Slice C — proof + contract alignment
1. add failing generated-app tests for validation failures
2. prove field-specific validation behavior on starter flows
3. update docs/contract tests only if visible starter surface changes

## Constraints

- keep the layer lightweight and generated-app-safe
- do not introduce a broad new framework abstraction unless it clearly reduces starter duplication
- prefer reuse only when it does not add confusing coupling
- keep future generator/resource integration in mind without overbuilding for it now

## Acceptance Criteria

- starter auth/account forms share a coherent validation path
- validation failures are field-specific and consistent
- generated-app proof covers invalid-input paths, not just happy paths
- the starter remains no-infra friendly and easy to understand
- this slice clearly improves the foundation for P1-03/P3-01

## Sequencing Recommendation

Implement in this order:
1. failing validation tests
2. starter validation primitives
3. auth/account route conversion
4. docs/contract alignment if needed
