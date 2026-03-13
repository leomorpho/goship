package ai

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
)

type Module struct {
	service *Service
}

func NewModule(service *Service) *Module {
	if service == nil {
		service = NewService(NewUnavailableProvider("missing AI service"), slog.Default())
	}

	return &Module{
		service: service,
	}
}

func (m *Module) ID() string {
	return "ai"
}

func (m *Module) Migrations() fs.FS {
	return nil
}

func (m *Module) Service() *Service {
	if m == nil {
		return nil
	}
	return m.service
}

func NewUnavailableProvider(reason string) Provider {
	return unavailableProvider{reason: reason}
}

type unavailableProvider struct {
	reason string
}

func (p unavailableProvider) Complete(_ context.Context, _ Request) (*Response, error) {
	return nil, fmt.Errorf("ai provider unavailable: %s", p.reason)
}

func (p unavailableProvider) Stream(_ context.Context, _ Request) (<-chan Token, error) {
	return nil, fmt.Errorf("ai provider unavailable: %s", p.reason)
}
