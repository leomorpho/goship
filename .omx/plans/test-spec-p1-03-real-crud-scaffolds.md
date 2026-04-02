# Test Spec — P1-03 Real CRUD Scaffolds

## Goal

Lock the first real starter CRUD scaffold with generated-app proof before widening the product claim.

## Test Strategy

Prefer generated-app proof and starter smoke proof over generator-unit-only confidence.

## New / Expanded Tests

### 1. `TestStarterCRUDScaffoldIsUseful`
A starter app should be able to generate one canonical CRUD resource and remain buildable.

Expected proof:
- resource generation produces more than a single shell page
- generated route/page/test artifacts exist
- app still builds

### 2. `TestFreshAppCRUDScaffoldFlow`
On a fresh generated app:
- scaffold one canonical resource
- boot the app
- prove index/new/show/edit/delete seams are reachable as designed
- prove validation failure paths exist for create/update forms

Expected proof:
- scaffolded CRUD is an actual app feature baseline

### 3. `TestStarterCRUDDestroyStaysSafe`
After generating the CRUD resource:
- destroy the resource
- app still builds
- owned generated artifacts are removed cleanly

### 4. `TestStarterScaffoldDocsMatchSupportedCRUDSurface`
If starter help/reference changes, prove docs do not overclaim starter scaffold support.

## Verification Commands

Primary:
- `go test ./tools/cli/ship/internal/commands -run 'TestStarter(CRUDScaffoldIsUseful|CRUDDestroyStaysSafe)|TestFreshAppCRUDScaffoldFlow' -count=1`

Secondary as needed:
- `go test ./tools/cli/ship/internal/commands -run 'TestStarterScaffoldDocsMatchSupportedCRUDSurface' -count=1`
- `make test-fresh-app-ci`

## Failure Conditions

Treat the slice as incomplete if:
- CRUD generation is still mostly placeholder output
- create/update paths lack validation integration
- destroy safety regresses
- starter docs/help suggest broader support than the actual starter CRUD path proves
- the new CRUD path is only generator-output proof and not generated-app runtime proof

## Notes

- it is acceptable for this slice to keep starter `make:scaffold` closed if the supported CRUD path becomes genuinely useful first
- do not widen unsupported generator surfaces just to make the CLI look broader
