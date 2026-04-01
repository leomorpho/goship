package page

import (
	"html/template"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/leomorpho/goship/v2/framework/flash"
	"github.com/leomorpho/goship/v2/framework/htmx"
	"github.com/leomorpho/goship/v2/framework/http/authcontext"
	"github.com/leomorpho/goship/v2/framework/http/requestcontext"
)

// Base captures reusable web-page fields and behavior that are app-agnostic.
type Base struct {
	AppName string
	Domain  string
	Title   string

	Context echo.Context
	ToURL   func(name string, params ...any) string

	Path string
	URL  string

	Component templ.Component
	Data      any
	Form      any

	IsHome  bool
	IsAuth  bool
	IsAdmin bool

	StatusCode int
	CSRF       string
	Headers    map[string]string
	RequestID  string

	HTMX struct {
		Request  htmx.Request
		Response *htmx.Response
	}

	Cache struct {
		Enabled    bool
		Expiration time.Duration
		Tags       []string
	}

	IsIosDevice bool
}

func NewBase(ctx echo.Context) Base {
	p := Base{
		Context:    ctx,
		ToURL:      ctx.Echo().Reverse,
		Path:       ctx.Request().URL.Path,
		URL:        ctx.Request().URL.String(),
		StatusCode: http.StatusOK,
		Headers:    make(map[string]string),
		RequestID:  ctx.Response().Header().Get(echo.HeaderXRequestID),
	}
	p.IsHome = p.Path == "/"

	if csrf := ctx.Get(echomw.DefaultCSRFConfig.ContextKey); csrf != nil {
		p.CSRF = csrf.(string)
	}
	if u := ctx.Get(authcontext.AuthenticatedUserIDKey); u != nil {
		p.IsAuth = true
	}
	if isAdmin, ok := ctx.Get(authcontext.AuthenticatedUserIsAdminKey).(bool); ok {
		p.IsAdmin = isAdmin
	}
	if u := ctx.Get(requestcontext.IsFromIOSApp); u != nil {
		p.IsIosDevice = u.(bool)
	}

	p.HTMX.Request = htmx.GetRequest(ctx)

	return p
}

// GetMessages gets all flash messages for a given type.
func (p Base) GetMessages(typ uxflashmessages.Type) []template.HTML {
	strs := uxflashmessages.Get(p.Context, typ)
	ret := make([]template.HTML, len(strs))
	for k, v := range strs {
		ret[k] = template.HTML(v)
	}
	return ret
}
