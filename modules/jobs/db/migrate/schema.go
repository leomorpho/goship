package migrate

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

const initJobsMigration = "migrations/20260305195000_init_jobs.sql"

func LoadInitJobsUpSQL() (string, error) {
	content, err := migrationFS.ReadFile(initJobsMigration)
	if err != nil {
		return "", err
	}
	up := extractGooseUp(string(content))
	if strings.TrimSpace(up) == "" {
		return "", fmt.Errorf("empty goose up section in %s", initJobsMigration)
	}
	return up, nil
}

func extractGooseUp(content string) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "-- +goose Up":
			continue
		case "-- +goose Down":
			return b.String()
		default:
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}
