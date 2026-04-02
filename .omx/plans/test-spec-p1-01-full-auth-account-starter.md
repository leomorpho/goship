# Test Spec — P1-01 Full Auth / Account Starter

## Goal

Lock the richer starter auth/account contract with generated-app proof before implementation claims are accepted.

## Test Strategy

Prefer generated-app proof over framework-internal unit-only confidence.

Use layered proof:
1. focused starter command/runtime tests in `tools/cli/ship/internal/commands/fresh_app_test.go`
2. targeted contract tests if docs/help/route declarations change
3. existing fast fresh-app lane should absorb the new auth/account proof over time

## New / Expanded Tests

### 1. `TestFreshAppAuthAccountLifecycle`
Covers the primary happy path on a fresh starter app:
- register new account
- confirm authenticated session via `GET /auth/session`
- update account settings
- logout
- login with updated account data/unchanged password as appropriate

Expected proof:
- authenticated session exists after register/login
- settings change persists for the current starter runtime contract
- protected routes still redirect when session is absent

### 2. `TestFreshAppPasswordResetFlow`
Covers password reset baseline:
- create starter user
- request password reset
- obtain deterministic starter-safe reset token/result
- complete reset with new password
- old password fails
- new password succeeds

Expected proof:
- password reset is real and does not require external mail infra
- starter-safe reset confirmation works deterministically in tests

### 3. `TestFreshAppDeleteAccountFlow`
Covers deletion baseline:
- register/login
- confirm delete-account page exists
- submit delete-account
- session clears
- protected route redirects
- old credentials no longer log in

Expected proof:
- deletion removes usable account state from the starter runtime contract

### 4. `TestFreshAppAuthRouteInventoryIncludesAccountLifecycle`
If route inventory stays machine-readable for starter routes, extend route expectations so starter route inventory truthfully includes the new auth/account routes.

### 5. `TestStarterAuthDocsMatchExpandedContract`
Only needed if docs/help are updated.
Ensures starter-facing docs/help do not drift from the implemented auth/account route surface.

## Verification Commands

Primary:
- `go test ./tools/cli/ship/internal/commands -run 'TestFreshApp(AuthFlow|AuthAccountLifecycle|PasswordResetFlow|DeleteAccountFlow|AuthRouteInventoryIncludesAccountLifecycle)' -count=1`

Secondary as needed:
- `go test ./tools/cli/ship/internal/commands -run 'TestStarterAuthDocsMatchExpandedContract' -count=1`
- `make test-fresh-app-ci`

## Failure Conditions

Treat the work as incomplete if any of these are true:
- richer routes exist only in docs but not generated-app runtime
- password reset requires external services to prove
- delete-account leaves the session usable
- session restore requires scraping HTML instead of a stable starter-safe surface
- updated docs/help describe account capabilities that the fresh app does not actually implement

## Notes

- keep this contract explicitly about the generated starter auth surface
- do not accidentally pull the framework-repo `/user/*` auth surface into the starter proof lane
- if richer validation ships as part of this ticket, keep those expectations narrow and starter-specific rather than prematurely asserting the full P1-02 validation contract
