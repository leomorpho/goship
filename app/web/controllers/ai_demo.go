package controllers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	templates "github.com/leomorpho/goship/app/views"
	layouts "github.com/leomorpho/goship/app/views/web/layouts/gen"
	pages "github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/modules/ai"
)

type aiDemo struct {
	ctr ui.Controller
}

func NewAIDemoRoute(ctr ui.Controller) *aiDemo {
	return &aiDemo{ctr: ctr}
}

func (r *aiDemo) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.AIDemo(&page)
	page.Name = templates.PageAIDemo
	page.Title = "AI Streaming Demo"
	page.ShowBottomNavbar = page.IsFullyOnboarded

	prompt := strings.TrimSpace(ctx.QueryParam("prompt"))
	data := viewmodels.NewAIDemoPageData()
	data.Prompt = prompt
	data.StreamURL = ctx.Echo().Reverse(routenames.RouteNameAIDemoStream) + "?prompt=" + url.QueryEscape(prompt)
	page.Data = data

	return r.ctr.RenderPage(ctx, page)
}

func (r *aiDemo) Stream(ctx echo.Context) error {
	prompt := strings.TrimSpace(ctx.QueryParam("prompt"))
	if prompt == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "prompt is required")
	}

	err := ai.StreamCompletion(ctx.Request().Context(), ctx.Response().Writer, ai.Request{
		Messages: []ai.Message{{Role: "user", Content: prompt}},
	}, r.ctr.Container.AI)
	if err == nil {
		return nil
	}
	if ctx.Response().Committed {
		return nil
	}

	return echo.NewHTTPError(http.StatusServiceUnavailable, err.Error())
}
