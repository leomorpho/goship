package jobs

import "context"

type noopJobsInspector struct{}

func newNoopJobsInspector() JobsInspector {
	return noopJobsInspector{}
}

func (noopJobsInspector) List(context.Context, JobListFilter) ([]JobRecord, error) {
	return []JobRecord{}, nil
}

func (noopJobsInspector) Get(context.Context, string) (JobRecord, bool, error) {
	return JobRecord{}, false, nil
}
