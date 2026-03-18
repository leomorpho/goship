package runtime

import "testing"

func TestResolveDevDefaultMode(t *testing.T) {
	t.Setenv("PAGODA_APP_ENVIRONMENT", "")
	t.Setenv("PAGODA_RUNTIME_PROFILE", "")
	t.Setenv("PAGODA_ADAPTERS_JOBS", "")
	t.Setenv("PAGODA_JOBS_DRIVER", "")

	mode, err := ResolveDevDefaultMode()
	if err != nil {
		t.Fatalf("ResolveDevDefaultMode() error = %v", err)
	}
	if mode != "web" {
		t.Fatalf("mode = %q, want web", mode)
	}
}

func TestResolveDevDefaultMode_RuntimeProfileWillDriveCanonicalLoop_RedSpec(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		profile     string
		jobs        string
		want        string
	}{
		{
			name:        "single-node stays web even if jobs adapter is asynq",
			environment: "local",
			profile:     "single-node",
			jobs:        "asynq",
			want:        "web",
		},
		{
			name:        "distributed defaults to full loop even if jobs adapter is backlite",
			environment: "prod",
			profile:     "distributed",
			jobs:        "backlite",
			want:        "all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PAGODA_APP_ENVIRONMENT", tt.environment)
			t.Setenv("PAGODA_RUNTIME_PROFILE", tt.profile)
			t.Setenv("PAGODA_ADAPTERS_JOBS", tt.jobs)
			t.Setenv("PAGODA_JOBS_DRIVER", "")

			mode, err := ResolveDevDefaultMode()
			if err != nil {
				t.Fatalf("ResolveDevDefaultMode() error = %v", err)
			}

			t.Skip("red spec: TKT-266 will make ship dev follow runtime profile semantics instead of inferring from jobs adapter alone")

			if mode != tt.want {
				t.Fatalf("mode = %q, want %q", mode, tt.want)
			}
		})
	}
}
