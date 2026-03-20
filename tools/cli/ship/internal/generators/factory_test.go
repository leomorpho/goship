package generators

import (
	"strings"
	"testing"
)

func TestParseMakeFactoryArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    MakeFactoryOptions
		wantErr string
	}{
		{
			name: "valid",
			args: []string{"User"},
			want: MakeFactoryOptions{Name: "User"},
		},
		{
			name:    "missing",
			args:    []string{},
			wantErr: "usage: ship make:factory",
		},
		{
			name:    "unknown option",
			args:    []string{"User", "--x"},
			wantErr: "unknown option",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseMakeFactoryArgs(tt.args)
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
