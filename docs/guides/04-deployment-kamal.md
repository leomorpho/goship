# Deployment: Kamal

This guide documents the current deployment path for GoShip using Kamal.

## Scope

Current deploy method covered here:

- Kamal

Current supported topology in this guide:

- web host + separate worker host
- external Postgres
- external Redis
- distributed-style process topology (not the single-binary local path)

## Required Files

- `infra/deploy/kamal/deploy.yml`
- `.kamal/secrets`

Keep non-secret config in `infra/deploy/kamal/deploy.yml`.
Keep secrets in `.kamal/secrets` (or your equivalent secret backend workflow).

## Preflight

Before deploying:

1. Confirm image/registry settings in `infra/deploy/kamal/deploy.yml`.
2. Confirm server hosts and SSH configuration.
3. Confirm runtime environment variables (DB, cache, app secrets).
4. Run local checks:
   - `go run ./tools/cli/ship/cmd/ship test`
   - `go run ./tools/cli/ship/cmd/ship test --integration`
   - `go run ./tools/cli/ship/cmd/ship verify --profile fast`

## First Setup

```bash
kamal setup -c infra/deploy/kamal/deploy.yml
```

This performs initial server bootstrapping and first deployment.

## Standard Deploy

```bash
kamal deploy -c infra/deploy/kamal/deploy.yml
```

Use this for normal release pushes.

## Traefik Recovery

If Traefik state is unhealthy after host restart or networking changes:

```bash
kamal traefik reboot -c infra/deploy/kamal/deploy.yml
```

## Notes

- This Kamal path assumes a distributed-style deployment with worker separation and reachable Redis/Postgres services.
- It does **not** document the single-binary local path as a deployment story.
- If using Redis-backed async/realtime paths, ensure Redis is reachable from deployed processes.
- Keep deployment docs in sync with `docs/roadmap/01-framework-plan.md` as runtime modes evolve.
