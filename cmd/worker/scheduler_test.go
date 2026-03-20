package main

import (
	"context"
	"testing"
)

type fakeCronScheduler struct {
	startCalls int
	stopCalls  int
}

func (f *fakeCronScheduler) Start() {
	f.startCalls++
}

func (f *fakeCronScheduler) Stop() context.Context {
	f.stopCalls++
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestStartWorkerScheduler(t *testing.T) {
	t.Parallel()

	fake := &fakeCronScheduler{}
	stop := startWorkerScheduler(fake)
	if fake.startCalls != 1 {
		t.Fatalf("start calls = %d, want 1", fake.startCalls)
	}
	stop()
	if fake.stopCalls != 1 {
		t.Fatalf("stop calls = %d, want 1", fake.stopCalls)
	}
}

func TestStartWorkerSchedulerNilSafe(t *testing.T) {
	t.Parallel()

	stop := startWorkerScheduler(nil)
	stop()
}
