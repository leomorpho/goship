package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/framework/web/middleware"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/leomorpho/goship/modules/auditlog"
	"github.com/leomorpho/goship/modules/flags"
)

type routes struct {
	controller ui.Controller
	db         *sql.DB
	auditLogs  *auditlog.Service
	flags      *flags.Service
}

func registerRoutes(r core.Router, controller ui.Controller, db *sql.DB, auditLogs *auditlog.Service, flagsService *flags.Service) error {
	h := &routes{controller: controller, db: db, auditLogs: auditLogs, flags: flagsService}
	g := r.Group("/admin", middleware.RequireAdmin())
	g.GET("", h.Index)
	g.GET("/queues", h.Queues)
	g.GET("/queues/*", h.Queues)
	g.GET("/managed-settings", h.ManagedSettings)
	g.GET("/audit-logs", h.AuditLogs)
	g.GET("/flags", h.Flags)
	g.POST("/flags/:key/toggle", h.ToggleFlag)
	g.GET("/trash", h.Trash)
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
	if len(resources) == 0 || resources[0].PluralName == "" {
		return ctx.Redirect(http.StatusFound, "/admin/managed-settings")
	}
	return ctx.Redirect(http.StatusFound, "/admin/"+strings.ToLower(resources[0].PluralName))
}

func (r *routes) ManagedSettings(ctx echo.Context) error {
	statuses := r.controller.Container.Config.ManagedSettingStatuses()
	report := r.controller.Container.Config.Managed.RuntimeReport

	var b strings.Builder
	b.WriteString("<html><body><h1>Admin - Managed Runtime Settings</h1>")
	if strings.EqualFold(string(report.Mode), "managed") {
		b.WriteString("<p>Managed mode: enabled")
		if strings.TrimSpace(report.Authority) != "" {
			b.WriteString(" (authority: " + html.EscapeString(report.Authority) + ")")
		}
		b.WriteString("</p>")
	} else {
		b.WriteString("<p>Managed mode: disabled (standalone)</p>")
	}
	b.WriteString("<table><thead><tr><th>Setting</th><th>Value</th><th>State</th><th>Source</th></tr></thead><tbody>")
	for _, status := range statuses {
		b.WriteString("<tr>")
		b.WriteString("<td>" + html.EscapeString(status.Label) + "</td>")
		b.WriteString("<td>" + html.EscapeString(status.Value) + "</td>")
		b.WriteString("<td>" + html.EscapeString(string(status.Access)) + "</td>")
		b.WriteString("<td>" + html.EscapeString(string(status.Source)) + "</td>")
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table></body></html>")
	return ctx.HTML(http.StatusOK, b.String())
}

func (r *routes) Flags(ctx echo.Context) error {
	if r.flags == nil {
		return echo.NewHTTPError(http.StatusNotFound, "feature flags are not configured")
	}

	items, err := r.flags.List(ctx.Request().Context())
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("<html><body><h1>Admin - Feature Flags</h1>")
	b.WriteString(`<p><a href="/admin/managed-settings">Managed settings</a></p>`)
	if len(items) == 0 {
		b.WriteString("<p>No feature flags found.</p></body></html>")
		return ctx.HTML(http.StatusOK, b.String())
	}

	type flagsRow struct {
		flag      flags.Flag
		def       flags.FlagDefinition
		registered bool
	}
	registeredRows := make([]flagsRow, 0, len(items))
	orphanRows := make([]flagsRow, 0, len(items))
	for _, item := range items {
		def, ok := flags.Lookup(flags.FlagKey(item.Key))
		row := flagsRow{flag: item, def: def, registered: ok}
		if ok {
			registeredRows = append(registeredRows, row)
			continue
		}
		orphanRows = append(orphanRows, row)
	}
	rows := append(registeredRows, orphanRows...)

	b.WriteString("<table><thead><tr><th>Constant key</th><th>Description</th><th>Enabled</th><th>Code default</th><th>Rollout %</th><th>Action</th></tr></thead><tbody>")
	for _, row := range rows {
		status := "off"
		if row.flag.Enabled {
			status = "on"
		}
		description := row.flag.Description
		codeDefault := "Code default: n/a"
		warningBadge := ""
		rowStyle := ""
		if row.registered {
			description = row.def.Description
			if row.def.Default {
				codeDefault = "Code default: On"
			} else {
				codeDefault = "Code default: Off"
			}
		} else {
			warningBadge = ` <span>[Orphaned - not in code]</span>`
			rowStyle = ` style="opacity:0.75"`
		}
		b.WriteString("<tr" + rowStyle + ">")
		b.WriteString("<td><code>" + html.EscapeString(row.flag.Key) + "</code>" + warningBadge + "</td>")
		b.WriteString("<td>" + html.EscapeString(description) + "</td>")
		b.WriteString("<td>" + status + "</td>")
		b.WriteString("<td>" + html.EscapeString(codeDefault) + "</td>")
		b.WriteString("<td>" + strconv.Itoa(row.flag.RolloutPct) + "</td>")
		b.WriteString(`<td><form method="post" action="/admin/flags/` + html.EscapeString(row.flag.Key) + `/toggle">`)
		b.WriteString(`<button type="submit">Toggle</button></form></td>`)
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table></body></html>")
	return ctx.HTML(http.StatusOK, b.String())
}

func (r *routes) ToggleFlag(ctx echo.Context) error {
	if r.flags == nil {
		return echo.NewHTTPError(http.StatusNotFound, "feature flags are not configured")
	}

	key := strings.TrimSpace(ctx.Param("key"))
	if key == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "flag key is required")
	}
	if _, err := r.flags.Toggle(ctx.Request().Context(), key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "flag not found")
		}
		return err
	}

	return ctx.Redirect(http.StatusFound, "/admin/flags")
}

func (r *routes) Trash(ctx echo.Context) error {
	if r.db == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "database is not configured")
	}

	tables, err := listSoftDeleteTableSummaries(ctx.Request().Context(), r.db)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("<html><body><h1>Admin - Trash</h1>")
	b.WriteString(`<p><a href="/admin/managed-settings">Managed settings</a></p>`)
	if len(tables) == 0 {
		b.WriteString("<p>No soft-deleted rows found.</p>")
		b.WriteString("</body></html>")
		return ctx.HTML(http.StatusOK, b.String())
	}

	b.WriteString("<table><thead><tr><th>Table</th><th>Deleted Rows</th></tr></thead><tbody>")
	for _, table := range tables {
		b.WriteString("<tr>")
		b.WriteString("<td>" + html.EscapeString(table.Table) + "</td>")
		b.WriteString("<td>" + strconv.FormatInt(table.Count, 10) + "</td>")
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table></body></html>")
	return ctx.HTML(http.StatusOK, b.String())
}

func (r *routes) AuditLogs(ctx echo.Context) error {
	if r.auditLogs == nil {
		return echo.NewHTTPError(http.StatusNotFound, "audit logs are not configured")
	}

	filters := auditlog.ListFilters{
		Action:       strings.TrimSpace(ctx.QueryParam("action")),
		ResourceType: strings.TrimSpace(ctx.QueryParam("resource_type")),
		ResourceID:   strings.TrimSpace(ctx.QueryParam("resource_id")),
		Limit:        parsePositiveInt(ctx.QueryParam("limit"), 100),
	}
	if rawUserID := strings.TrimSpace(ctx.QueryParam("user_id")); rawUserID != "" {
		userID, err := strconv.ParseInt(rawUserID, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "user_id must be numeric")
		}
		filters.UserID = &userID
	}

	logs, err := r.auditLogs.List(ctx.Request().Context(), filters)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("<html><body><h1>Admin - Audit Logs</h1>")
	b.WriteString(`<p><a href="/admin/managed-settings">Managed settings</a></p>`)
	b.WriteString(`<form method="get" action="/admin/audit-logs">`)
	b.WriteString(`User <input name="user_id" value="` + html.EscapeString(ctx.QueryParam("user_id")) + `">`)
	b.WriteString(` Action <input name="action" value="` + html.EscapeString(filters.Action) + `">`)
	b.WriteString(` Resource <input name="resource_type" value="` + html.EscapeString(filters.ResourceType) + `">`)
	b.WriteString(` Resource ID <input name="resource_id" value="` + html.EscapeString(filters.ResourceID) + `">`)
	b.WriteString(` <button type="submit">Filter</button></form>`)
	b.WriteString("<table><thead><tr><th>When</th><th>User</th><th>Action</th><th>Resource</th><th>IP</th></tr></thead><tbody>")
	for _, entry := range logs {
		user := "-"
		if entry.UserID != nil {
			user = strconv.FormatInt(*entry.UserID, 10)
		}
		resource := entry.ResourceType
		if entry.ResourceID != "" {
			resource = strings.Trim(resource+":"+entry.ResourceID, ":")
		}
		b.WriteString("<tr>")
		b.WriteString("<td>" + html.EscapeString(entry.CreatedAt.Format(time.RFC3339)) + "</td>")
		b.WriteString("<td>" + html.EscapeString(user) + "</td>")
		b.WriteString("<td>" + html.EscapeString(entry.Action) + "</td>")
		b.WriteString("<td>" + html.EscapeString(resource) + "</td>")
		b.WriteString("<td>" + html.EscapeString(entry.IPAddress) + "</td>")
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table></body></html>")
	return ctx.HTML(http.StatusOK, b.String())
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
	b.WriteString(` | <a href="/admin/managed-settings">Managed settings</a>`)
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
	inspector := r.controller.Container.CoreJobsInspector
	if inspector == nil {
		return ctx.HTML(http.StatusServiceUnavailable, renderQueuesUnavailable("jobs inspector is not configured"))
	}

	jobID := strings.Trim(strings.TrimPrefix(ctx.Request().URL.Path, "/admin/queues"), "/")
	if jobID != "" {
		return r.renderQueueDetail(ctx, inspector, jobID)
	}
	return r.renderQueueList(ctx, inspector)
}

func (r *routes) renderQueueList(ctx echo.Context, inspector core.JobsInspector) error {
	records, err := inspector.List(ctx.Request().Context(), core.JobListFilter{Limit: 50})
	if err != nil {
		return ctx.HTML(http.StatusServiceUnavailable, renderQueuesUnavailable(err.Error()))
	}

	var b strings.Builder
	b.WriteString("<html><body><h1>Admin - Queue Monitor</h1>")
	b.WriteString(`<p><a href="/admin/managed-settings">Managed settings</a></p>`)
	if len(records) == 0 {
		b.WriteString("<p>No jobs found.</p></body></html>")
		return ctx.HTML(http.StatusOK, b.String())
	}

	b.WriteString("<table><thead><tr><th>ID</th><th>Name</th><th>Queue</th><th>Status</th><th>Attempts</th></tr></thead><tbody>")
	for _, record := range records {
		b.WriteString("<tr>")
		b.WriteString(`<td><a href="/admin/queues/` + html.EscapeString(record.ID) + `">` + html.EscapeString(record.ID) + "</a></td>")
		b.WriteString("<td>" + html.EscapeString(record.Name) + "</td>")
		b.WriteString("<td>" + html.EscapeString(record.Queue) + "</td>")
		b.WriteString("<td>" + html.EscapeString(string(record.Status)) + "</td>")
		b.WriteString("<td>" + strconv.Itoa(record.Attempt) + "/" + strconv.Itoa(record.MaxRetries) + "</td>")
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table></body></html>")
	return ctx.HTML(http.StatusOK, b.String())
}

func (r *routes) renderQueueDetail(ctx echo.Context, inspector core.JobsInspector, jobID string) error {
	record, ok, err := inspector.Get(ctx.Request().Context(), jobID)
	if err != nil {
		return ctx.HTML(http.StatusServiceUnavailable, renderQueuesUnavailable(err.Error()))
	}
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "job not found")
	}

	var b strings.Builder
	b.WriteString("<html><body><h1>Admin - Queue Job</h1>")
	b.WriteString(`<p><a href="/admin/queues">Back to queue monitor</a></p>`)
	b.WriteString("<dl>")
	writeQueueDetailRow(&b, "ID", record.ID)
	writeQueueDetailRow(&b, "Name", record.Name)
	writeQueueDetailRow(&b, "Queue", record.Queue)
	writeQueueDetailRow(&b, "Status", string(record.Status))
	writeQueueDetailRow(&b, "Attempts", fmt.Sprintf("%d/%d", record.Attempt, record.MaxRetries))
	writeQueueDetailRow(&b, "Run At", formatQueueTime(record.RunAt))
	writeQueueDetailRow(&b, "Created At", formatQueueTime(record.CreatedAt))
	writeQueueDetailRow(&b, "Updated At", formatQueueTime(record.UpdatedAt))
	if strings.TrimSpace(record.LastError) != "" {
		writeQueueDetailRow(&b, "Last Error", record.LastError)
	}
	if len(record.Payload) > 0 {
		writeQueueDetailRow(&b, "Payload", string(record.Payload))
	}
	b.WriteString("</dl></body></html>")
	return ctx.HTML(http.StatusOK, b.String())
}

func renderQueuesUnavailable(reason string) string {
	var b strings.Builder
	b.WriteString("<html><body><h1>Admin - Queue Monitor</h1>")
	b.WriteString("<p>Queue monitor unavailable.</p>")
	if strings.TrimSpace(reason) != "" {
		b.WriteString("<p>" + html.EscapeString(reason) + "</p>")
	}
	b.WriteString("</body></html>")
	return b.String()
}

func writeQueueDetailRow(b *strings.Builder, label, value string) {
	if b == nil {
		return
	}
	b.WriteString("<dt>" + html.EscapeString(label) + "</dt><dd>" + html.EscapeString(value) + "</dd>")
}

func formatQueueTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
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
