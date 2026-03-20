package generators

import (
	"strings"
	"testing"
)

func TestParseMakeScheduleArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    MakeScheduleOptions
		wantErr string
	}{
		{
			name: "valid",
			args: []string{"DailyReport", "--cron", "0 9 * * *"},
			want: MakeScheduleOptions{Name: "DailyReport", Cron: "0 9 * * *"},
		},
		{
			name:    "missing name",
			args:    []string{"--cron", "0 9 * * *"},
			wantErr: "usage: ship make:schedule",
		},
		{
			name:    "missing cron",
			args:    []string{"DailyReport"},
			wantErr: "usage: ship make:schedule",
		},
		{
			name:    "unknown option",
			args:    []string{"DailyReport", "--when", "0 9 * * *"},
			wantErr: "unknown option",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseMakeScheduleArgs(tt.args)
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
