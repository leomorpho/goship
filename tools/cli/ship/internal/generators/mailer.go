package generators

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type MakeMailerOptions struct {
	Name string
}

type MakeMailerDeps struct {
	Out io.Writer
	Err io.Writer
	Cwd string
}

func RunMakeMailer(args []string, d MakeMailerDeps) int {
	opts, err := ParseMakeMailerArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid make:mailer arguments: %v\n", err)
		return 1
	}

	cwd := d.Cwd
	if strings.TrimSpace(cwd) == "" {
		var wdErr error
		cwd, wdErr = os.Getwd()
		if wdErr != nil {
			fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", wdErr)
			return 1
		}
	}

	tokens := splitWords(opts.Name)
	if len(tokens) == 0 {
		fmt.Fprintln(d.Err, "invalid make:mailer arguments: usage: ship make:mailer <Name>")
		return 1
	}
	pascal := toPascalFromParts(tokens)
	snake := strings.Join(tokens, "_")
	kebab := strings.Join(tokens, "-")

	templatePath := filepath.Join(cwd, "app", "views", "emails", snake+".templ")
	if _, err := os.Stat(templatePath); err == nil {
		fmt.Fprintf(d.Err, "refusing to overwrite existing mailer template: %s\n", templatePath)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(templatePath), 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create email views directory: %v\n", err)
		return 1
	}
	if err := os.WriteFile(templatePath, []byte(renderMailerTemplateFile(pascal)), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write mailer template: %v\n", err)
		return 1
	}

	mailPreviewPath := filepath.Join(cwd, "app", "web", "controllers", "mail_preview.go")
	if err := updateMailerPreviewController(mailPreviewPath, pascal, kebab); err != nil {
		fmt.Fprintf(d.Err, "failed to wire mail preview controller: %v\n", err)
		return 1
	}

	routerPath := filepath.Join(cwd, "app", "router.go")
	if err := updateMailerPreviewRoutes(routerPath, pascal, kebab); err != nil {
		fmt.Fprintf(d.Err, "failed to wire mail preview route: %v\n", err)
		return 1
	}

	routeNamesPath := filepath.Join(cwd, "app", "web", "routenames", "routenames.go")
	if err := updateMailerRouteNames(routeNamesPath, pascal); err != nil {
		fmt.Fprintf(d.Err, "failed to wire mail preview route name: %v\n", err)
		return 1
	}

	writeGeneratorReport(
		d.Out,
		"mailer",
		false,
		[]string{templatePath},
		[]string{mailPreviewPath, routerPath, routeNamesPath},
		nil,
		nil,
	)
	return 0
}

func ParseMakeMailerArgs(args []string) (MakeMailerOptions, error) {
	opts := MakeMailerOptions{}
	if len(args) == 0 {
		return opts, errors.New("usage: ship make:mailer <Name>")
	}
	opts.Name = strings.TrimSpace(args[0])
	if opts.Name == "" || strings.HasPrefix(opts.Name, "-") {
		return opts, errors.New("usage: ship make:mailer <Name>")
	}
	for i := 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
		return opts, fmt.Errorf("unexpected argument: %s", args[i])
	}
	return opts, nil
}

func renderMailerTemplateFile(pascal string) string {
	return fmt.Sprintf(`package emails

import (
	controller "github.com/leomorpho/goship/framework/web/ui"
	"github.com/leomorpho/goship/framework/web/viewmodels"
)

templ %s(page *controller.Page) {
	if data, ok := page.Data.(viewmodels.EmailDefaultData); ok {
		<div>
			<h1>%s Email</h1>
			<p>Hello from { data.AppName }.</p>
			<p>Support: { data.SupportEmail }</p>
			<p>Domain: { data.Domain }</p>
		</div>
	}
}
`, pascal, pascal)
}

func updateMailerPreviewController(path, pascal, kebab string) error {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	linkLine := fmt.Sprintf("\t\t\"/dev/mail/%s\",", kebab)
	updated, changed, err := insertAfterAnchor(content, "\t\t\"/dev/mail/verify-email\",", linkLine)
	if err != nil {
		return err
	}
	content = updated

	methodSnippet := fmt.Sprintf(`
func (r *mailPreview) %s(ctx echo.Context) error {
	data := viewmodels.NewEmailDefaultData()
	data.AppName = string(r.ctr.Container.Config.App.Name)
	data.SupportEmail = r.ctr.Container.Config.App.SupportEmail
	data.Domain = r.ctr.Container.Config.HTTP.Domain

	page := &ui.Page{
		Base: frameworkpage.Base{
			Data: data,
		},
	}
	return r.renderEmailPreview(ctx, emailviews.%s(page))
}

`, pascal, pascal)
	updated, methodChanged, err := insertBeforeAnchor(content, "func (r *mailPreview) renderEmailPreview", methodSnippet)
	if err != nil {
		return err
	}
	content = updated

	if changed || methodChanged {
		return os.WriteFile(path, []byte(content), 0o644)
	}
	return nil
}

func updateMailerPreviewRoutes(path, pascal, kebab string) error {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	snippet := fmt.Sprintf("\tmailGroup.GET(\"/%s\", mailPreview.%s).Name = routeNames.RouteNameMailPreview%s", kebab, pascal, pascal)
	updated, changed, err := insertAfterAnchor(content, "\tmailGroup.GET(\"/verify-email\", mailPreview.VerifyEmail).Name = routeNames.RouteNameMailPreviewVerifyEmail", snippet)
	if err != nil {
		return err
	}
	if changed {
		return os.WriteFile(path, []byte(updated), 0o644)
	}
	return nil
}

func updateMailerRouteNames(path, pascal string) error {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	snippet := fmt.Sprintf("\tRouteNameMailPreview%s = \"mail_preview.%s\"", pascal, strings.ToLower(strings.Join(splitWords(pascal), "_")))
	updated, changed, err := insertAfterAnchor(content, "\tRouteNameMailPreviewVerifyEmail    = \"mail_preview.verify_email\"", snippet)
	if err != nil {
		return err
	}
	if changed {
		return os.WriteFile(path, []byte(updated), 0o644)
	}
	return nil
}
