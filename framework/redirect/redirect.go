package redirect

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/v2/framework/htmx"
)

type Redirect struct {
	ctx        echo.Context
	route      string
	params     []any
	query      url.Values
	statusCode int
}

func New(ctx echo.Context) *Redirect {
	return &Redirect{
		ctx:        ctx,
		query:      url.Values{},
		statusCode: http.StatusFound,
	}
}

func (r *Redirect) Route(name string) *Redirect {
	r.route = strings.TrimSpace(name)
	return r
}

func (r *Redirect) Params(params ...any) *Redirect {
	r.params = append([]any{}, params...)
	return r
}

func (r *Redirect) Query(q url.Values) *Redirect {
	r.query = cloneValues(q)
	return r
}

func (r *Redirect) Status(code int) *Redirect {
	if code >= 300 && code < 400 {
		r.statusCode = code
	}
	return r
}

func (r *Redirect) Go() error {
	target := r.url()
	if htmx.GetRequest(r.ctx).Boosted {
		r.ctx.Response().Header().Set(htmx.HeaderRedirect, target)
		r.ctx.Response().WriteHeader(http.StatusOK)
		return nil
	}
	return r.ctx.Redirect(r.statusCode, target)
}

func (r *Redirect) url() string {
	base := r.ctx.Echo().Reverse(r.route, r.params...)
	if len(r.query) == 0 {
		return base
	}
	encoded := r.query.Encode()
	if encoded == "" {
		return base
	}
	return base + "?" + encoded
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for key, values := range v {
		out[key] = append([]string(nil), values...)
	}
	return out
}
