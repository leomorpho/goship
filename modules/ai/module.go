package ai

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	dbmigrate "github.com/leomorpho/goship/modules/ai/db/migrate"
)

const ModuleID = "ai"

type Module struct {
	service       *Service
	conversations *ConversationService
}

func NewModule(service *Service, conversations *ConversationService) *Module {
	if service == nil {
		service = NewService(NewUnavailableProvider("missing AI service"), slog.Default())
	}

	return &Module{
		service:       service,
		conversations: conversations,
	}
}

func (m *Module) ID() string {
	return ModuleID
}

func (m *Module) Migrations() fs.FS {
	return dbmigrate.Migrations()
}

func (m *Module) Service() *Service {
	if m == nil {
		return nil
	}
	return m.service
}

func (m *Module) Conversations() *ConversationService {
	if m == nil {
		return nil
	}
	return m.conversations
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
