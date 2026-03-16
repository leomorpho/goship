# I18n Adapter Contract

This guide defines the compatibility contract for installable i18n adapters.

## Required Interface

Adapters must satisfy `core.I18n`:

1. `DefaultLanguage() string`
2. `SupportedLanguages() []string`
3. `NormalizeLanguage(raw string) string`
4. `T(ctx context.Context, key string, templateData ...map[string]any) string`

## Runtime Semantics

1. `DefaultLanguage` must return a non-empty locale code.
2. `SupportedLanguages` must include `DefaultLanguage`.
3. `NormalizeLanguage` must fall back to `DefaultLanguage` for unsupported inputs.
4. `T` must return the key for missing translations (stable fallback behavior).
5. `T` should support template interpolation through `templateData`.

## Compatibility Harness

Shared contract tests live in `framework/core/contracttests`:

- `RunI18nContract(t, subject)` runs baseline adapter checks.
- Adapter implementations should wire this in their own package tests.

Default adapter example:

```go
contracttests.RunI18nContract(t, contracttests.I18nContractSubject{
    Name:               "modules/i18n default adapter",
    KnownDefaultKey:    "app.welcome",
    KnownDefaultResult: "Welcome",
    Build: func(t *testing.T) contracttests.I18nContractAdapter {
        // construct adapter with test locales...
    },
})
```

## Implementer Checklist

1. Implement `core.I18n`.
2. Add adapter-specific behavior tests (plurals, fallback chain, parsing).
3. Add `RunI18nContract` to adapter tests.
4. Document locale file format and loading behavior.
5. Document missing-key behavior and fallback order.
