package auditlog

import (
	"context"
	"io/fs"
	"strconv"

	"github.com/leomorpho/goship/framework/events"
	eventtypes "github.com/leomorpho/goship/framework/events/types"
	dbmigrate "github.com/leomorpho/goship/modules/auditlog/db/migrate"
)

type Module struct {
	service *Service
}

func NewModule(service *Service) *Module {
	return &Module{service: service}
}

func (m *Module) ID() string {
	return "auditlog"
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

func Subscribe(bus *events.Bus, service *Service) {
	if bus == nil || service == nil {
		return
	}

	events.Subscribe(bus, func(ctx context.Context, event eventtypes.UserLoggedIn) error {
		userID := event.UserID
		return service.Record(WithRequestMetadata(ctx, &userID, event.IP, ""), "user.login", "user", strconv.FormatInt(event.UserID, 10), nil)
	})
	events.Subscribe(bus, func(ctx context.Context, event eventtypes.UserLoggedOut) error {
		userID := event.UserID
		return service.Record(WithRequestMetadata(ctx, &userID, "", ""), "user.logout", "user", strconv.FormatInt(event.UserID, 10), nil)
	})
	events.Subscribe(bus, func(ctx context.Context, event eventtypes.PasswordChanged) error {
		userID := event.UserID
		return service.Record(WithRequestMetadata(ctx, &userID, "", ""), "user.password_changed", "user", strconv.FormatInt(event.UserID, 10), nil)
	})
}
