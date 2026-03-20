package i18n

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/core"
)

func New(cfg *config.Config) core.I18n {
	if cfg == nil {
		return nil
	}
	if !cfg.I18n.Enabled {
		return nil
	}

	defaultLanguage := cfg.I18n.DefaultLanguage
	if defaultLanguage == "" {
		defaultLanguage = "en"
	}

	service, err := NewService(Options{
		LocaleDir:       LocaleDir(),
		DefaultLanguage: defaultLanguage,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialize i18n service: %v", err))
	}
	return service
}

func LocaleDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "locales"
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return filepath.Join(root, "locales")
}
