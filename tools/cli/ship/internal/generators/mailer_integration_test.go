package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMakeMailer_GeneratesTemplateAndPreviewWiring(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "app", "web", "controllers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "app", "web", "routenames"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "app"), 0o755); err != nil {
		t.Fatal(err)
	}

	mailPreview := `package controllers

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	emailviews "github.com/leomorpho/goship/app/views/emails/gen"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	frameworkpage "github.com/leomorpho/goship/framework/web/page"
)

type mailPreview struct { ctr ui.Controller }
func NewMailPreviewRoute(ctr ui.Controller) mailPreview { return mailPreview{ctr: ctr} }
func (r *mailPreview) Index(ctx echo.Context) error {
	links := []string{
		"/dev/mail/welcome",
		"/dev/mail/password-reset",
		"/dev/mail/verify-email",
	}
	var b strings.Builder
	b.WriteString("<html><body><h1>Email previews</h1><ul>")
	for _, link := range links { b.WriteString(link) }
	return ctx.HTML(http.StatusOK, b.String())
}
func (r *mailPreview) renderEmailPreview(ctx echo.Context, component templ.Component) error { return nil }
`
	if err := os.WriteFile(filepath.Join(root, "app", "web", "controllers", "mail_preview.go"), []byte(mailPreview), 0o644); err != nil {
		t.Fatal(err)
	}

	router := `package app
func registerMailPreviewRoutes() {
	mailGroup.GET("/verify-email", mailPreview.VerifyEmail).Name = routeNames.RouteNameMailPreviewVerifyEmail
}
`
	if err := os.WriteFile(filepath.Join(root, "app", "router.go"), []byte(router), 0o644); err != nil {
		t.Fatal(err)
	}

	routeNames := `package routenames
const (
	RouteNameMailPreviewVerifyEmail    = "mail_preview.verify_email"
)
`
	if err := os.WriteFile(filepath.Join(root, "app", "web", "routenames", "routenames.go"), []byte(routeNames), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunMakeMailer([]string{"WelcomeDigest"}, MakeMailerDeps{Out: out, Err: errOut, Cwd: root})
	if code != 0 {
		t.Fatalf("exit code=%d stderr=%s", code, errOut.String())
	}

	templateBytes, err := os.ReadFile(filepath.Join(root, "app", "views", "emails", "welcome_digest.templ"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(templateBytes), "templ WelcomeDigest") {
		t.Fatalf("template missing generated component:\n%s", string(templateBytes))
	}

	mailPreviewBytes, err := os.ReadFile(filepath.Join(root, "app", "web", "controllers", "mail_preview.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(mailPreviewBytes)
	for _, required := range []string{`"/dev/mail/welcome-digest"`, "func (r *mailPreview) WelcomeDigest", "emailviews.WelcomeDigest(page)"} {
		if !strings.Contains(text, required) {
			t.Fatalf("mail preview missing %q\n%s", required, text)
		}
	}

	routerBytes, err := os.ReadFile(filepath.Join(root, "app", "router.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(routerBytes), `mailGroup.GET("/welcome-digest", mailPreview.WelcomeDigest).Name = routeNames.RouteNameMailPreviewWelcomeDigest`) {
		t.Fatalf("router missing generated preview route:\n%s", string(routerBytes))
	}

	routeNamesBytes, err := os.ReadFile(filepath.Join(root, "app", "web", "routenames", "routenames.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(routeNamesBytes), `RouteNameMailPreviewWelcomeDigest = "mail_preview.welcome_digest"`) {
		t.Fatalf("route names missing generated preview route name:\n%s", string(routeNamesBytes))
	}
}

func TestRunMakeMailer_IsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "app", "web", "controllers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "app", "web", "routenames"), 0o755); err != nil {
		t.Fatal(err)
	}
	mailPreview := `package controllers

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	emailviews "github.com/leomorpho/goship/app/views/emails/gen"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	frameworkpage "github.com/leomorpho/goship/framework/web/page"
)

type mailPreview struct { ctr ui.Controller }
func (r *mailPreview) Index(ctx echo.Context) error {
	links := []string{
		"/dev/mail/welcome",
		"/dev/mail/password-reset",
		"/dev/mail/verify-email",
	}
	var b strings.Builder
	b.WriteString("<html><body><h1>Email previews</h1><ul>")
	for _, link := range links { b.WriteString(link) }
	return ctx.HTML(http.StatusOK, b.String())
}
func (r *mailPreview) renderEmailPreview(ctx echo.Context, component templ.Component) error { return nil }
`
	if err := os.WriteFile(filepath.Join(root, "app", "web", "controllers", "mail_preview.go"), []byte(mailPreview), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "router.go"), []byte("package app\n\tmailGroup.GET(\"/verify-email\", mailPreview.VerifyEmail).Name = routeNames.RouteNameMailPreviewVerifyEmail\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "web", "routenames", "routenames.go"), []byte("package routenames\n\tRouteNameMailPreviewVerifyEmail    = \"mail_preview.verify_email\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	deps := MakeMailerDeps{Out: out, Err: errOut, Cwd: root}
	if code := RunMakeMailer([]string{"WelcomeDigest"}, deps); code != 0 {
		t.Fatalf("first run failed: code=%d stderr=%s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := RunMakeMailer([]string{"WelcomeDigest"}, deps); code == 0 {
		t.Fatal("expected duplicate run to fail")
	}
}
