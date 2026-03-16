package foundation

import (
	"fmt"
	"path/filepath"
	"runtime"

	i18nmodule "github.com/leomorpho/goship/modules/i18n"
)

func (c *Container) initI18n() {
	service, err := i18nmodule.NewService(i18nmodule.Options{
		LocaleDir:       localeDir(),
		DefaultLanguage: "en",
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialize i18n service: %v", err))
	}
	c.I18n = service
}

func localeDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "locales"
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return filepath.Join(root, "locales")
}
