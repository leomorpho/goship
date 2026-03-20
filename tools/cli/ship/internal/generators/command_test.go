package generators

import (
	"strings"
	"testing"
)

func TestParseMakeCommandArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    MakeCommandOptions
		wantErr string
	}{
		{
			name: "valid",
			args: []string{"BackfillUserStats"},
			want: MakeCommandOptions{Name: "BackfillUserStats"},
		},
		{
			name:    "missing",
			args:    []string{},
			wantErr: "usage: ship make:command",
		},
		{
			name:    "unknown option",
			args:    []string{"BackfillUserStats", "--x"},
			wantErr: "unknown option",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseMakeCommandArgs(tt.args)
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
