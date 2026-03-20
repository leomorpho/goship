# Framework Guide

## Role

The `framework/` layer provides the reusable platform for GoShip:

- routing primitives
- dependency injection contracts
- config loading
- DB and repository helpers
- session and middleware helpers
- rendering pipeline support

It does not own auth flows, app-specific business logic, or module-specific code.

## Read Before Changing

Read `framework/core/interfaces.go` before making any framework change. This file defines the
contracts that the app layer and modules rely on.

## Dependency Rules

- All optional services must go through adapter interfaces.
- Do not add direct dependencies from framework code into app or module packages.
- Allowed dependencies are the standard library, Echo, Bob, and cleanenv.
- Do not add new external packages without explicit approval.

## Breaking Change Rule

Any change to `framework/core/interfaces.go` requires updating:

- all adapter implementations
- all affected framework tests

Do not leave interface changes partially wired.

## Testing Rule

Every exported function in `framework/` must have a test. Run `ship verify` after every change.
