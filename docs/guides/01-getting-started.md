# Getting Started in Under 30 Minutes

This guide walks a Go developer from zero to a production-credible GoShip app.

## Prerequisites

- Go 1.25+
- Git
- Node.js 18+
- `overmind` (for full multi-process dev loops)

## 1. Install the `ship` CLI

From anywhere:

```bash
go install github.com/leomorpho/goship/tools/cli/ship/v2/cmd/ship@v2.0.5
```

Verify install:

```bash
ship --help
```

## 2. Generate a New App

Create a fresh starter app scaffold:

```bash
ship new myapp --module example.com/myapp --no-i18n
cd myapp
```

The default `ship new` output is the minimal starter scaffold, not the full module-capable framework workspace.
Installable batteries still target the full framework workspace today, so do not rely on `ship module:add` inside a fresh starter app yet.

Run the canonical first-boot sequence:

```bash
ship db:migrate
ship dev
```

## 3. Start Development Mode

`ship dev` is the canonical local loop for the generated starter:

```bash
ship dev
```

Optional explicit modes:

```bash
ship dev --web
ship dev --all
```

## 4. Starter Auth Surface

The starter includes the landing/auth/home/profile route surface.

```bash
# build on the starter routes first, then move to the full workspace shape if you need installable batteries
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
- Starter landing/auth/home/profile routes available for further app-specific work
- Test and verification commands passing
