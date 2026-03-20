package commands

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

func runGooseStatus(d DBDeps, dbURL string) int {
	return runGooseStatusForDir(d, dbURL, d.GooseDir)
}

func runGooseStatusForDir(d DBDeps, dbURL string, dir string) int {
	driver, conn, err := gooseTarget(dbURL)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve goose driver/url: %v\n", err)
		return 1
	}
	return d.RunGoose("-dir", dir, driver, conn, "status")
}

func runGooseUp(d DBDeps, dbURL string) int {
	return runGooseUpForDir(d, dbURL, d.GooseDir)
}

func runGooseUpForDir(d DBDeps, dbURL string, dir string) int {
	driver, conn, err := gooseTarget(dbURL)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve goose driver/url: %v\n", err)
		return 1
	}
	return d.RunGoose("-dir", dir, driver, conn, "up")
}

func runGooseReset(d DBDeps, dbURL string) int {
	return runGooseResetForDir(d, dbURL, d.GooseDir)
}

func runGooseResetForDir(d DBDeps, dbURL string, dir string) int {
	driver, conn, err := gooseTarget(dbURL)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve goose driver/url: %v\n", err)
		return 1
	}
	return d.RunGoose("-dir", dir, driver, conn, "reset")
}

func runGooseDown(d DBDeps, dbURL string, amount string) int {
	driver, conn, err := gooseTarget(dbURL)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve goose driver/url: %v\n", err)
		return 1
	}
	if amount == "1" {
		return d.RunGoose("-dir", d.GooseDir, driver, conn, "down")
	}
	return d.RunGoose("-dir", d.GooseDir, driver, conn, "down-to", amount)
}

func gooseTarget(dbURL string) (driver string, conn string, err error) {
	if strings.HasPrefix(dbURL, "sqlite://") {
		dsn := strings.TrimPrefix(dbURL, "sqlite://")
		if strings.TrimSpace(dsn) == "" {
			return "", "", fmt.Errorf("sqlite URL is missing DSN")
		}
		return "sqlite3", dsn, nil
	}
	if strings.HasPrefix(dbURL, "sqlite3://") {
		dsn := strings.TrimPrefix(dbURL, "sqlite3://")
		if strings.TrimSpace(dsn) == "" {
			return "", "", fmt.Errorf("sqlite3 URL is missing DSN")
		}
		return "sqlite3", dsn, nil
	}
	u, parseErr := url.Parse(dbURL)
	if parseErr != nil {
		return "", "", parseErr
	}
	switch strings.ToLower(u.Scheme) {
	case "postgres", "postgresql":
		return "postgres", dbURL, nil
	case "mysql":
		return "mysql", dbURL, nil
	case "sqlite", "sqlite3":
		dsn := strings.TrimPrefix(dbURL, u.Scheme+"://")
		if strings.TrimSpace(dsn) == "" {
			return "", "", fmt.Errorf("%s URL is missing DSN", u.Scheme)
		}
		return "sqlite3", dsn, nil
	default:
		return "", "", fmt.Errorf("unsupported database scheme %q for goose (supported: postgres, mysql, sqlite)", u.Scheme)
	}
}

func runGooseUpAll(d DBDeps, dbURL string) int {
	dirs, err := resolveGooseDirs(d)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve migration directories: %v\n", err)
		return 1
	}
	for _, dir := range dirs {
		if code := runGooseUpForDir(d, dbURL, dir); code != 0 {
			return code
		}
	}
	return 0
}

func runGooseStatusAll(d DBDeps, dbURL string) int {
	dirs, err := resolveGooseDirs(d)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve migration directories: %v\n", err)
		return 1
	}
	for _, dir := range dirs {
		fmt.Fprintf(d.Out, "== %s ==\n", gooseDirLabel(dir))
		if code := runGooseStatusForDir(d, dbURL, dir); code != 0 {
			return code
		}
	}
	return 0
}

func runGooseResetAll(d DBDeps, dbURL string) int {
	dirs, err := resolveGooseDirs(d)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve migration directories: %v\n", err)
		return 1
	}
	// Reset in reverse order to reduce cross-module dependency teardown issues.
	for i := len(dirs) - 1; i >= 0; i-- {
		if code := runGooseResetForDir(d, dbURL, dirs[i]); code != 0 {
			return code
		}
	}
	return 0
}

func resolveGooseDirs(d DBDeps) ([]string, error) {
	dirs := []string{d.GooseDir}
	if d.FindGoModule == nil {
		return dirs, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.Join(root, "config", "modules.yaml")
	if !pathExists(manifestPath) {
		return dirs, nil
	}
	manifest, err := rt.LoadModulesManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	for _, module := range manifest.Modules {
		migrationDir := filepath.ToSlash(filepath.Join("modules", module, "db", "migrate", "migrations"))
		absDir := filepath.Join(root, filepath.FromSlash(migrationDir))
		if !isDir(absDir) {
			return nil, fmt.Errorf("enabled module %q missing migrations directory: %s", module, migrationDir)
		}
		dirs = append(dirs, migrationDir)
	}
	return dirs, nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func gooseDirLabel(dir string) string {
	normalized := filepath.ToSlash(strings.TrimSpace(dir))
	if normalized == filepath.ToSlash(filepath.Join("db", "migrate", "migrations")) {
		return "core migrations"
	}
	if strings.HasPrefix(normalized, "modules/") {
		parts := strings.Split(normalized, "/")
		if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
			return fmt.Sprintf("module %s migrations", parts[1])
		}
	}
	return fmt.Sprintf("migrations: %s", normalized)
}
