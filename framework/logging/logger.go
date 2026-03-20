package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/leomorpho/goship/config"
)

func NewLogger(cfg config.LogConfig) *slog.Logger {
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{Level: level}

	if strings.ToLower(cfg.Format) == "json" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func parseLevel(l string) slog.Level {
	switch strings.ToLower(l) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
