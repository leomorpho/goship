# Test Spec — P1-02 First-Class Validation Layer

## Goal

Lock starter validation behavior with generated-app proof before implementation is accepted.

## Strategy

Prefer generated-app invalid-input proof over unit-only confidence.

The first target is the current starter auth/account surface introduced in P1-01.

## New / Expanded Tests

### 1. `TestFreshAppAuthValidationFailures`
Covers invalid starter auth/account submissions:
- register with missing display name
- register with missing email
- register with missing password
- login with missing email/password
- settings with missing display name

Expected proof:
- responses are validation failures, not generic success/redirects
- body exposes stable field-level validation details

### 2. `TestFreshAppPasswordResetValidationFailures`
Covers:
- password reset request with missing email
- password reset confirm with missing token/password/email

Expected proof:
- stable field-level validation details are returned

### 3. `TestFreshAppDeleteAccountValidationFailures`
Covers:
- delete-account with missing email confirmation
- delete-account with mismatched email confirmation

Expected proof:
- validation failures are field-specific and do not mutate account state

### 4. Optional contract test
If starter-visible docs/help mention validation behavior explicitly, add a small contract test so docs do not drift.

## Verification Commands

Primary:
- `go test ./tools/cli/ship/internal/commands -run 'TestFreshApp(AuthValidationFailures|PasswordResetValidationFailures|DeleteAccountValidationFailures|AuthFlow|AuthAccountLifecycle|PasswordResetFlow|DeleteAccountFlow)' -count=1`

Secondary as needed:
- `make test-fresh-app-ci`

## Failure Conditions

Treat the slice as incomplete if:
- invalid submissions still rely on ad hoc generic error strings with no reusable field mapping
- validation failures accidentally authenticate, mutate, or delete state
- only success paths are covered by generated-app proof
- starter validation shape becomes so abstract that it is harder to understand than the original inline code

## Notes

- keep this slice focused on starter auth/account validation and the foundation for future scaffolds
- do not drag in full framework validation unification unless it is clearly the smallest truthful path
