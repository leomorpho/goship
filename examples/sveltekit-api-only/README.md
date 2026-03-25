# SvelteKit API-Only Reference App

This reference app is the canonical implementation of GoShip's blessed split-frontend contract:
`api-only-same-origin-sveltekit-v1`.

## Scope

- SvelteKit is the frontend (`SvelteKit-first`).
- GoShip is scaffolded in API-only mode.
- Browser auth uses `same-origin auth/session`.
- Browser writes preserve cookie/CSRF protections.

## Backend Setup

Create the API backend with the canonical command:

```bash
ship new demo --module example.com/demo --api-only
```

```bash
ship new demo --module example.com/demo --api
```

The backend keeps standard auth/session endpoints such as `/auth/login` and `/auth/register`.

## Frontend Contract

Use [`src/lib/server/goship-contract.ts`](src/lib/server/goship-contract.ts) as the stable TypeScript-facing contract surface for:

- response envelope typing
- API error typing
- session-aware fetch with CSRF header forwarding (`X-CSRF-Token`)

## Local Dev Topology

- Run GoShip API and SvelteKit separately in development.
- Keep browser-facing requests on one origin via the SvelteKit server.
- Route browser writes through server handlers/actions that forward cookies and `X-CSRF-Token`.

## Deployment Topology

Deploy behind one public origin:

- `/api/*` and `/auth/*` go to GoShip.
- UI/page requests go to SvelteKit.

Cross-origin browser session writes are out of scope for this blessed contract.
