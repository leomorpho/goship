package jobs

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Backend string

const (
	BackendSQL      Backend = "sql"
	BackendRedis    Backend = "redis"
	BackendBacklite Backend = "backlite"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type Config struct {
	Backend Backend
	SQLDB   *sql.DB
	Redis   RedisConfig
}

func (c Config) Validate() error {
	switch c.Backend {
	case BackendSQL:
		if c.SQLDB == nil {
			return errors.New("jobs backend sql requires SQL DB")
		}
		if c.hasRedisSettings() {
			return errors.New("jobs backend sql forbids redis settings")
		}
		return nil
	case BackendBacklite:
		if c.SQLDB == nil {
			return errors.New("jobs backend backlite requires SQL DB")
		}
		if c.hasRedisSettings() {
			return errors.New("jobs backend backlite forbids redis settings")
		}
		return nil
	case BackendRedis:
		if c.SQLDB != nil {
			return errors.New("jobs backend redis forbids SQL settings")
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
