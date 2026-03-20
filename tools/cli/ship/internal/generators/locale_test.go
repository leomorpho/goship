package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestParseMakeLocaleArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    MakeLocaleOptions
		wantErr string
	}{
		{
			name: "valid",
			args: []string{"fr"},
			want: MakeLocaleOptions{Code: "fr"},
		},
		{
			name: "normalize underscore",
			args: []string{"fr_CA"},
			want: MakeLocaleOptions{Code: "fr-ca"},
		},
		{
			name:    "missing",
			args:    []string{},
			wantErr: "usage: ship make:locale",
		},
		{
			name:    "unknown option",
			args:    []string{"fr", "--x"},
			wantErr: "unknown option",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseMakeLocaleArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRunMakeLocale_GeneratesLocaleFile(t *testing.T) {
	root := t.TempDir()
	localesDir := filepath.Join(root, "locales")
	if err := os.MkdirAll(localesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localesDir, "en.toml"), []byte(`
"auth.login.title" = "Sign in to your account"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunMakeLocale([]string{"fr"}, LocaleDeps{
		Out: out,
		Err: errOut,
		Cwd: root,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}

	target := filepath.Join(localesDir, "fr.toml")
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read fr locale: %v", err)
	}
	var parsed map[string]any
	if _, err := toml.Decode(string(content), &parsed); err != nil {
		t.Fatalf("parse fr locale: %v", err)
	}

	if got, _ := parsed["auth.login.title"].(string); got != "" {
		t.Fatalf("fr auth.login.title = %q, want empty", got)
	}
}
