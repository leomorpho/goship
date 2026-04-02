# Test Spec — Post-V1 Unified Generation / Default App / Batteries

## Verification strategy
The program is complete only when each program has fresh evidence at three layers:
1. generator contract tests
2. generated-app proof tests
3. integration / packaging / CI truth tests

## Program 1 proof expectations
- failing-proof-first coverage for resource/controller/model generation contracts
- destroy/idempotency/drift tests for generator-owned artifacts
- starter-generated CRUD/resources backed by a truthful data contract
- no regression in existing fresh-app proof lanes

## Program 2 proof expectations
- admin/backoffice generated-app proof
- policy generation and enforcement proof
- mailer preview/runtime proof
- docs/help/runtime truth alignment tests

## Program 3 proof expectations
- downstream battery install proof outside repo-local assumptions
- add/remove/build/runtime-report proof for storage first
- machine-readable battery metadata tests
- compatibility matrix CI coverage

## Mandatory verification commands (evolve as implementation lands)
- targeted `go test` for affected generator/command/template areas
- `go build ./tools/cli/ship/cmd/ship`
- `make test-fresh-app-ci`
- any new matrix/package/install proof targets introduced by the tranche

## Reopen gate for starter `make:scaffold`
Do not reopen until tests prove:
- starter-safe controller/resource generation is truthful
- model/data generation is coherent enough for scaffold composition
- destroy/idempotency are safe on the composed path
