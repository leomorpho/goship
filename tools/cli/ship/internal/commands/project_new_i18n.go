package commands

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

func resolveNewI18nOptions(opts NewProjectOptions, d NewDeps) (NewProjectOptions, error) {
	if opts.I18nLocalePackSet && !opts.I18nSet {
		opts.I18nEnabled = true
		opts.I18nSet = true
	}
	if opts.I18nSet && !opts.I18nEnabled && opts.I18nLocalePackSet {
		return opts, fmt.Errorf("cannot use --i18n-locale-pack with --no-i18n")
	}

	if opts.I18nSet {
		if opts.I18nEnabled && strings.TrimSpace(opts.I18nLocalePack) == "" {
			opts.I18nLocalePack = i18nLocalePackStarter
		}
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
	if enabled && strings.TrimSpace(opts.I18nLocalePack) == "" {
		opts.I18nLocalePack = i18nLocalePackStarter
	}
	return opts, nil
}

func i18nScaffoldFiles(opts NewProjectOptions) map[string]string {
	if !opts.I18nEnabled {
		return nil
	}

	packName := strings.TrimSpace(opts.I18nLocalePack)
	if packName == "" {
		packName = i18nLocalePackStarter
	}
	locales, ok := i18nLocalePacks[packName]
	if !ok {
		return nil
	}

	displayName := starterDisplayName(opts.Name)
	files := make(map[string]string, len(locales))
	for _, localeCode := range locales {
		files[filepath.Join(opts.AppPath, "locales", localeCode+".toml")] = renderStarterLocaleFile(displayName, localeCode, packName)
	}
	return files
}

func renderStarterLocaleFile(appName string, localeCode string, packName string) string {
	landingTitle := starterLandingTitleForLocale(localeCode)
	if packName == i18nLocalePackTop15 {
		return fmt.Sprintf(`# Starter locale scaffold (%s pack). Review translations before production.
"app.name" = "%s"
"pages.landing.title" = "%s"
`, packName, appName, landingTitle)
	}
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
		packName := strings.TrimSpace(opts.I18nLocalePack)
		if packName == "" {
			packName = i18nLocalePackStarter
		}
		codes := sortedLocalePackCodes(packName)
		fmt.Fprintf(w, "I18n enabled: scaffolded %d locale files (%s pack: %s).\n", len(codes), packName, strings.Join(codes, ", "))
		return
	}
	fmt.Fprintln(w, "I18n disabled by default. You can enable and migrate later with ship i18n tooling and LLM-guided doctor loops.")
}

const (
	i18nLocalePackStarter = "starter"
	i18nLocalePackTop15   = "top15"
)

var i18nLocalePacks = map[string][]string{
	i18nLocalePackStarter: {"en", "fr"},
	i18nLocalePackTop15: {
		"ar",
		"de",
		"en",
		"es",
		"fr",
		"hi",
		"id",
		"it",
		"ja",
		"ko",
		"nl",
		"pt",
		"ru",
		"tr",
		"zh",
	},
}

func isValidI18nLocalePack(pack string) bool {
	_, ok := i18nLocalePacks[strings.TrimSpace(pack)]
	return ok
}

func sortedLocalePackCodes(pack string) []string {
	codes := append([]string(nil), i18nLocalePacks[pack]...)
	sort.Strings(codes)
	return codes
}

func starterLandingTitleForLocale(localeCode string) string {
	switch localeCode {
	case "fr":
		return "Bienvenue"
	default:
		return "Welcome"
	}
}
