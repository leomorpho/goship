package flags

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

type fakeSyncStore struct {
	flags       map[string]Flag
	createCalls int
	updateCalls int
}

func (f *fakeSyncStore) Find(context.Context, string) (Flag, error) { return Flag{}, nil }
func (f *fakeSyncStore) List(context.Context) ([]Flag, error) {
	out := make([]Flag, 0, len(f.flags))
	for _, flag := range f.flags {
		out = append(out, flag)
	}
	return out, nil
}
func (f *fakeSyncStore) Create(_ context.Context, flag Flag) error {
	if f.flags == nil {
		f.flags = map[string]Flag{}
	}
	f.flags[flag.Key] = flag
	f.createCalls++
	return nil
}
func (f *fakeSyncStore) Update(context.Context, Flag) error { return nil }
func (f *fakeSyncStore) UpsertDescription(_ context.Context, key string, description string) error {
	flag := f.flags[key]
	flag.Description = description
	f.flags[key] = flag
	f.updateCalls++
	return nil
}
func (f *fakeSyncStore) Delete(context.Context, string) error { return nil }

func TestSync_CreatesNewFlags(t *testing.T) {
	resetRegistryForTest()
	Register(FlagDefinition{Key: "alpha", Description: "alpha flag", Default: true})
	Register(FlagDefinition{Key: "beta", Description: "beta flag", Default: false})

	store := &fakeSyncStore{flags: map[string]Flag{}}
	syncer := NewSyncer(store, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	summary, err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if summary.Created != 2 || summary.Updated != 0 || summary.Unchanged != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	if !store.flags["alpha"].Enabled || store.flags["beta"].Enabled {
		t.Fatalf("created flags = %+v", store.flags)
	}
}

func TestSync_UpdatesDescriptionWithoutTouchingEnabled(t *testing.T) {
	resetRegistryForTest()
	Register(FlagDefinition{Key: "alpha", Description: "new description", Default: false})

	store := &fakeSyncStore{flags: map[string]Flag{
		"alpha": {Key: "alpha", Enabled: true, Description: "old description"},
	}}
	syncer := NewSyncer(store, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	summary, err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if summary.Created != 0 || summary.Updated != 1 || summary.Unchanged != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	if !store.flags["alpha"].Enabled {
		t.Fatalf("enabled was mutated, flags = %+v", store.flags)
	}
	if store.flags["alpha"].Description != "new description" {
		t.Fatalf("description not updated, flags = %+v", store.flags)
	}
}

func TestSync_Idempotent(t *testing.T) {
	resetRegistryForTest()
	Register(FlagDefinition{Key: "alpha", Description: "alpha flag", Default: true})

	store := &fakeSyncStore{flags: map[string]Flag{}}
	syncer := NewSyncer(store, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	first, err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("first Sync() error = %v", err)
	}
	second, err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}

	if first.Created != 1 || first.Updated != 0 || first.Unchanged != 0 {
		t.Fatalf("first summary = %+v", first)
	}
	if second.Created != 0 || second.Updated != 0 || second.Unchanged != 1 {
		t.Fatalf("second summary = %+v", second)
	}
}

func TestSync_LogsSummary(t *testing.T) {
	resetRegistryForTest()
	Register(FlagDefinition{Key: "alpha", Description: "alpha flag", Default: true})

	store := &fakeSyncStore{flags: map[string]Flag{}}
	var out bytes.Buffer
	syncer := NewSyncer(store, slog.New(slog.NewTextHandler(&out, nil)))

	if _, err := syncer.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	logLine := out.String()
	for _, token := range []string{"flags sync complete", "created=1", "updated=0", "unchanged=0"} {
		if !bytes.Contains([]byte(logLine), []byte(token)) {
			t.Fatalf("log output %q missing %q", logLine, token)
		}
	}
}

