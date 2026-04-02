# Boundary Charter

## Purpose

This document defines the intended ownership boundaries between:
- `framework/`
- `modules/`
- the starter/default generated-app preset

The goal is to keep GoShip honest as it grows into a higher-productivity framework.

## Core rule

- `framework/` owns reusable seams and infrastructure contracts.
- `modules/` owns installable product capabilities (batteries).
- the starter preset owns the default app experience and day-one leverage.

## `framework/` ownership

Put something in `framework/` when it is:
- reusable across many apps
- a seam or contract rather than a product feature
- generic enough to survive many batteries plugging into it

Examples:
- runtime/container/router/schedule seams
- request/response primitives
- validation primitives
- auth/session primitives
- policy/authorization primitives
- storage interfaces
- mailer interfaces
- jobs interfaces
- cache/pubsub/db abstractions
- runtime report / route export / contract machinery
- framework test helpers for those seams

### Do not put in `framework/`
Do not put product behavior in core just because module packaging is not finished yet.

Examples that should not be core-owned product features:
- paid subscription product flows
- notification center/product inbox behavior
- email subscription product behavior
- profile/account product features
- admin product screens
- billing product workflows

## `modules/` ownership

Put something in `modules/` when it is:
- optional
- product-facing
- installable/removable
- expected to plug into core seams

Examples:
- `jobs`
- `storage`
- `paidsubscriptions`
- `emailsubscriptions`
- `notifications`

The long-term notifications direction should be split batteries rather than one monolith:
- `notifications-inbox`
- `notifications-push`
- `notifications-email`
- `notifications-sms`
- `notifications-schedule`

### Good battery traits
A first-party installable battery should have:
- a standalone module boundary
- one public entrypoint
- an explicit install contract
- clean ownership over routes/config/assets/jobs/templates/migrations/tests as needed
- no leakage of app-domain behavior into framework core

## Starter preset ownership

The starter/default generated-app preset should own:
- the default app experience
- starter auth/account UX
- starter landing/home/profile/admin-ish presets if included by default
- starter form/page wiring that increases day-one product leverage

Rule:
- if it exists to make `ship new` useful immediately, it belongs to the starter preset layer

Important:
- the starter preset may use `framework/` seams
- the starter preset may later install `modules/`
- but the starter preset should not become the dumping ground for framework ownership either

## Decision rules

### Put it in `framework/` if:
- it is a generic seam
- it reduces duplication across many apps or batteries
- it has no app-domain opinion baked in

### Put it in `modules/` if:
- it is optional
- it expresses product behavior
- apps should be able to add/remove it

### Put it in starter preset if:
- it is part of the default generated-app promise
- it exists to increase immediate app leverage

## Anti-cheating rule

Do not move a real battery into `framework/` just because installability is temporarily weak.

That hides the real architecture problem instead of solving it.

If something is conceptually a battery:
- keep it conceptually battery-owned
- improve packaging/installability
- or narrow the product claim honestly

## Current blunt reading

Architecturally:
- `paidsubscriptions` should be a module
- `emailsubscriptions` should be a module
- `notifications` should be a module (and later split into narrower modules)
- `storage` should be a module
- `jobs` should be a module

Core/framework should own only the seams those batteries plug into.

## Relationship to existing policy

This charter is consistent with:
- `docs/architecture/11-module-surface-reset.md`
- the current installable-battery direction
- the broader post-v1 productivity goal of making batteries real downstream capabilities
