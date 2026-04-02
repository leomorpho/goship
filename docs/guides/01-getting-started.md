# Getting Started in Under 30 Minutes

This guide walks a Go developer from zero to a production-credible GoShip app.

## Prerequisites

- Go 1.25+
- Git
- Node.js 18+
- `overmind` (for full multi-process dev loops)

## 1. Install the `ship` CLI

From a fresh clone of the repo:

```bash
git clone https://github.com/leomorpho/goship.git
cd goship
go build -o ./bin/ship ./tools/cli/ship/cmd/ship
./bin/ship --help
```

If you want `ship` on your `PATH` locally:

```bash
cp ./bin/ship ~/.local/bin/ship
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

Stay on the default `ship dev` path for the starter happy path.
Advanced multi-process modes are covered in `docs/guides/02-development-workflows.md`
once you intentionally move beyond the minimal starter loop.

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

Before deploy, validate repo/runtime health with the starter-safe checks:

```bash
ship doctor
ship verify --profile fast
```

If you are using the default Kamal workflow, continue with the deployment steps documented in:

- `docs/guides/04-deployment-kamal.md`

## What You Should Have Now

- A booting GoShip app created by `ship new`
- Local DB migrated with `ship db:migrate`
- Dev loop running via `ship dev`
- Starter landing/auth/account/admin/home/profile routes available for further app-specific work
- Test and verification commands passing
