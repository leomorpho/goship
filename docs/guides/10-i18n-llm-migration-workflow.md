# I18n LLM Migration Workflow

This guide defines the deterministic i18n migration loop for humans and LLM agents.

## Canonical Command Loop

1. `ship i18n:init`
2. `ship i18n:scan --format json --limit 50`
3. `ship i18n:instrument --apply --limit 50`
4. `ship doctor --json`
5. `ship i18n:missing`
6. `ship i18n:unused`
7. Repeat from step 2 until no actionable findings remain.

## Fix-One-Issue Loop

1. Run `ship i18n:scan --format json --limit 1`.
2. If confidence is `high` and context is supported, run `ship i18n:instrument --apply --limit 1`.
3. If confidence is not `high` or rewrite is skipped, apply a manual key migration.
4. Run `ship doctor --json`.
5. Commit once tests pass.

## JSON Schemas (v1)

`ship i18n:scan --format json`:

```json
{
  "version": "v1",
  "issues": [
    {
      "id": "I18N-XXXXXXXXXXXX",
      "kind": "missing_i18n_key",
      "severity": "warning",
      "file": "app/views/...",
      "line": 12,
      "column": 8,
      "message": "hardcoded user-facing string ...",
      "suggested_key": "app.example_key",
      "confidence": "high"
    }
  ]
}
```

`ship i18n:instrument`:

```json
{
  "version": "v1",
  "apply": false,
  "applied": 0,
  "rewrites": [
    {
      "id": "I18N-XXXXXXXXXXXX",
      "file": "app/web/controllers/...",
      "line": 42,
      "column": 10,
      "before": "c.String(..., \"Text\")",
      "after": "c.String(..., i18n.T(...))",
      "message": "rewrite plan entry",
      "suggested_key": "app.example_key",
      "confidence": "high"
    }
  ],
  "skipped": [
    {
      "id": "I18N-XXXXXXXXXXXX",
      "file": "app/views/...",
      "line": 17,
      "column": 3,
      "message": "scanner finding",
      "reason": "unsupported_source_type",
      "confidence": "medium"
    }
  ]
}
```

`ship doctor --json`:

```json
{
  "ok": false,
  "issues": [
    {
      "type": "DX029",
      "file": "app/web/controllers/example.go",
      "detail": "i18n literal app/web/controllers/example.go:12:5 (I18N-...)",
      "severity": "warning"
    }
  ]
}
```

## Issue ID Contract

1. Scan findings use `I18N-<12 uppercase hex>`.
2. Plural/select completeness findings use `I18N-C-<10 uppercase hex>`.
3. Doctor strict-mode gate uses issue code `DX029` and includes the underlying i18n finding ID in `detail`.

## Confidence Tiers

1. `high`: eligible for automated rewrite in supported contexts (current safe path is selected Go controller string responses).
2. `medium`: report-only; use manual migration.

## Strict-Mode Rollout

1. `PAGODA_I18N_STRICT_MODE=off`: no strict enforcement.
2. `PAGODA_I18N_STRICT_MODE=warn`: findings are warnings (`DX029`) and non-blocking.
3. `PAGODA_I18N_STRICT_MODE=error`: findings are blocking errors (`DX029`).

Use `.i18n-allowlist` for intentional exceptions:

1. Add a finding ID (`I18N-...` or `I18N-C-...`), or
2. Add a location key (`path/to/file.go:line`).

## Recommended LLM Policy

1. Never bulk-rewrite all findings in one pass.
2. Limit each loop (`--limit 1..50`) and re-run diagnostics after each applied batch.
3. Prefer `high` confidence rewrites first.
4. Run `go test ./...` and `ship doctor --json` before each commit.

## Starter Locale Policy

`ship new` supports:

1. Starter locales (`en`, `fr`) when i18n is enabled.

Maintenance policy:

1. `en.toml` remains canonical source for key shape.
2. New keys are introduced in `en.toml` first, then synchronized to other locales via `ship make:locale`, `ship i18n:missing`, and `ship i18n:normalize`.
3. Starter-pack translations must be reviewed before production use; scaffold values are bootstrapping defaults, not guaranteed fully translated product copy.
