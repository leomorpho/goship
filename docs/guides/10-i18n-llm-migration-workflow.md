# I18n LLM Migration Workflow

This guide defines the deterministic i18n migration loop for humans and LLM agents.

## Canonical Command Loop

1. `ship i18n:init`
2. `ship i18n:scan --format json --limit 50`
3. `ship i18n:instrument --apply --limit 50`
4. `ship doctor --json`
5. `ship i18n:missing`
6. `ship i18n:compile`
7. `ship i18n:unused`
8. Repeat from step 2 until no actionable findings remain.

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

1. `high`: eligible for automated rewrite in supported contexts (current safe path is selected Go controller `*.String` response literals).
2. `medium`: report-only; use manual migration.

## Strict-Mode Rollout

1. `PAGODA_I18N_STRICT_MODE=off`: no strict enforcement.
2. `PAGODA_I18N_STRICT_MODE=warn`: findings are warnings (`DX029`) and non-blocking.
3. `PAGODA_I18N_STRICT_MODE=error`: findings are blocking errors (`DX029`).

Use `.i18n-allowlist` for intentional exceptions:

1. Add a stable finding ID (`I18N-S-...`) for literal findings (preferred).
2. Add a finding ID (`I18N-...` or `I18N-C-...`) for compatibility.
3. Add a location key (`path/to/file.go:line`) only for legacy compatibility.

## I18n Coverage and Enforcement Policy

Mandatory i18n-backed surfaces:

1. Go HTTP handlers/controllers for user-facing text (HTML and plain text responses).
2. Templ-rendered HTML copy in `app/views/**` and `modules/*/views/**`.
3. Islands UI copy in `frontend/islands/**` that is visible to end users.
4. API human-readable JSON messages (while machine `code` values remain locale-invariant).
5. User-facing email template copy.

Explicit exclusions (not required to be translation-key backed):

1. Logs, metrics labels, tracing tags, and other operator-only diagnostics.
2. SQL text, migration text, schema names, route names, and internal constants.
3. Machine-facing API error `code` values and protocol field names.
4. Test-only literals (`*_test.go`, fixtures, test snapshots) unless validating localized behavior.
5. Internal-only panic/debug strings that are never user-visible.

Enforcement mapping by command/policy:

1. `ship i18n:scan`:
   - Finds hardcoded user-facing literals in supported Go/templ/islands paths.
   - Emits deterministic JSON diagnostics and confidence tier metadata.
2. `ship i18n:instrument`:
   - Applies high-confidence rewrites only for currently supported safe patterns.
   - Lower confidence findings remain report-only/manual.
3. `ship doctor --json` (`DX029`):
   - Enforces strict-mode hardcoded literal checks on controllers/views/islands.
   - Enforces plural/select completeness for `I18n.TC(...)` and `I18n.TS(...)`.
   - Supports intentional exceptions through `.i18n-allowlist`.
4. `ship i18n:missing`:
   - Reports missing/empty locale keys against baseline English catalogs.
5. `ship i18n:unused`:
   - Reports keys not referenced by `I18n.T(...)`/`i18n.T(...)` usage.

Current known enforcement gaps (explicit, temporary):

1. Automatic rewrites are intentionally narrower than scanner detection coverage.
2. `.i18n-allowlist` still permits legacy `path:line` selectors for migration safety, but stable IDs should be preferred.
3. Scanner/doctor focus on application user copy, not operational/internal developer text.

## Recommended LLM Policy

1. Never bulk-rewrite all findings in one pass.
2. Limit each loop (`--limit 1..50`) and re-run diagnostics after each applied batch.
3. Prefer `high` confidence rewrites first.
4. Run `go test ./...` and `ship doctor --json` before each commit.

## CI Profile (Strict)

Use `ship i18n:ci` for deterministic strict i18n CI gates in i18n-enabled apps.

Recommended rollout:

1. Start with migration loops and `PAGODA_I18N_STRICT_MODE=warn`.
2. Clean scanner + completeness findings and adopt stable allowlist entries only where intentional.
3. Switch to `PAGODA_I18N_STRICT_MODE=error`.
4. Gate CI with:
   - `go test ./...`
   - `ship i18n:ci`

## Starter Locale Policy

`ship new` supports:

1. Starter locales (`en`, `fr`) when i18n is enabled.

Maintenance policy:

1. `en.toml` remains canonical source for key shape.
2. New keys are introduced in `en.toml` first, then synchronized to other locales via `ship make:locale`, `ship i18n:missing`, and `ship i18n:normalize`.
3. Starter-pack translations must be reviewed before production use; scaffold values are bootstrapping defaults, not guaranteed fully translated product copy.

## Translator Workflow Contract

Ownership and baseline policy:

1. `locales/en.toml` is the canonical baseline owned by application developers.
2. Non-English locale files are translator-owned artifacts synchronized from English keys.
3. Key renames/deletions must be intentional and reviewed because they invalidate downstream translations.

Key lifecycle contract:

1. Add new key in `en.toml` with final placeholder structure.
2. Run `ship make:locale <code>` (for new locales) or sync existing locale files.
3. Run `ship i18n:missing` to identify untranslated keys.
4. Fill translations and re-run `ship i18n:missing` until clean.
5. Run `ship i18n:normalize` and `ship i18n:compile` for deterministic artifacts.
6. Run `ship i18n:unused` to remove stale keys before merge.

Translation quality gates:

1. No empty string values for required production locales.
2. No placeholder passthrough values that simply duplicate key names.
3. `TC`/`TS` keys must include required fallback forms (`*.other` and at least one non-`other` variant).
4. `ship doctor --json` in strict mode should be clean (or explicitly allowlisted).

Production-readiness expectations:

1. Scaffolded locale values are bootstrap defaults, not production-grade copy.
2. Locale packs must be reviewed by humans before release in that locale.
3. Teams should roll out strict mode gradually (`off -> warn -> error`) as translation coverage matures.

## Installable Adapter Contract

Required runtime interface (`core.I18n`):

1. `DefaultLanguage() string`
2. `SupportedLanguages() []string`
3. `NormalizeLanguage(raw string) string`
4. `T(ctx context.Context, key string, templateData ...map[string]any) string`
5. `TC(ctx context.Context, key string, count any, templateData ...map[string]any) string`
6. `TS(ctx context.Context, key string, choice string, templateData ...map[string]any) string`

Runtime semantics:

1. `DefaultLanguage` must be non-empty.
2. `SupportedLanguages` must include the default language.
3. `NormalizeLanguage` must fall back to default for unsupported input.
4. Missing key behavior must be deterministic (default adapter returns key/fallback text path).
5. `TC` and `TS` must support fallback behavior required by this guide (`*.other` rules and default locale fallback).

Compatibility harness:

1. Use `framework/core/contracttests.RunI18nContract` in adapter tests.
2. Include adapter-specific tests for plural/select behavior and locale fallback chain.
3. Keep adapter behavior documented in one place (this guide) so LLM and human workflows stay aligned.
