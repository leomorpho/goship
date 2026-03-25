package commands

import (
	"strings"
	"testing"
)

func TestRenderBaseLayoutTempl(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		wantContains []string
		wantExcludes []string
	}{
		{
			name:     "franken includes uikit and franken css",
			provider: newUIProviderFranken,
			wantContains: []string{
				`https://cdn.jsdelivr.net/npm/uikit`,
				`https://cdn.jsdelivr.net/npm/franken-ui`,
			},
			wantExcludes: []string{
				`https://cdn.jsdelivr.net/npm/flowbite`,
			},
		},
		{
			name:     "daisy includes flowbite only",
			provider: newUIProviderDaisy,
			wantContains: []string{
				`https://cdn.jsdelivr.net/npm/flowbite`,
			},
			wantExcludes: []string{
				`https://cdn.jsdelivr.net/npm/uikit`,
				`https://cdn.jsdelivr.net/npm/franken-ui`,
			},
		},
		{
			name:     "bare has no external asset tags",
			provider: newUIProviderBare,
			wantExcludes: []string{
				`https://cdn.jsdelivr.net/npm/uikit`,
				`https://cdn.jsdelivr.net/npm/franken-ui`,
				`https://cdn.jsdelivr.net/npm/flowbite`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := renderBaseLayoutTempl(tt.provider)
			for _, token := range tt.wantContains {
				if !strings.Contains(layout, token) {
					t.Fatalf("layout for %s missing %q:\n%s", tt.provider, token, layout)
				}
			}
			for _, token := range tt.wantExcludes {
				if strings.Contains(layout, token) {
					t.Fatalf("layout for %s should not include %q:\n%s", tt.provider, token, layout)
				}
			}
		})
	}
}
