package emailsubscriptions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStaticListCatalog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		specs   []ListSpec
		wantErr string
	}{
		{
			name: "ok",
			specs: []ListSpec{
				{Key: List("newsletter"), Active: true},
				{Key: List("product-updates"), Active: true},
			},
		},
		{
			name:    "empty",
			specs:   nil,
			wantErr: "list catalog cannot be empty",
		},
		{
			name: "duplicate",
			specs: []ListSpec{
				{Key: List("newsletter"), Active: true},
				{Key: List("NEWSLETTER"), Active: true},
			},
			wantErr: `duplicate list key "newsletter"`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, err := NewStaticListCatalog(tt.specs)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, c)
		})
	}
}
