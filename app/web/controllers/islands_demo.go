package controllers

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/ui"
	i18nmodule "github.com/leomorpho/goship/modules/i18n"
)

type islandsDemo struct {
	ctr ui.Controller
}

func NewIslandsDemoRoute(ctr ui.Controller) islandsDemo {
	return islandsDemo{ctr: ctr}
}

func (r *islandsDemo) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.AppName = string(r.ctr.Container.Config.App.Name)
	page.Name = templates.PageIslandsDemo
	page.Data = ui.IslandsDemoBootstrap{
		Locale: r.currentLocale(ctx),
		Labels: map[string]string{
			"VanillaCounter": r.translate(ctx, "demo.islands.vanilla.label", "Vanilla JS Counter"),
			"ReactCounter":   r.translate(ctx, "demo.islands.react.label", "React Counter"),
			"VueCounter":     r.translate(ctx, "demo.islands.vue.label", "Vue Counter"),
			"SvelteCounter":  r.translate(ctx, "demo.islands.svelte.label", "Svelte Counter"),
		},
	}
	page.Component = pages.IslandsDemoPage(&page)

	return r.ctr.RenderPage(ctx, page)
}

func (r *islandsDemo) currentLocale(ctx echo.Context) string {
	if r == nil || r.ctr.Container == nil || r.ctr.Container.I18n == nil {
		return "en"
	}
	raw := i18nmodule.LanguageFromContext(ctx.Request().Context())
	return r.ctr.Container.I18n.NormalizeLanguage(raw)
}

func (r *islandsDemo) translate(ctx echo.Context, key string, fallback string) string {
	if r == nil || r.ctr.Container == nil || r.ctr.Container.I18n == nil {
		return fallback
	}
	translated := strings.TrimSpace(r.ctr.Container.I18n.T(ctx.Request().Context(), key))
	if translated == "" || translated == key {
		return fallback
	}
	return translated
}
