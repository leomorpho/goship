package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdminModuleTemplatesUseSharedStyleRecipes(t *testing.T) {
	root := repoRootForUIModuleContract(t)
	files := []string{
		filepath.Join(root, "modules", "admin", "views", "web", "components", "admin_layout.templ"),
		filepath.Join(root, "modules", "admin", "views", "web", "components", "admin_list.templ"),
		filepath.Join(root, "modules", "admin", "views", "web", "components", "admin_form.templ"),
		filepath.Join(root, "modules", "admin", "views", "web", "components", "admin_field_input.templ"),
		filepath.Join(root, "modules", "admin", "views", "web", "components", "admin_delete_confirm.templ"),
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(content)
		if !strings.Contains(text, "gs-") {
			t.Fatalf("expected shared gs recipe classes in %s", file)
		}
		for _, forbidden := range []string{"slate-", "text-red-", "border-red-"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("found forbidden ad hoc style token %q in %s", forbidden, file)
			}
		}
	}
}

func repoRootForUIModuleContract(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "modules")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}
