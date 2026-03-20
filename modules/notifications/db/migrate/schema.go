package migrate

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

const initNotificationsMigration = "migrations/20260305193000_init_notifications.sql"

func LoadInitNotificationsUpSQL() (string, error) {
	content, err := migrationFS.ReadFile(initNotificationsMigration)
	if err != nil {
		return "", err
	}
	up := extractGooseUp(content)
	if strings.TrimSpace(up) == "" {
		return "", fmt.Errorf("empty goose up section in %s", initNotificationsMigration)
	}
	return up, nil
}

func extractGooseUp(content []byte) string {
	lines := strings.Split(string(content), "\n")
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
