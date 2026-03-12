package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	backliteui "github.com/mikestefanello/backlite/ui"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/web/middleware"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/core"
)

type routes struct {
	controller ui.Controller
	db         *sql.DB
}

func registerRoutes(r core.Router, controller ui.Controller, db *sql.DB) error {
	h := &routes{controller: controller, db: db}
	g := r.Group("/admin", middleware.RequireAdmin())
	g.GET("", h.Index)
	g.GET("/queues", h.Queues)
	g.GET("/queues/*", h.Queues)
	g.GET("/:resource", h.List)
	g.GET("/:resource/new", h.New)
	g.POST("/:resource", h.Create)
	g.GET("/:resource/:id", h.Edit)
	g.PUT("/:resource/:id", h.Update)
	g.DELETE("/:resource/:id", h.Delete)
	return nil
}

func (r *routes) Index(ctx echo.Context) error {
	resources := RegisteredResources()
	if len(resources) == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "no admin resources registered")
	}
	return ctx.Redirect(http.StatusFound, "/admin/"+strings.ToLower(resources[0].PluralName))
}

func (r *routes) List(ctx echo.Context) error {
	res, err := r.resourceFromParam(ctx)
	if err != nil {
		return err
	}
	pageNum := parsePositiveInt(ctx.QueryParam("page"), 1)
	perPage := parsePositiveInt(ctx.QueryParam("per_page"), 20)

	rows, total, err := List(ctx.Request().Context(), r.db, res, pageNum, perPage)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("<html><body>")
	b.WriteString("<h1>Admin - ")
	b.WriteString(res.PluralName)
	b.WriteString("</h1>")
	b.WriteString(`<a href="/admin/` + strings.ToLower(res.PluralName) + `/new">Add new</a>`)
	b.WriteString("<ul>")
	for _, row := range rows {
		id := fmt.Sprint(row[res.IDField])
		b.WriteString("<li>")
		b.WriteString(fmt.Sprintf("<a href=\"/admin/%s/%s\">row %s</a>", strings.ToLower(res.PluralName), id, id))
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
	b.WriteString(fmt.Sprintf("<p>Total: %d</p>", total))
	b.WriteString("</body></html>")
	return ctx.HTML(http.StatusOK, b.String())
}

func (r *routes) New(ctx echo.Context) error {
	res, err := r.resourceFromParam(ctx)
	if err != nil {
		return err
	}
	return ctx.HTML(http.StatusOK, renderForm(res, map[string]any{}, map[string]string{}, ""))
}

func (r *routes) Create(ctx echo.Context) error {
	res, err := r.resourceFromParam(ctx)
	if err != nil {
		return err
	}
	values, formErrs := readValuesFromRequest(ctx, res)
	if len(formErrs) > 0 {
		return ctx.HTML(http.StatusBadRequest, renderForm(res, values, formErrs, ""))
	}
	if err := Create(ctx.Request().Context(), r.db, res, values); err != nil {
		return err
	}
	return ctx.Redirect(http.StatusFound, "/admin/"+strings.ToLower(res.PluralName))
}

func (r *routes) Edit(ctx echo.Context) error {
	res, err := r.resourceFromParam(ctx)
	if err != nil {
		return err
	}
	id := ctx.Param("id")
	row, err := Get(ctx.Request().Context(), r.db, res, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "resource not found")
		}
		return err
	}
	return ctx.HTML(http.StatusOK, renderForm(res, row, map[string]string{}, id))
}

func (r *routes) Update(ctx echo.Context) error {
	res, err := r.resourceFromParam(ctx)
	if err != nil {
		return err
	}
	values, formErrs := readValuesFromRequest(ctx, res)
	if len(formErrs) > 0 {
		return ctx.HTML(http.StatusBadRequest, renderForm(res, values, formErrs, ctx.Param("id")))
	}
	if err := Update(ctx.Request().Context(), r.db, res, ctx.Param("id"), values); err != nil {
		return err
	}
	return ctx.Redirect(http.StatusFound, "/admin/"+strings.ToLower(res.PluralName))
}

func (r *routes) Delete(ctx echo.Context) error {
	res, err := r.resourceFromParam(ctx)
	if err != nil {
		return err
	}
	if err := Delete(ctx.Request().Context(), r.db, res, ctx.Param("id")); err != nil {
		return err
	}
	return ctx.Redirect(http.StatusFound, "/admin/"+strings.ToLower(res.PluralName))
}

func (r *routes) Queues(ctx echo.Context) error {
	if !strings.EqualFold(strings.TrimSpace(r.controller.Container.Config.Adapters.Jobs), "backlite") {
		return echo.NewHTTPError(http.StatusNotFound, "queue monitor is only available with backlite")
	}

	handler, err := backliteui.NewHandler(backliteui.Config{
		DB:       r.db,
		BasePath: "/admin/queues",
	})
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	handler.Register(mux)
	mux.ServeHTTP(ctx.Response(), ctx.Request())
	return nil
}

func (r *routes) resourceFromParam(ctx echo.Context) (AdminResource, error) {
	name := strings.TrimSpace(ctx.Param("resource"))
	if name == "" {
		return AdminResource{}, echo.NewHTTPError(http.StatusBadRequest, "resource is required")
	}
	resource, ok := FindResourceByPluralName(name)
	if !ok {
		return AdminResource{}, echo.NewHTTPError(http.StatusNotFound, "unknown admin resource")
	}
	return resource, nil
}

func parsePositiveInt(raw string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func readValuesFromRequest(ctx echo.Context, res AdminResource) (map[string]any, map[string]string) {
	values := map[string]any{}
	errs := map[string]string{}
	for _, field := range res.Fields {
		if field.Type == FieldTypeReadOnly || strings.EqualFold(field.Name, res.IDField) {
			continue
		}
		raw := strings.TrimSpace(ctx.FormValue(field.Name))
		switch field.Type {
		case FieldTypeBool:
			values[field.Name] = raw == "on" || raw == "true" || raw == "1"
		case FieldTypeInt:
			if raw == "" {
				if field.Required {
					errs[field.Name] = "This field is required."
				}
				continue
			}
			i, err := strconv.Atoi(raw)
			if err != nil {
				errs[field.Name] = "Must be a valid number."
				continue
			}
			values[field.Name] = i
		default:
			if raw == "" && field.Required {
				errs[field.Name] = "This field is required."
				continue
			}
			values[field.Name] = raw
		}
	}
	return values, errs
}

func renderForm(res AdminResource, values map[string]any, errs map[string]string, id string) string {
	var b strings.Builder
	b.WriteString("<html><body>")
	b.WriteString("<h1>")
	if id == "" {
		b.WriteString("New ")
	} else {
		b.WriteString("Edit ")
	}
	b.WriteString(res.Name)
	b.WriteString("</h1>")
	if id == "" {
		b.WriteString(`<form method="post" action="/admin/` + strings.ToLower(res.PluralName) + `">`)
	} else {
		b.WriteString(`<form method="post" action="/admin/` + strings.ToLower(res.PluralName) + `/` + id + `">`)
		b.WriteString(`<input type="hidden" name="_method" value="PUT">`)
	}
	for _, field := range res.Fields {
		if field.Type == FieldTypeReadOnly {
			continue
		}
		b.WriteString("<div>")
		b.WriteString(`<label>` + field.Label + `</label>`)
		b.WriteString(`<input name="` + field.Name + `" value="` + fmt.Sprint(values[field.Name]) + `">`)
		if msg := errs[field.Name]; msg != "" {
			b.WriteString(`<p style="color:red">` + msg + `</p>`)
		}
		b.WriteString("</div>")
	}
	b.WriteString(`<button type="submit">Save</button></form>`)
	if id != "" {
		b.WriteString(`<form method="post" action="/admin/` + strings.ToLower(res.PluralName) + `/` + id + `"><input type="hidden" name="_method" value="DELETE"><button type="submit">Delete</button></form>`)
	}
	b.WriteString("</body></html>")
	return b.String()
}
