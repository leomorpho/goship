package runtimeadapters

import (
	"context"
	"errors"

	"github.com/leomorpho/goship/framework/core"
)

var _ core.JobsInspector = (*CoreJobsInspectorAdapter)(nil)

type CoreJobsInspectorAdapter struct {
	delegate core.JobsInspector
}

func NewCoreJobsInspectorAdapter(delegate core.JobsInspector) *CoreJobsInspectorAdapter {
	return &CoreJobsInspectorAdapter{delegate: delegate}
}

func (a *CoreJobsInspectorAdapter) List(ctx context.Context, filter core.JobListFilter) ([]core.JobRecord, error) {
	if a == nil || a.delegate == nil {
		return nil, errors.New("jobs inspector is not initialized")
	}
	return a.delegate.List(ctx, filter)
}

func (a *CoreJobsInspectorAdapter) Get(ctx context.Context, id string) (core.JobRecord, bool, error) {
	if a == nil || a.delegate == nil {
		return core.JobRecord{}, false, errors.New("jobs inspector is not initialized")
	}
	return a.delegate.Get(ctx, id)
}
