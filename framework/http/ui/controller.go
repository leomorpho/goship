package ui

import (
	"bytes"
	ctx "context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/a-h/templ"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	frameworkmiddleware "github.com/leomorpho/goship/framework/middleware"
	redirector "github.com/leomorpho/goship/framework/redirect"
	webmiddleware "github.com/leomorpho/goship/framework/http/middleware"
	"github.com/leomorpho/goship/framework/http/requestcontext"

	"github.com/labstack/echo/v4"
)

// Controller provides base functionality and dependencies to controllers.
// The proposed pattern is to embed a Controller in each individual route struct and to use
// the router to inject the container so your routes have access to the services within the container
type Controller struct {
	// Container stores a services container which contains dependencies
	Container *frameworkbootstrap.Container
}

// NewController creates a new Controller
func NewController(c *frameworkbootstrap.Container) Controller {
	return Controller{
		Container: c,
	}
}

func (c *Controller) preparePage(page Page, domain string) (Page, error) {
	if page.Name == "" {
		return page, errors.New("page render failed due to missing name")
	}
	if page.Component == nil {
		return page, errors.New("page render failed due to missing component")
	}
	if page.AppName == "" {
		page.AppName = string(c.Container.Config.App.Name)
	}
	page.Domain = domain
	return page, nil
}

// RenderToHTMLBlob renders a page to HTML without returning any HTTP reponse, unlike RenderPage.
// It is meant to generate small blobs of HTML, for example for use in SSE events.
func (c *Controller) RenderToHTMLBlob(ctx ctx.Context, page Page) (string, error) {
	buf := &bytes.Buffer{}
	page, err := c.preparePage(page, c.Container.Config.HTTP.Domain)
	if err != nil {
		return "", err
	}

	// Render the templates only for the content portion of the page
	renderCtx := templ.WithNonce(ctx, frameworkmiddleware.CSPNonce(page.Context))
	err = page.Component.Render(renderCtx, buf)

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
	page, err := c.preparePage(page, c.Container.Config.HTTP.Domain)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Check if this is an HTMX non-boosted request which indicates that only partial
	// content should be rendered
	renderCtx := templ.WithNonce(ctx.Request().Context(), frameworkmiddleware.CSPNonce(ctx))
	if page.HTMX.Request.Enabled && !page.HTMX.Request.Boosted {
		// Render the templates only for the content portion of the page
		err = page.Component.Render(renderCtx, buf)
	} else {
		// Render the templates for the Page
		// If the page Layout is set, that will be used to wrap the page component
		component := page.Component
		if page.Layout != nil {
			component = page.Layout(component, &page)
		}
		err = component.Render(renderCtx, buf)
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
	if c == nil || c.Container == nil || c.Container.Cache == nil {
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
	cp := webmiddleware.CachedPage{
		URL:        key,
		HTML:       html.Bytes(),
		Headers:    headers,
		StatusCode: ctx.Response().Status,
	}

	err := c.Container.Cache.
		Set().
		Group(webmiddleware.CachedPageGroup).
		Key(key).
		Tags(page.Cache.Tags...).
		Expiration(page.Cache.Expiration).
		Data(cp).
		Save(ctx.Request().Context())

	switch {
	case err == nil:
		ctx.Logger().Info("cached page")
	case !requestcontext.IsCanceledError(err):
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
	r := redirector.New(ctx).
		Route(route).
		Params(routeParams...).
		Status(statusCode)
	if q := parseQueryParams(queryParams); len(q) > 0 {
		r.Query(q)
	}
	return r.Go()
}

func parseQueryParams(raw string) url.Values {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	s = strings.TrimPrefix(s, "?")
	q, err := url.ParseQuery(s)
	if err != nil {
		return nil
	}
	return q
}

// Fail is a helper to fail a request by returning a 500 error and logging the error
func (c *Controller) Fail(err error, log string) error {
	return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("%s: %v", log, err))
}
