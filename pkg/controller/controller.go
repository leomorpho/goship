package controller

import (
	"bytes"
	ctx "context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/htmx"
	"github.com/mikestefanello/pagoda/pkg/middleware"
	"github.com/mikestefanello/pagoda/pkg/services"

	"github.com/labstack/echo/v4"
)

// Controller provides base functionality and dependencies to routes.
// The proposed pattern is to embed a Controller in each individual route struct and to use
// the router to inject the container so your routes have access to the services within the container
type Controller struct {
	// Container stores a services container which contains dependencies
	Container *services.Container
}

// NewController creates a new Controller
func NewController(c *services.Container) Controller {
	return Controller{
		Container: c,
	}
}

// TODO: RenderPage is repeated in RenderToHTMLBlob. Combine the two for clarity/succintness.
// RenderToHTMLBlob renders a page to HTML without returning any HTTP reponse, unlike RenderPage.
// It is meant to generate small blobs of HTML, for example for use in SSE events.
func (c *Controller) RenderToHTMLBlob(ctx ctx.Context, page Page) (string, error) {
	buf := &bytes.Buffer{}
	var err error

	// Page name is required
	if page.Name == "" {
		return "", errors.New("template name should be set")
	}

	// Page component required
	if page.Component == nil {
		return "", errors.New("page render failed due to missing component")
	}

	// Use the app name in configuration if a value was not set
	if page.AppName == "" {
		page.AppName = string(c.Container.Config.App.Name)

	}

	page.Domain = c.Container.Config.HTTP.Domain

	// Render the templates only for the content portion of the page
	err = page.Component.Render(ctx, buf)

	if err != nil {
		return "", c.Fail(err, "failed to parse and execute templates")
	}

	return convertToSingleLineString(buf), nil
}

func convertToSingleLineString(blob *bytes.Buffer) string {
	// Convert the buffer to a string
	str := blob.String()

	// Use regular expression to remove all extra white spaces
	re := regexp.MustCompile(`\s+`)
	str = re.ReplaceAllString(str, " ")

	// Further replacements for newline, carriage return, and tabs if needed
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\r", "")
	str = strings.ReplaceAll(str, "\t", "")

	return str
}

// RenderPage renders a Page as an HTTP response
func (c *Controller) RenderPage(ctx echo.Context, page Page) error {
	buf := &bytes.Buffer{}
	var err error

	// TODO: there is a freaking bug where setting the domain here sets it for the layout (it appears in the navbar),
	// but it is not set for any components trying to access it in the page. Currently, the only fix is to set it
	// in the route's page object. I am confused bywhat's happening...
	page.Domain = c.Container.Config.HTTP.Domain

	// Page name is required
	if page.Name == "" {
		return echo.NewHTTPError(http.StatusInternalServerError, "page render failed due to missing name")
	}

	// Page component required
	if page.Component == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "page render failed due to missing component")
	}

	// Use the app name in configuration if a value was not set
	if page.AppName == "" {
		page.AppName = string(c.Container.Config.App.Name)

	}

	// Check if this is an HTMX non-boosted request which indicates that only partial
	// content should be rendered
	if page.HTMX.Request.Enabled && !page.HTMX.Request.Boosted {
		// Render the templates only for the content portion of the page
		err = page.Component.Render(ctx.Request().Context(), buf)
	} else {
		// Render the templates for the Page
		// If the page Layout is set, that will be used to wrap the page component
		component := page.Component
		if page.Layout != nil {
			component = page.Layout(component, &page)
		}
		err = component.Render(ctx.Request().Context(), buf)
	}

	if err != nil {
		return c.Fail(err, "failed to parse and execute templates")
	}

	// Set the status code
	ctx.Response().Status = page.StatusCode

	// Set any headers
	for k, v := range page.Headers {
		ctx.Response().Header().Set(k, v)
	}

	// Apply the HTMX response, if one
	if page.HTMX.Response != nil {
		page.HTMX.Response.Apply(ctx)
	}

	// Cache this page, if caching was enabled
	c.cachePage(ctx, page, buf)

	return ctx.HTMLBlob(ctx.Response().Status, buf.Bytes())
}

// RenderJSON renders a JSON response
func (c *Controller) RenderJSON(ctx echo.Context, data interface{}) error {
	// data should be the structure that you want to serialize into JSON
	// for example, it could be a struct representing the API response

	statusCode := http.StatusOK

	// Set custom headers as needed
	ctx.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)

	// Return the JSON response
	return ctx.JSON(statusCode, data)
}

// cachePage caches the HTML for a given Page if the Page has caching enabled
func (c *Controller) cachePage(ctx echo.Context, page Page, html *bytes.Buffer) {
	if !page.Cache.Enabled || page.IsAuth {
		return
	}

	// If no expiration time was provided, default to the configuration value
	if page.Cache.Expiration == 0 {
		page.Cache.Expiration = c.Container.Config.Cache.Expiration.Page
	}

	// Extract the headers
	headers := make(map[string]string)
	for k, v := range ctx.Response().Header() {
		headers[k] = v[0]
	}

	// The request URL is used as the cache key so the middleware can serve the
	// cached page on matching requests
	key := ctx.Request().URL.String()
	cp := middleware.CachedPage{
		URL:        key,
		HTML:       html.Bytes(),
		Headers:    headers,
		StatusCode: ctx.Response().Status,
	}

	err := c.Container.Cache.
		Set().
		Group(middleware.CachedPageGroup).
		Key(key).
		Tags(page.Cache.Tags...).
		Expiration(page.Cache.Expiration).
		Data(cp).
		Save(ctx.Request().Context())

	switch {
	case err == nil:
		ctx.Logger().Info("cached page")
	case !context.IsCanceledError(err):
		ctx.Logger().Errorf("failed to cache page: %v", err)
	}
}

// Redirect redirects to a given route name with optional route parameters.
func (c *Controller) Redirect(ctx echo.Context, route string, routeParams ...any) error {
	return c.redirectHelper(ctx, route, http.StatusFound, "", routeParams...)
}

// Redirect redirects to a given route name with optional route parameters.
func (c *Controller) RedirectWithDetails(ctx echo.Context, route string, queryParams string, statusCode int, routeParams ...any) error {
	return c.redirectHelper(ctx, route, statusCode, queryParams, routeParams...)
}

// redirectHelper contains the common logic for redirection.
func (c *Controller) redirectHelper(ctx echo.Context, route string, statusCode int, queryParams string, routeParams ...any) error {
	url := ctx.Echo().Reverse(route, routeParams...) + queryParams

	if htmx.GetRequest(ctx).Boosted {
		htmx.Response{
			Redirect: url,
		}.Apply(ctx)
		return nil
	} else {
		return ctx.Redirect(statusCode, url)
	}
}

// Fail is a helper to fail a request by returning a 500 error and logging the error
func (c *Controller) Fail(err error, log string) error {
	return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("%s: %v", log, err))
}
