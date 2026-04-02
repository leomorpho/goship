# P1-03 Product-Truth Decision — Starter `make:scaffold`

## Decision

Keep starter `make:scaffold` **closed for now**.

## Why

The starter path now has a first genuinely useful CRUD baseline through the supported resource flow, but starter `make:scaffold` still composes generator surfaces that are not yet truthfully starter-ready.

Current reality:
- `make:resource` is starter-safe and now materially more useful
- `make:model` is starter-safe but still primarily a query/migration stub surface
- `make:controller` is explicitly starter-rejected
- `make:scaffold` still depends on `make:controller`

That means reopening starter `make:scaffold` right now would create a broader product claim than the generated-app path can honestly support.

## Product-truth rule applied

Do not reopen starter orchestration surfaces until the underlying component generators are genuinely starter-ready.

In this case:
- a better supported `make:resource` path is real
- a truthful starter `make:controller` path is not yet real
- therefore starter `make:scaffold` should remain closed

## What should happen next instead

Before reconsidering starter `make:scaffold`, land:
1. stronger controller/resource generation (`P3-01`)
2. stronger model/data generation (`P3-02`)
3. further CRUD scaffold depth on the starter path

Only then revisit whether a narrow starter-safe scaffold orchestration is truthful.

## Practical guidance

For now, the starter product story should be:
- use the improved `make:resource`
- use `make:model`
- keep `make:scaffold` documented as framework-workspace-only

Do not widen the starter claim until the composed path is proven by generated-app runtime evidence, not just by generator composition logic.
