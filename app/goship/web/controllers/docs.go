package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/app/goship/webui"
)

type docsRoute struct {
	ctr webui.Controller
}

func NewDocsRoute(ctr webui.Controller) *docsRoute {
	return &docsRoute{
		ctr: ctr,
	}
}

func (w *docsRoute) GetDocsHome(ctx echo.Context) error {
	page := webui.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Introduction"
	page.Component = pages.DocumentationLandingPage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}

func (w *docsRoute) GetDocsGettingStarted(ctx echo.Context) error {
	page := webui.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Architecture"
	page.Component = pages.DocumentationArchitecturePage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}

func (w *docsRoute) GetDocsGuidedTour(ctx echo.Context) error {
	page := webui.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Architecture"
	page.Component = pages.DocumentationArchitecturePage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}

func (w *docsRoute) GetDocsArchitecture(ctx echo.Context) error {
	page := webui.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Architecture"
	page.Component = pages.DocumentationArchitecturePage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}
