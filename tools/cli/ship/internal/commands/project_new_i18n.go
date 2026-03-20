package commands

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func resolveNewI18nOptions(opts NewProjectOptions, d NewDeps) (NewProjectOptions, error) {
	if opts.I18nSet {
		return opts, nil
	}
	if d.IsInteractive == nil || d.PromptI18nEnable == nil || !d.IsInteractive() {
		return opts, nil
	}

	enabled, err := d.PromptI18nEnable()
	if err != nil {
		return opts, fmt.Errorf("failed to read i18n prompt: %w", err)
	}
	opts.I18nEnabled = enabled
	opts.I18nSet = true
	return opts, nil
}

var starterI18nLocales = []string{"en", "fr"}

func i18nScaffoldFiles(opts NewProjectOptions) map[string]string {
	if !opts.I18nEnabled {
		return nil
	}

	displayName := starterDisplayName(opts.Name)
	files := make(map[string]string, len(starterI18nLocales))
	for _, localeCode := range starterI18nLocales {
		files[filepath.Join(opts.AppPath, "locales", localeCode+".toml")] = renderStarterLocaleFile(displayName, localeCode)
	}
	return files
}

func renderStarterLocaleFile(appName string, localeCode string) string {
	landingTitle := starterLandingTitleForLocale(localeCode)
	return fmt.Sprintf(`# Starter locale scaffold.
"app.name" = "%s"
"pages.landing.title" = "%s"
`, appName, landingTitle)
}

func rewriteStarterI18nTemplate(relPath, content string, opts NewProjectOptions) string {
	if !opts.I18nEnabled {
		return content
	}
	if relPath != "app/foundation/container.go" {
		return content
	}
	return strings.Replace(content, `[]string{"auth", "profile"}`, `[]string{"auth", "profile", "i18n"}`, 1)
}

func printNewI18nStatus(w io.Writer, opts NewProjectOptions) {
	if opts.I18nEnabled {
		fmt.Fprintf(w, "I18n enabled: scaffolded %d locale files (%s).\n", len(starterI18nLocales), strings.Join(starterI18nLocales, ", "))
		return
	}
	fmt.Fprintln(w, "I18n disabled by default. You can enable and migrate later with ship i18n tooling and LLM-guided doctor loops.")
}

func starterLandingTitleForLocale(localeCode string) string {
	switch localeCode {
	case "fr":
		return "Bienvenue"
	default:
		return "Welcome"
	}
}
