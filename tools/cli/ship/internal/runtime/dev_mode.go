package runtime

import (
	appconfig "github.com/leomorpho/goship/config"
)

// ResolveDevDefaultMode determines the default `ship dev` execution mode.
// Convention:
// - single-node profile => canonical app-on web loop (`web`)
// - distributed profile => full multiprocess loop (`all`)
// - fallback => web-only mode (`web`)
func ResolveDevDefaultMode() (string, error) {
	cfg, err := appconfig.GetConfig()
	if err != nil {
		return "", err
	}

	switch cfg.Runtime.Profile {
	case appconfig.RuntimeProfileDistributed:
		return "all", nil
	case appconfig.RuntimeProfileSingleNode, appconfig.RuntimeProfileServerDB, "":
		return "web", nil
	default:
		return "web", nil
	}
}
