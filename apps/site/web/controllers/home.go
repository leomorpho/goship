package controllers

import (
	"fmt"

	"github.com/leomorpho/goship/apps/site/views"
	"github.com/leomorpho/goship/apps/site/views/web/layouts/gen"
	"github.com/leomorpho/goship/apps/site/views/web/pages/gen"
	"github.com/leomorpho/goship/apps/site/web/ui"
	"github.com/leomorpho/goship/apps/site/web/viewmodels"

	"github.com/labstack/echo/v4"
)

type (
	home struct {
		ui.Controller
	}
)

func (c *home) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)

	if page.AuthUser != nil {
		return c.Redirect(ctx, "dashboard")

	}

	page.Layout = layouts.Main
	page.Name = templates.PageHome
	page.Metatags.Description = "Welcome to the homepage."
	page.Metatags.Keywords = []string{"Go", "MVC", "Web", "Software"}
	page.Pager = ui.NewPager(ctx, 4)
	page.Data = c.fetchPosts(&page.Pager)
	page.Component = pages.Home(&page)
	page.HTMX.Request.Boosted = true

	return c.RenderPage(ctx, page)
}

// fetchPosts is an mock example of fetching posts to illustrate how paging works
func (c *home) fetchPosts(pager *ui.Pager) []viewmodels.Post {
	pager.SetItems(20)
	posts := make([]viewmodels.Post, 20)

	for k := range posts {
		posts[k] = viewmodels.Post{
			Title: fmt.Sprintf("Post example #%d", k+1),
			Body:  fmt.Sprintf("Lorem ipsum example #%d ddolor sit amet, consectetur adipiscing elit. Nam elementum vulputate tristique.", k+1),
		}
	}

	return posts[pager.GetOffset() : pager.GetOffset()+pager.ItemsPerPage]
}
