package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/apps/goship/views"
	"github.com/leomorpho/goship/apps/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/apps/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/apps/goship/web/ui"
)

type docsRoute struct {
	ctr ui.Controller
}

func NewDocsRoute(ctr ui.Controller) *docsRoute {
	return &docsRoute{
		ctr: ctr,
	}
}

func (w *docsRoute) GetDocsHome(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Introduction"
	page.Component = pages.DocumentationLandingPage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}

func (w *docsRoute) GetDocsGettingStarted(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Architecture"
	page.Component = pages.DocumentationArchitecturePage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}

func (w *docsRoute) GetDocsGuidedTour(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Architecture"
	page.Component = pages.DocumentationArchitecturePage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}

func (w *docsRoute) GetDocsArchitecture(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Architecture"
	page.Component = pages.DocumentationArchitecturePage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}
