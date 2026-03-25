package commands

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestUIProviderScaffoldGolden(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		wantContains []string
		wantExcludes []string
	}{
		{
			name:     "franken",
			provider: newUIProviderFranken,
			wantContains: []string{
				"https://cdn.jsdelivr.net/npm/uikit",
				"https://cdn.jsdelivr.net/npm/franken-ui",
			},
			wantExcludes: []string{
				"https://cdn.jsdelivr.net/npm/flowbite",
			},
		},
		{
			name:     "daisy",
			provider: newUIProviderDaisy,
			wantContains: []string{
				"https://cdn.jsdelivr.net/npm/flowbite",
			},
			wantExcludes: []string{
				"https://cdn.jsdelivr.net/npm/uikit",
				"https://cdn.jsdelivr.net/npm/franken-ui",
			},
		},
		{
			name:     "bare",
			provider: newUIProviderBare,
			wantExcludes: []string{
				"https://cdn.jsdelivr.net/npm/flowbite",
				"https://cdn.jsdelivr.net/npm/uikit",
				"https://cdn.jsdelivr.net/npm/franken-ui",
			},
		},
	}

	// Guardrail: scaffolded templ files should stay structural and provider-neutral.
	classLeakRe := regexp.MustCompile(`class="[^"]*(starter-|uk-|btn\b|daisyui-)`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			opts := NewProjectOptions{
				Name:       "demo",
				Module:     "example.com/demo",
				AppPath:    filepath.Join(root, "demo"),
				UIProvider: tt.provider,
			}

			if err := ScaffoldNewProject(opts, NewDeps{
				ParseAgentPolicyBytes:      func(b []byte) (policies.AgentPolicy, error) { return policies.ParsePolicyBytes(b) },
				RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
				AgentPolicyFilePath:        policies.AgentPolicyFilePath,
			}); err != nil {
				t.Fatalf("ScaffoldNewProject failed: %v", err)
			}

			layoutPath := filepath.Join(opts.AppPath, "app", "views", "web", "layouts", "base.templ")
			layoutBytes, err := os.ReadFile(layoutPath)
			if err != nil {
				t.Fatalf("read base layout: %v", err)
			}
			layout := string(layoutBytes)
			for _, token := range tt.wantContains {
				if !strings.Contains(layout, token) {
					t.Fatalf("layout missing %q for provider %q:\n%s", token, tt.provider, layout)
				}
			}
			for _, token := range tt.wantExcludes {
				if strings.Contains(layout, token) {
					t.Fatalf("layout should not include %q for provider %q:\n%s", token, tt.provider, layout)
				}
			}

			dotEnvBytes, err := os.ReadFile(filepath.Join(opts.AppPath, ".env"))
			if err != nil {
				t.Fatalf("read .env: %v", err)
			}
			if !strings.Contains(string(dotEnvBytes), "UI_PROVIDER="+tt.provider) {
				t.Fatalf(".env missing provider %q:\n%s", tt.provider, string(dotEnvBytes))
			}

			templRoot := filepath.Join(opts.AppPath, "app", "views", "web")
			err = filepath.WalkDir(templRoot, func(path string, d os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if d.IsDir() || filepath.Ext(path) != ".templ" {
					return nil
				}
				contentBytes, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				content := string(contentBytes)
				if classLeakRe.MatchString(content) {
					return &uiClassLeakError{path: path, content: content}
				}
				return nil
			})
			if err != nil {
				if leakErr, ok := err.(*uiClassLeakError); ok {
					t.Fatalf("scaffolded templ file leaked UI-library classes at %s:\n%s", leakErr.path, leakErr.content)
				}
				t.Fatalf("walk templ files: %v", err)
			}
		})
	}
}

type uiClassLeakError struct {
	path    string
	content string
}

func (e *uiClassLeakError) Error() string {
	return "ui class leak in templ file"
}
