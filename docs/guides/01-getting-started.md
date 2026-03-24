# Getting Started in Under 30 Minutes

This guide walks a Go developer from zero to a production-credible GoShip app.

## Prerequisites

- Go 1.25+
- Git
- Node.js 18+
- `overmind` (for full multi-process dev loops)

## 1. Install the `ship` CLI

From the repository root:

```bash
go install ./tools/cli/ship/cmd/ship
```

Verify install:

```bash
ship --help
```

## 2. Generate a New App

Create a fresh app scaffold:

```bash
ship new myapp --module example.com/myapp --no-i18n
cd myapp
```

Initialize runtime dependencies and migrate your local DB:

```bash
ship db:migrate
```

## 3. Start Development Mode

Run the canonical local loop:

```bash
ship dev
```

Optional explicit modes:

```bash
ship dev --web
ship dev --all
```

## 4. Add an Auth Battery Surface

The starter includes core auth routes; add the `2fa` battery to harden auth flows:

```bash
ship module:add 2fa
```

Re-run verification after module wiring:

```bash
ship verify --profile fast
```

## 5. Run Tests and Verification

Run the default test workflow:

```bash
ship test
```

Run integration tests when needed:

```bash
ship test --integration
```

## 6. Deployment Readiness and Deploy Hand-off

Before deploy, validate repo/runtime health:

```bash
ship doctor
ship verify --profile strict
```

If you are using the default Kamal workflow, continue with the deployment steps documented in:

- `docs/guides/04-deployment-kamal.md`

## What You Should Have Now

- A booting GoShip app created by `ship new`
- Local DB migrated with `ship db:migrate`
- Dev loop running via `ship dev`
- Auth hardening battery added with `ship module:add 2fa`
- Test and verification commands passing

