package flags

import (
	"context"
	"log/slog"
)

type SyncSummary struct {
	Created   int
	Updated   int
	Unchanged int
}

type Syncer struct {
	store Store
	log   *slog.Logger
}

func NewSyncer(store Store, logger *slog.Logger) *Syncer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Syncer{store: store, log: logger}
}

func (s *Syncer) Sync(ctx context.Context) (SyncSummary, error) {
	summary := SyncSummary{}
	if s == nil || s.store == nil {
		return summary, nil
	}

	existing, err := s.store.List(ctx)
	if err != nil {
		return summary, err
	}
	byKey := make(map[FlagKey]Flag, len(existing))
	for _, flag := range existing {
		byKey[FlagKey(flag.Key)] = flag
	}

	for _, def := range All() {
		flag, exists := byKey[def.Key]
		if !exists {
			if err := s.store.Create(ctx, Flag{
				Key:         string(def.Key),
				Enabled:     def.Default,
				Description: def.Description,
			}); err != nil {
				return summary, err
			}
			summary.Created++
			continue
		}
		if flag.Description != def.Description {
			if err := s.store.UpsertDescription(ctx, string(def.Key), def.Description); err != nil {
				return summary, err
			}
			summary.Updated++
			continue
		}
		summary.Unchanged++
	}

	s.log.Info("flags sync complete", "created", summary.Created, "updated", summary.Updated, "unchanged", summary.Unchanged)
	return summary, nil
}

