package flags

import "context"

type Store interface {
	Find(ctx context.Context, key string) (Flag, error)
	List(ctx context.Context) ([]Flag, error)
	Create(ctx context.Context, flag Flag) error
	Update(ctx context.Context, flag Flag) error
	UpsertDescription(ctx context.Context, key string, description string) error
	Delete(ctx context.Context, key string) error
}
