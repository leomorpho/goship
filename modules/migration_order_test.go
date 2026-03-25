package modules_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var moduleMigrationNamePattern = regexp.MustCompile(`^([0-9]{14})_.+\.sql$`)

func TestFirstPartyModuleMigrations_HaveDeterministicGlobalOrder(t *testing.T) {
	t.Parallel()

	modulesDir := testModulesRoot(t)
	type migrationRef struct {
		module    string
		file      string
		timestamp string
	}
	migrations := make([]migrationRef, 0)
	timestampOwners := map[string][]string{}

	err := filepath.WalkDir(modulesDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".sql" {
			return nil
		}
		rel, err := filepath.Rel(modulesDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !strings.Contains(rel, "/db/migrate/migrations/") {
			return nil
		}

		base := filepath.Base(path)
		match := moduleMigrationNamePattern.FindStringSubmatch(base)
		if len(match) != 2 {
			t.Fatalf("module migration file %q must match %s", rel, moduleMigrationNamePattern.String())
		}

		moduleName := strings.SplitN(rel, "/", 2)[0]
		timestamp := match[1]
		migrations = append(migrations, migrationRef{
			module:    moduleName,
			file:      rel,
			timestamp: timestamp,
		})
		timestampOwners[timestamp] = append(timestampOwners[timestamp], rel)
		return nil
	})
	if err != nil {
		t.Fatalf("scan module migrations: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected at least one module migration file")
	}

	var timestampCollisions []string
	for timestamp, owners := range timestampOwners {
		if len(owners) < 2 {
			continue
		}
		sort.Strings(owners)
		timestampCollisions = append(timestampCollisions, timestamp+" -> "+strings.Join(owners, ", "))
	}
	if len(timestampCollisions) > 0 {
		sort.Strings(timestampCollisions)
		t.Fatalf("module migration timestamp collisions detected:\n%s", strings.Join(timestampCollisions, "\n"))
	}

	sort.Slice(migrations, func(i, j int) bool {
		if migrations[i].timestamp == migrations[j].timestamp {
			return migrations[i].file < migrations[j].file
		}
		return migrations[i].timestamp < migrations[j].timestamp
	})

	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].timestamp > migrations[i].timestamp {
			t.Fatalf(
				"module migration order is not monotonic: %s (%s) before %s (%s)",
				migrations[i-1].file,
				migrations[i-1].timestamp,
				migrations[i].file,
				migrations[i].timestamp,
			)
		}
	}
}

func testModulesRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}
	return filepath.Dir(currentFile)
}
