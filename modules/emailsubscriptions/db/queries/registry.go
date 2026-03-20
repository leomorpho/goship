package queries

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.sql
var queryFS embed.FS

var registry = mustLoadQueries()

func Get(name string) (string, error) {
	query, ok := registry[name]
	if !ok {
		return "", fmt.Errorf("query %q not found", name)
	}
	return query, nil
}

func mustLoadQueries() map[string]string {
	entries, err := queryFS.ReadDir(".")
	if err != nil {
		panic(fmt.Sprintf("read queries dir: %v", err))
	}

	out := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		b, readErr := queryFS.ReadFile(entry.Name())
		if readErr != nil {
			panic(fmt.Sprintf("read query file %s: %v", entry.Name(), readErr))
		}
		fileQueries := parseNamedQueries(string(b))
		for k, v := range fileQueries {
			if _, exists := out[k]; exists {
				panic(fmt.Sprintf("duplicate query name %q", k))
			}
			out[k] = v
		}
	}
	return out
}

func parseNamedQueries(content string) map[string]string {
	lines := strings.Split(content, "\n")
	out := make(map[string]string)
	var currentName string
	var current []string

	flush := func() {
		if currentName == "" {
			return
		}
		query := strings.TrimSpace(strings.Join(current, "\n"))
		if query != "" {
			out[currentName] = query
		}
		currentName = ""
		current = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- name:") {
			flush()
			currentName = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- name:"))
			continue
		}
		if currentName != "" {
			current = append(current, line)
		}
	}
	flush()
	return out
}
