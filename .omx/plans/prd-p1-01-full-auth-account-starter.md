# PRD — P1-01 Full Auth / Account Starter

## Objective

Upgrade the generated starter app from a minimal auth demo to a real account baseline that materially increases day-one app leverage.

The starter should remain intentionally lighter than the full framework-repo app, but it should no longer stop at register/login/logout. It should cover the minimum account lifecycle a serious app builder expects from a modern high-productivity framework.

## Problem

The current starter auth surface is real, but too thin:
- register
- login
- logout
- protected-route redirect

That is enough to prove the auth seam exists.
It is not enough to make `ship new` feel like a real app platform.

In Rails/Laravel terms, the starter is still missing too much of the “boring but always needed” account lifecycle:
- password reset request flow
- password reset completion flow
- session restore / who-am-I surface
- account settings surface
- account deletion flow
- stronger validation and error behavior around auth forms

## Product Goal

A fresh starter app should provide a coherent account baseline that lets an app builder immediately extend product features instead of first building account infrastructure.

## Scope

### In scope
- richer starter auth/account runtime on the generated-app path
- starter-safe HTML auth/account flows
- starter-safe session restore contract
- starter-safe account settings page/flow
- starter-safe delete-account confirmation + completion flow
- password reset request + reset completion baseline
- generated-app proof coverage for the expanded lifecycle
- docs/help alignment for the starter auth contract

### Out of scope
- OAuth/social login
- 2FA
- managed-mode account branching
- full framework-repo `/user/*` contract unification
- enterprise-grade email delivery infra
- full production-grade auth hardening beyond the starter contract

## Current Truth Baseline

Today the generated starter proves:
- `GET /auth/register`
- `POST /auth/register`
- `GET /auth/login`
- `POST /auth/login`
- `GET /auth/logout`
- protected-route redirect to `/auth/login?next=...`
- authenticated access to `/auth/profile`

Current implementation lives primarily in:
- `tools/cli/ship/internal/templates/starter/testdata/scaffold/cmd/web/main.go`
- starter routes in `tools/cli/ship/internal/templates/starter/testdata/scaffold/app/router.go`
- generated-app proof in `tools/cli/ship/internal/commands/fresh_app_test.go`

## Target Starter Contract

### Public auth/account entrypoints
- `GET /auth/register`
- `POST /auth/register`
- `GET /auth/login`
- `POST /auth/login`
- `GET /auth/password/reset`
- `POST /auth/password/reset`
- `GET /auth/password/reset/confirm`
- `POST /auth/password/reset/confirm`

### Authenticated entrypoints
- `GET /auth/logout`
- `GET /auth/profile`
- `GET /auth/settings`
- `POST /auth/settings`
- `GET /auth/delete-account`
- `POST /auth/delete-account`
- `GET /auth/session`

### Required behavioral expectations
- successful register starts an authenticated session
- successful login respects `next`
- session restore lets the starter confirm authenticated identity without scraping HTML-only state
- password reset request is starter-real and testable without external delivery infra
- password reset completion updates credentials and invalidates stale session assumptions
- account settings update succeeds through a real starter-safe path
- delete-account clears auth state and makes the old credentials unusable

## Delivery Slices

### Slice A — session/account baseline
1. add `GET /auth/session`
2. add `GET /auth/settings`
3. add `POST /auth/settings`
4. extend starter templates/pages as needed
5. prove session restore + settings update

### Slice B — password reset baseline
1. add password-reset request surface
2. add deterministic starter-safe reset token flow
3. add reset confirmation surface
4. prove old password fails and new password succeeds

### Slice C — delete-account baseline
1. add delete-account confirmation page
2. add delete-account completion handler
3. clear session and invalidate credentials
4. prove protected-route behavior after deletion

### Slice D — docs/truth alignment
1. update starter docs if they mention auth/account surface
2. update any contract tests tied to starter auth wording
3. keep the generated-app auth story explicitly separate from the framework-repo `/user/*` surface

## Constraints

- keep the starter path lightweight and generated-app-safe
- prefer simple deterministic starter state over introducing heavy auth infrastructure
- do not blur framework-repo auth surface and starter auth surface
- do not introduce optional batteries or external services just to make starter auth look richer
- preserve the same-origin cookie/session browser boundary

## Acceptance Criteria

- a fresh starter app supports register/login/logout/password-reset/settings/delete-account/session-restore flows
- generated-app proof covers the expanded lifecycle end-to-end
- the starter remains buildable with the existing no-infra default path
- docs/help do not overclaim beyond the starter-auth contract actually implemented
- the auth/account starter meaningfully increases day-one app value compared with the current minimal baseline

## Sequencing Recommendation

Implement in this order:
1. session/settings
2. password reset
3. delete-account
4. docs alignment

This keeps the work incremental and keeps the proof lane readable.
