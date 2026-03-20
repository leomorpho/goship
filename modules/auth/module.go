package auth

import (
	"io/fs"

	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/leomorpho/goship/framework/core"
	dbmigrate "github.com/leomorpho/goship/modules/auth/db/migrate"
	profilesvc "github.com/leomorpho/goship/modules/profile"
)

type Deps struct {
	Controller                    ui.Controller
	ProfileService                profilesvc.ProfileService
	SubscriptionsService          *paidsubscriptions.Service
	NotificationPermissionService *notifications.NotificationPermissionService
	TwoFactorAuth                 TwoFactorAuth
}

type Module struct {
	service *Service
}

func New(deps Deps) *Module {
	return &Module{
		service: NewService(deps),
	}
}

func (m *Module) ID() string {
	return "auth"
}

func (m *Module) Migrations() fs.FS {
	return dbmigrate.Migrations()
}

func (m *Module) RegisterRoutes(r core.Router) error {
	registerRoutes(r, m.service)
	return nil
}
