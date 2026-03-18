package generators

import (
	"strings"
	"testing"
)

func TestParseMakeMailerArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    MakeMailerOptions
		wantErr string
	}{
		{name: "valid", args: []string{"WelcomeDigest"}, want: MakeMailerOptions{Name: "WelcomeDigest"}},
		{name: "missing", args: []string{}, wantErr: "usage: ship make:mailer"},
		{name: "unknown option", args: []string{"WelcomeDigest", "--x"}, wantErr: "unknown option"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseMakeMailerArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want contains %q", err, tt.wantErr)
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
