package main

import (
	"fmt"

	"github.com/leomorpho/goship/config"
)

func validateWorkerConfig(cfg config.Config) error {
	if cfg.Adapters.Jobs != "asynq" {
		return fmt.Errorf("worker requires jobs adapter \"asynq\"; current adapter is %q", cfg.Adapters.Jobs)
	}
	return nil
}
