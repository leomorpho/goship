package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leomorpho/goship/framework/core"
)

type fakeCache struct {
	data map[string][]byte
	err  error
}

func (f *fakeCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	v, ok := f.data[key]
	return v, ok, nil
}

func (f *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	if f.err != nil {
		return f.err
	}
	if f.data == nil {
		f.data = map[string][]byte{}
	}
	f.data[key] = value
	return nil
}

func (f *fakeCache) Delete(_ context.Context, key string) error {
	if f.err != nil {
		return f.err
	}
	delete(f.data, key)
	return nil
}

func (*fakeCache) InvalidatePrefix(context.Context, string) error { return nil }
func (*fakeCache) Close() error                                   { return nil }

type fakeJobsInspector struct {
	rows []core.JobRecord
	err  error
}

func (f fakeJobsInspector) List(context.Context, core.JobListFilter) ([]core.JobRecord, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func (f fakeJobsInspector) Get(context.Context, string) (core.JobRecord, bool, error) {
	return core.JobRecord{}, false, nil
}

func TestCacheChecker(t *testing.T) {
	checker := NewCacheChecker(&fakeCache{}, time.Second)
	result := checker.Check(context.Background())
	if result.Status != StatusOK {
		t.Fatalf("expected ok status, got %q", result.Status)
	}

	checker = NewCacheChecker(&fakeCache{err: errors.New("cache down")}, time.Second)
	result = checker.Check(context.Background())
	if result.Status != StatusError {
		t.Fatalf("expected error status, got %q", result.Status)
	}
}

func TestDBCheckerNilDB(t *testing.T) {
	checker := NewDBChecker(nil, time.Second)
	result := checker.Check(context.Background())
	if result.Status != StatusError {
		t.Fatalf("expected error status, got %q", result.Status)
	}
}

func TestJobsChecker(t *testing.T) {
	checker := NewJobsChecker(fakeJobsInspector{
		rows: []core.JobRecord{{ID: "1"}, {ID: "2"}},
	}, time.Second)
	result := checker.Check(context.Background())
	if result.Status != StatusOK {
		t.Fatalf("expected ok status, got %q", result.Status)
	}
	if result.QueueDepth != 2 {
		t.Fatalf("expected queue depth 2, got %d", result.QueueDepth)
	}

	checker = NewJobsChecker(fakeJobsInspector{err: context.DeadlineExceeded}, time.Second)
	result = checker.Check(context.Background())
	if result.Status != StatusError {
		t.Fatalf("expected error status on timeout, got %q", result.Status)
	}

	checker = NewJobsChecker(fakeJobsInspector{err: errors.New("not implemented")}, time.Second)
	result = checker.Check(context.Background())
	if result.Status != StatusOK {
		t.Fatalf("expected ok status for non-fatal inspector error, got %q", result.Status)
	}
}
