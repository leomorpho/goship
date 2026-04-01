package runtime

import (
	"bufio"
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
	appconfig "github.com/leomorpho/goship/v2/config"
)

func FindGoModule(start string) (string, string, error) {
	dir := start
	for {
		goMod := filepath.Join(dir, "go.mod")
		f, err := os.Open(goMod)
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "module ") {
					modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
					if modulePath == "" {
						return "", "", errors.New("empty module path in go.mod")
					}
					return dir, modulePath, nil
				}
			}
			if scanErr := scanner.Err(); scanErr != nil {
				return "", "", scanErr
			}
			return "", "", errors.New("module line not found in go.mod")
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", errors.New("go.mod not found from current path")
		}
		dir = parent
	}
}

func HasFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func HasMakefile() bool {
	wd, err := os.Getwd()
	if err != nil {
		return false
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

type runtimeConfig struct {
	DatabaseURL       string `env:"DATABASE_URL"`
	LegacyDatabaseURL string `env:"PAGODA_DATABASE_URL"`
	DBDriver          string `env:"DB_DRIVER,PAGODA_DATABASE_DRIVER,PAGODA_DB_DRIVER"`
	AppEnvironment    string `env:"PAGODA_APP_ENVIRONMENT"`
}

func ResolveDBURL() (string, error) {
	runtimeEnv, err := loadRuntimeConfig()
	if err != nil {
		return "", err
	}
	if u := strings.TrimSpace(runtimeEnv.DatabaseURL); u != "" {
		return u, nil
	}
	if u := strings.TrimSpace(runtimeEnv.LegacyDatabaseURL); u != "" {
		return "", errors.New("PAGODA_DATABASE_URL is not supported; use DATABASE_URL")
	}

	cfg, err := appconfig.GetConfig()
	if err != nil {
		return "", err
	}
	if strings.EqualFold(string(cfg.Database.DbMode), "embedded") {
		dsn := strings.TrimSpace(cfg.Database.EmbeddedConnection)
		if dsn == "" {
			dsn = strings.TrimSpace(cfg.Database.Path)
		}
		if dsn == "" {
			return "", errors.New("database mode is embedded but sqlite DSN/path is empty")
		}
		return "sqlite://" + dsn, nil
	}

	env := strings.TrimSpace(os.Getenv("APP_ENV"))
	if env == "" {
		env = strings.TrimSpace(runtimeEnv.AppEnvironment)
	}
	if env == "" {
		env = strings.TrimSpace(string(cfg.App.Environment))
	}
	if env == "" {
		env = "local"
	}

	dbName := strings.TrimSpace(cfg.Database.DatabaseNameLocal)
	switch env {
	case "production", "prod":
		dbName = strings.TrimSpace(cfg.Database.DatabaseNameProd)
	case "test":
		if t := strings.TrimSpace(cfg.Database.TestDatabase); t != "" {
			dbName = t
		}
	}
	if dbName == "" {
		return "", errors.New("database name is empty in config; set DATABASE_URL or PAGODA_DATABASE_DATABASENAMELOCAL")
	}
	if strings.TrimSpace(cfg.Database.Hostname) == "" || cfg.Database.Port == 0 {
		return "", errors.New("database host/port missing in config; set DATABASE_URL or PAGODA_DATABASE_HOSTNAME/PAGODA_DATABASE_PORT")
	}

	query := url.Values{}
	sslMode := strings.TrimSpace(cfg.Database.SslMode)
	if sslMode == "" {
		sslMode = "disable"
	}
	query.Set("sslmode", sslMode)
	if cert := strings.TrimSpace(cfg.Database.SslCertPath); cert != "" {
		query.Set("sslrootcert", cert)
	}

	u := &url.URL{
		Scheme:   "postgresql",
		Host:     net.JoinHostPort(cfg.Database.Hostname, strconv.Itoa(int(cfg.Database.Port))),
		Path:     "/" + dbName,
		RawQuery: query.Encode(),
	}
	if user := strings.TrimSpace(cfg.Database.User); user != "" {
		u.User = url.UserPassword(user, cfg.Database.Password)
	}
	return u.String(), nil
}

func ResolveDBDriver() (string, error) {
	runtimeEnv, err := loadRuntimeConfig()
	if err != nil {
		return "", err
	}
	if driver := normalizeRuntimeDBDriver(runtimeEnv.DBDriver); driver != "" {
		return driver, nil
	}
	if strings.TrimSpace(runtimeEnv.DBDriver) != "" {
		return "", errors.New("unsupported DB_DRIVER; supported values are postgres, mysql, sqlite")
	}

	cfg, err := appconfig.GetConfig()
	if err != nil {
		return "", err
	}

	if strings.EqualFold(string(cfg.Database.DbMode), "embedded") {
		return "sqlite", nil
	}
	if driver := normalizeRuntimeDBDriver(string(cfg.Database.Driver)); driver != "" {
		return driver, nil
	}
	if strings.EqualFold(string(cfg.Database.DbMode), "standalone") {
		return "postgres", nil
	}
	return "", errors.New("database driver is empty; set DB_DRIVER")
}

func normalizeRuntimeDBDriver(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "postgres", "postgresql", "pgx":
		return "postgres"
	case "mysql", "mariadb":
		return "mysql"
	case "sqlite", "sqlite3":
		return "sqlite"
	default:
		return ""
	}
}

func loadRuntimeConfig() (runtimeConfig, error) {
	var cfg runtimeConfig
	if path, ok := findDotEnv(); ok {
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return cfg, err
		}
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func ResolveAppEnvironment() string {
	if env := strings.TrimSpace(os.Getenv("APP_ENV")); env != "" {
		return env
	}
	cfg, err := loadRuntimeConfig()
	if err == nil && strings.TrimSpace(cfg.AppEnvironment) != "" {
		return strings.TrimSpace(cfg.AppEnvironment)
	}
	return strings.TrimSpace(os.Getenv("PAGODA_APP_ENVIRONMENT"))
}

func findDotEnv() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	dir := wd
	for {
		path := filepath.Join(dir, ".env")
		if HasFile(path) {
			return path, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
