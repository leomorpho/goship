package flags

import "context"

type Store interface {
	Find(ctx context.Context, key string) (Flag, error)
	Create(ctx context.Context, flag Flag) error
	Update(ctx context.Context, flag Flag) error
	Delete(ctx context.Context, key string) error
}
