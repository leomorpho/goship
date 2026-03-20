package pwa

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/core"
	templates "github.com/leomorpho/goship/app/views"
	pages "github.com/leomorpho/goship/modules/pwa/views/web/pages/gen"
)

type RouteService struct {
	controller ui.Controller
}

func NewRouteService(controller ui.Controller) *RouteService {
	return &RouteService{controller: controller}
}

func (m *Module) RegisterRoutes(r core.Router) error {
	r.GET("/install-app", m.service.GetInstallPage).Name = routenames.RouteNameInstallApp
	return nil
}

func (s *RouteService) GetInstallPage(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageInstallApp
	page.Component = pages.InstallApp(&page)
	page.HTMX.Request.Boosted = true

	return s.controller.RenderPage(ctx, page)
}
