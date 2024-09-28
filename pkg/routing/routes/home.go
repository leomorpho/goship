package routes

import (
	"fmt"

	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"

	"github.com/labstack/echo/v4"
)

type (
	home struct {
		controller.Controller
	}
)

func (c *home) Get(ctx echo.Context) error {
	page := controller.NewPage(ctx)

	if page.AuthUser != nil {
		return c.Redirect(ctx, "dashboard")

	}

	page.Layout = layouts.Main
	page.Name = templates.PageHome
	page.Metatags.Description = "Welcome to the homepage."
	page.Metatags.Keywords = []string{"Go", "MVC", "Web", "Software"}
	page.Pager = controller.NewPager(ctx, 4)
	page.Data = c.fetchPosts(&page.Pager)
	page.Component = pages.Home(&page)
	page.HTMX.Request.Boosted = true

	return c.RenderPage(ctx, page)
}

// fetchPosts is an mock example of fetching posts to illustrate how paging works
func (c *home) fetchPosts(pager *controller.Pager) []types.Post {
	pager.SetItems(20)
	posts := make([]types.Post, 20)

	for k := range posts {
		posts[k] = types.Post{
			Title: fmt.Sprintf("Post example #%d", k+1),
			Body:  fmt.Sprintf("Lorem ipsum example #%d ddolor sit amet, consectetur adipiscing elit. Nam elementum vulputate tristique.", k+1),
		}
	}

	return posts[pager.GetOffset() : pager.GetOffset()+pager.ItemsPerPage]
}
