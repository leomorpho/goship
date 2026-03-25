package modules_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var routeRegistrationPattern = regexp.MustCompile(`\.(GET|POST|PUT|DELETE)\("([^"]+)"`)

func TestFirstPartyModuleRouteSurface_HasNoMethodPathCollisions(t *testing.T) {
	t.Parallel()

	modulesDir := testModulesDir(t)
	type routeLocation struct {
		file string
		line int
	}

	seen := map[string][]routeLocation{}
	err := filepath.WalkDir(modulesDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "routes.go" {
			return nil
		}

		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(body)
		rel, err := filepath.Rel(modulesDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		for _, match := range routeRegistrationPattern.FindAllStringSubmatchIndex(text, -1) {
			method := text[match[2]:match[3]]
			routePath := text[match[4]:match[5]]
			key := method + " " + routePath
			seen[key] = append(seen[key], routeLocation{
				file: rel,
				line: 1 + strings.Count(text[:match[0]], "\n"),
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan module routes: %v", err)
	}

	var collisions []string
	for key, locations := range seen {
		if len(locations) < 2 {
			continue
		}
		parts := make([]string, 0, len(locations))
		for _, location := range locations {
			parts = append(parts, location.file+":"+strconv.Itoa(location.line))
		}
		collisions = append(collisions, key+" -> "+strings.Join(parts, ", "))
	}
	if len(collisions) == 0 {
		return
	}
	sort.Strings(collisions)
	t.Fatalf("route collisions detected across first-party modules:\n%s", strings.Join(collisions, "\n"))
}

func testModulesDir(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}
	return filepath.Dir(currentFile)
}
