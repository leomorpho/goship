package jobs

import (
	"context"
	"strings"
	"testing"
)

func TestRedisCoreJobsStartSchedulerRequiresInitializedClient(t *testing.T) {
	t.Parallel()

	core := &redisCoreJobs{}
	err := core.StartScheduler(context.Background())
	if err == nil {
		t.Fatal("expected scheduler initialization error")
	}
	if !strings.Contains(err.Error(), "jobs scheduler is not initialized") {
		t.Fatalf("error = %v, want scheduler init diagnostic", err)
	}
}
