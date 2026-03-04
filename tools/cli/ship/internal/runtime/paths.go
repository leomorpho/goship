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

type atlasConfig struct {
	App struct {
		Environment string `yaml:"environment"`
	} `yaml:"app"`
	Database struct {
		DbMode            string `yaml:"dbMode"`
		Hostname          string `yaml:"hostname"`
		Port              uint16 `yaml:"port"`
		User              string `yaml:"user"`
		Password          string `yaml:"password"`
		DatabaseNameLocal string `yaml:"databaseNameLocal"`
		DatabaseNameProd  string `yaml:"databaseNameProd"`
		TestDatabase      string `yaml:"testDatabase"`
		SslMode           string `yaml:"sslMode"`
		SslCertPath       string `yaml:"sslCertPath"`
	} `yaml:"database"`
}

func ResolveAtlasDBURL() (string, error) {
	if u := strings.TrimSpace(os.Getenv("DATABASE_URL")); u != "" {
		return u, nil
	}
	if u := strings.TrimSpace(os.Getenv("PAGODA_DATABASE_URL")); u != "" {
		return "", errors.New("PAGODA_DATABASE_URL is not supported; use DATABASE_URL")
	}

	cfg, err := loadAtlasConfig()
	if err != nil {
		return "", err
	}
	if strings.EqualFold(cfg.Database.DbMode, "embedded") {
		return "", errors.New("database mode is embedded; set DATABASE_URL or switch runtime profile to server-db for atlas migrations")
	}

	env := strings.TrimSpace(os.Getenv("APP_ENV"))
	if env == "" {
		env = strings.TrimSpace(cfg.App.Environment)
	}
	if env == "" {
		env = "local"
	}

	dbName := strings.TrimSpace(cfg.Database.DatabaseNameLocal)
	switch env {
	case "production":
		dbName = strings.TrimSpace(cfg.Database.DatabaseNameProd)
	case "test":
		if t := strings.TrimSpace(cfg.Database.TestDatabase); t != "" {
			dbName = t
		}
	}
	if dbName == "" {
		return "", errors.New("database name is empty in config; set DATABASE_URL or database.databaseNameLocal")
	}
	if strings.TrimSpace(cfg.Database.Hostname) == "" || cfg.Database.Port == 0 {
		return "", errors.New("database host/port missing in config; set DATABASE_URL or database hostname/port")
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

func loadAtlasConfig() (atlasConfig, error) {
	var cfg atlasConfig
	configDir, err := findConfigDir()
	if err != nil {
		return cfg, err
	}
	if err := unmarshalYAMLFile(filepath.Join(configDir, "application.yaml"), &cfg); err != nil {
		return cfg, err
	}

	env := strings.TrimSpace(os.Getenv("APP_ENV"))
	if env == "" {
		env = strings.TrimSpace(cfg.App.Environment)
	}
	if env == "" {
		env = "local"
	}
	envFile := filepath.Join(configDir, "environments", env+".yaml")
	if HasFile(envFile) {
		if err := unmarshalYAMLFile(envFile, &cfg); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func findConfigDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		cfgDir := filepath.Join(dir, "config")
		if HasFile(filepath.Join(cfgDir, "application.yaml")) {
			return cfgDir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("config/application.yaml not found; set DATABASE_URL")
		}
		dir = parent
	}
}

func unmarshalYAMLFile(path string, dst any) error {
	cfg, ok := dst.(*atlasConfig)
	if !ok {
		return errors.New("unsupported config type")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return parseAtlasConfigYAML(string(b), cfg)
}

func parseAtlasConfigYAML(content string, cfg *atlasConfig) error {
	section := ""
	lines := strings.Split(content, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(raw, " ") && strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		if !strings.HasPrefix(raw, "  ") {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimSpace(raw), ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = normalizeYAMLScalar(value)
		switch section {
		case "app":
			if key == "environment" {
				cfg.App.Environment = value
			}
		case "database":
			switch key {
			case "dbMode":
				cfg.Database.DbMode = value
			case "hostname":
				cfg.Database.Hostname = value
			case "port":
				if v, err := strconv.Atoi(value); err == nil && v > 0 && v <= 65535 {
					cfg.Database.Port = uint16(v)
				}
			case "user":
				cfg.Database.User = value
			case "password":
				cfg.Database.Password = value
			case "databaseNameLocal":
				cfg.Database.DatabaseNameLocal = value
			case "databaseNameProd":
				cfg.Database.DatabaseNameProd = value
			case "testDatabase":
				cfg.Database.TestDatabase = value
			case "sslMode":
				cfg.Database.SslMode = value
			case "sslCertPath":
				cfg.Database.SslCertPath = value
			}
		}
	}
	return nil
}

func normalizeYAMLScalar(v string) string {
	s := strings.TrimSpace(v)
	if idx := strings.Index(s, "#"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	s = strings.Trim(s, `"`)
	s = strings.Trim(s, `'`)
	return s
}
