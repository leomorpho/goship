package paidsubscriptions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStaticPlanCatalog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		plans   []Plan
		free    string
		trial   string
		wantErr string
	}{
		{
			name:  "ok",
			plans: []Plan{{Key: "free", Paid: false}, {Key: "team", Paid: true}},
			free:  "free",
			trial: "team",
		},
		{
			name:    "empty",
			plans:   nil,
			free:    "free",
			wantErr: "plans cannot be empty",
		},
		{
			name:    "missing free",
			plans:   []Plan{{Key: "team", Paid: true}},
			free:    "free",
			wantErr: `free plan key "free" not found in catalog`,
		},
		{
			name:    "duplicate",
			plans:   []Plan{{Key: "team", Paid: true}, {Key: "TEAM", Paid: false}},
			free:    "team",
			wantErr: `duplicate plan key "team"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewStaticPlanCatalog(tt.plans, tt.free, tt.trial)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}
