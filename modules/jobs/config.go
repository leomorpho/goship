package jobs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/leomorpho/goship/db/ent"
)

type Backend string

const (
	BackendSQL   Backend = "sql"
	BackendRedis Backend = "redis"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type Config struct {
	Backend   Backend
	EntClient *ent.Client
	Redis     RedisConfig
}

func (c Config) Validate() error {
	switch c.Backend {
	case BackendSQL:
		if c.EntClient == nil {
			return errors.New("jobs backend sql requires Ent client")
		}
		if c.hasRedisSettings() {
			return errors.New("jobs backend sql forbids redis settings")
		}
		return nil
	case BackendRedis:
		if c.EntClient != nil {
			return errors.New("jobs backend redis forbids Ent client")
		}
		if strings.TrimSpace(c.Redis.Addr) == "" {
			return errors.New("jobs backend redis requires redis address")
		}
		return nil
	default:
		return fmt.Errorf("unsupported jobs backend %q", c.Backend)
	}
}

func (c Config) hasRedisSettings() bool {
	return strings.TrimSpace(c.Redis.Addr) != "" || strings.TrimSpace(c.Redis.Password) != "" || c.Redis.DB != 0
}
