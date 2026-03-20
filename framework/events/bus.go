package events

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type HandlerFunc func(ctx context.Context, event any) error

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]HandlerFunc
}

func NewBus() *Bus {
	return &Bus{
		handlers: map[string][]HandlerFunc{},
	}
}

func (b *Bus) Publish(ctx context.Context, event any) error {
	if b == nil {
		return fmt.Errorf("event bus is nil")
	}

	typeName, err := eventTypeName(event)
	if err != nil {
		return err
	}

	b.mu.RLock()
	handlers := append([]HandlerFunc(nil), b.handlers[typeName]...)
	b.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func Subscribe[T any](b *Bus, handler func(ctx context.Context, event T) error) {
	if b == nil || handler == nil {
		return
	}

	var zero T
	typeName, err := eventTypeName(zero)
	if err != nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[typeName] = append(b.handlers[typeName], func(ctx context.Context, event any) error {
		typed, ok := event.(T)
		if !ok {
			return fmt.Errorf("event type mismatch: expected %s, got %T", typeName, event)
		}
		return handler(ctx, typed)
	})
}

func eventTypeName(event any) (string, error) {
	typ := reflect.TypeOf(event)
	if typ == nil {
		return "", fmt.Errorf("event cannot be nil")
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ.String(), nil
}
