package policies

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTemplFunctionsMissingRenders_Matrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		content     string
		wantMissing []string
	}{
		{
			name: "renders then routes",
			content: `// Renders: profile card with avatar, display name, and bio
// Route(s): /profile
templ ProfileCard() {}
`,
			wantMissing: []string{},
		},
		{
			name: "routes then renders",
			content: `// Route(s): /profile
// Renders: profile card with avatar, display name, and bio
templ ProfileCard() {}
`,
			wantMissing: []string{},
		},
		{
			name: "renders then blank line then routes",
			content: `// Renders: profile card with avatar, display name, and bio

// Route(s): /profile
templ ProfileCard() {}
`,
			wantMissing: []string{},
		},
		{
			name: "missing renders remains a violation",
			content: `// Route(s): /profile
templ MissingRenders() {}
`,
			wantMissing: []string{"MissingRenders"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "component.templ")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("write fixture: %v", err)
			}

			gotMissing := templFunctionsMissingRenders(path)
			if !reflect.DeepEqual(gotMissing, tc.wantMissing) {
				t.Fatalf("templFunctionsMissingRenders() = %v, want %v", gotMissing, tc.wantMissing)
			}
		})
	}
}
