package i18ncheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectCompletenessIssues_DeterministicAndComplete(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "locales"), 0o755); err != nil {
		t.Fatalf("mkdir locales: %v", err)
	}
if err := os.WriteFile(filepath.Join(root, "locales", "en.toml"), []byte(`
"cart.items.one" = "one item"
`), 0o644); err != nil {
		t.Fatalf("write locale: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "app", "web", "controllers"), 0o755); err != nil {
		t.Fatalf("mkdir controllers: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "web", "controllers", "sample.go"), []byte(`
package controllers
func demo() {
	_ = container.I18n.TC(ctx, "cart.items", 2)
	_ = container.I18n.TS(ctx, "profile.role", "admin")
}
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	first, err := CollectCompletenessIssues(root)
	if err != nil {
		t.Fatalf("CollectCompletenessIssues first: %v", err)
	}
	second, err := CollectCompletenessIssues(root)
	if err != nil {
		t.Fatalf("CollectCompletenessIssues second: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("issues length mismatch: %d vs %d", len(first), len(second))
	}
	if len(first) != 3 {
		t.Fatalf("issues length = %d, want 3", len(first))
	}

	expectedKinds := map[string]bool{
		"plural_missing_other":  false,
		"select_missing_other":  false,
		"select_missing_variant": false,
	}
	for idx, issue := range first {
		if issue.ID == "" || len(issue.ID) != len("I18N-C-XXXXXXXXXX") || issue.ID[:7] != "I18N-C-" {
			t.Fatalf("issue[%d] has unexpected id %q", idx, issue.ID)
		}
		if _, ok := expectedKinds[issue.Kind]; ok {
			expectedKinds[issue.Kind] = true
		}
		if issue != second[idx] {
			t.Fatalf("issue[%d] differs between runs:\nfirst=%+v\nsecond=%+v", idx, issue, second[idx])
		}
	}
	for kind, seen := range expectedKinds {
		if !seen {
			t.Fatalf("missing expected kind %q in %+v", kind, first)
		}
	}
}
