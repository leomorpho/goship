# Pagoda Intake Log

Canonical adopt/adapt/skip decision log for recurring Pagoda upstream intake.

## Cadence

- Review Pagoda upstream changes weekly or per-tag, whichever happens first.
- Record each reviewed upstream item here even when the decision is `skip`.
- Link follow-up GoShip tickets when an item is adopted or adapted.

## Decision Meanings

- `adopt`: bring the upstream idea into GoShip substantially as-is.
- `adapt`: port the idea with GoShip-specific changes to fit current architecture.
- `skip`: explicitly decline the idea for now.

## Intake Table

| Upstream Ref | Area | Decision | Follow-Up | Notes |
|---|---|---|---|---|
| Pagoda SQLite-first local runtime shift | local runtime | adapt | `TKT-198`, `TKT-199` | Aligns with GoShip single-node-first direction, but through GoShip adapter boundaries. |
| Pagoda Backlite jobs direction | jobs backend | adapt | `TKT-200`, `TKT-250` | Reused as a GoShip jobs backend candidate instead of a one-to-one runtime copy. |
| Pagoda in-memory cache default | cache | adapt | `TKT-200` | Matches GoShip single-binary ergonomics while preserving adapter portability. |
| Pagoda gomponents UI stack | UI rendering | skip | none | Conflicts with GoShip Templ + HTMX direction. |
