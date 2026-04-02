package generators

import (
	"fmt"
	"strings"
)

type StarterGeneratedRouteSpec struct {
	OwnershipKind string
	Snake         string
	Kebab         string
	Pascal        string
	RoutePath     string
	Actions       []string
	Description   string
}

func renderStarterGeneratedPageSpec(spec StarterGeneratedRouteSpec) string {
	description := strings.TrimSpace(spec.Description)
	if description == "" {
		description = fmt.Sprintf("Starter scaffold for %s with actions: %s.", spec.Kebab, strings.Join(spec.Actions, ", "))
	}
	ownershipKind := strings.TrimSpace(spec.OwnershipKind)
	if ownershipKind == "" {
		ownershipKind = "resource"
	}
	return generatedGoFileHeader(ownershipKind, spec.Snake) + fmt.Sprintf(`package pages

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

func %s() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, err := io.WriteString(w, %q)
		return err
	})
}
`, spec.Pascal, fmt.Sprintf(`<section data-component=%q><div data-slot="status">Generated starter route</div><h1>%s</h1><p>%s</p></section>`, spec.Kebab, spec.Pascal, description))
}

func renderStarterRoutePreview(spec StarterGeneratedRouteSpec, auth string) string {
	target := "public"
	if auth == "auth" {
		target = "auth"
	}
	return fmt.Sprintf(`// In starter %s routes:
%s`, target, strings.TrimSpace(renderStarterRouteInsertSnippetForSpec(spec)))
}

func renderStarterRouteInsertSnippetForSpec(spec StarterGeneratedRouteSpec) string {
	actionList := make([]string, 0, len(spec.Actions))
	for _, action := range spec.Actions {
		actionList = append(actionList, fmt.Sprintf("%q", action))
	}
	return fmt.Sprintf(`			// ship:generated:%s
			{Name: routenames.RouteName%s, Path: %q, Page: templates.Page%s, Kind: RouteKindResource, Actions: []string{%s}},
`, spec.Snake, spec.Pascal, spec.RoutePath, spec.Pascal, strings.Join(actionList, ", "))
}
