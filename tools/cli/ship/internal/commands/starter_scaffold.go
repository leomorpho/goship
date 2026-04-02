package commands

import (
	"os"
	"path/filepath"
	"strings"
)

func looksLikeStarterScaffoldApp(root string) bool {
	routerPath := filepath.Join(root, "app", "router.go")
	templatesPath := filepath.Join(root, "app", "views", "templates.go")
	mainPath := filepath.Join(root, "cmd", "web", "main.go")
	if _, err := os.Stat(routerPath); err != nil {
		return false
	}
	if _, err := os.Stat(templatesPath); err != nil {
		return false
	}
	if _, err := os.Stat(mainPath); err != nil {
		return false
	}

	routerBody, err := os.ReadFile(routerPath)
	if err != nil {
		return false
	}
	mainBody, err := os.ReadFile(mainPath)
	if err != nil {
		return false
	}

	return strings.Contains(string(routerBody), "type Route struct") &&
		strings.Contains(string(routerBody), "templates.Page") &&
		strings.Contains(string(mainBody), "func componentForPage(page templates.Page)")
}
