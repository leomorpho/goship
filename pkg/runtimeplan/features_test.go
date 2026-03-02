package runtimeplan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveWebFeatures(t *testing.T) {
	tests := []struct {
		name        string
		plan        Plan
		hasCache    bool
		hasNotifier bool
		want        WebFeatures
	}{
		{
			name: "web disabled disables all web features",
			plan: Plan{
				RunWeb: false,
			},
			hasCache:    true,
			hasNotifier: true,
			want:        WebFeatures{},
		},
		{
			name: "page cache enabled when web runs and cache exists",
			plan: Plan{
				RunWeb: true,
				Adapters: Adapters{
					PubSub: "inproc",
				},
			},
			hasCache:    true,
			hasNotifier: false,
			want: WebFeatures{
				EnablePageCache: true,
				EnableRealtime:  false,
			},
		},
		{
			name: "realtime enabled when notifier and pubsub are present",
			plan: Plan{
				RunWeb: true,
				Adapters: Adapters{
					PubSub: "redis",
				},
			},
			hasCache:    false,
			hasNotifier: true,
			want: WebFeatures{
				EnablePageCache: false,
				EnableRealtime:  true,
			},
		},
		{
			name: "realtime disabled without pubsub adapter",
			plan: Plan{
				RunWeb: true,
				Adapters: Adapters{
					PubSub: "",
				},
			},
			hasCache:    true,
			hasNotifier: true,
			want: WebFeatures{
				EnablePageCache: true,
				EnableRealtime:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveWebFeatures(tt.plan, tt.hasCache, tt.hasNotifier)
			assert.Equal(t, tt.want, got)
		})
	}
}
