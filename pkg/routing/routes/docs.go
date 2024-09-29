package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
)

type docsRoute struct {
	ctr controller.Controller
}

func NewDocsRoute(ctr controller.Controller) *docsRoute {
	return &docsRoute{
		ctr: ctr,
	}
}

func (w *docsRoute) GetDocsHome(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Introduction"
	page.Component = pages.DocumentationLandingPage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}

func (w *docsRoute) GetDocsGettingStarted(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Architecture"
	page.Component = pages.DocumentationArchitecturePage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}

func (w *docsRoute) GetDocsGuidedTour(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Architecture"
	page.Component = pages.DocumentationArchitecturePage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}

func (w *docsRoute) GetDocsArchitecture(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Documentation
	page.Name = templates.PageWiki
	page.Title = "Architecture"
	page.Component = pages.DocumentationArchitecturePage(&page)
	page.HTMX.Request.Boosted = true

	return w.ctr.RenderPage(ctx, page)
}
