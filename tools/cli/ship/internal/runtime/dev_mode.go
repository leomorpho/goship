package runtime

import (
	"strings"

	appconfig "github.com/leomorpho/goship/config"
)

// ResolveDevDefaultMode determines the default `ship dev` execution mode.
// Convention:
// - jobs adapter `asynq` => full multiprocess dev mode (`all`)
// - all other adapters => web-only mode (`web`)
func ResolveDevDefaultMode() (string, error) {
	cfg, err := appconfig.GetConfig()
	if err != nil {
		return "", err
	}

	if strings.EqualFold(strings.TrimSpace(cfg.Adapters.Jobs), "asynq") {
		return "all", nil
	}
	return "web", nil
}
