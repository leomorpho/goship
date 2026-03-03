package ship

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeResourceName(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSnake  string
		wantKebab  string
		wantPascal string
		wantErr    bool
	}{
		{name: "snake", input: "blog_post", wantSnake: "blog_post", wantKebab: "blog-post", wantPascal: "BlogPost"},
		{name: "kebab", input: "blog-post", wantSnake: "blog_post", wantKebab: "blog-post", wantPascal: "BlogPost"},
		{name: "camel", input: "BlogPost", wantSnake: "blog_post", wantKebab: "blog-post", wantPascal: "BlogPost"},
		{name: "acronym", input: "APIClient", wantSnake: "api_client", wantKebab: "api-client", wantPascal: "ApiClient"},
		{name: "spaced", input: "user profile", wantSnake: "user_profile", wantKebab: "user-profile", wantPascal: "UserProfile"},
		{name: "empty", input: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeResourceName(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Snake != tt.wantSnake {
				t.Fatalf("snake = %q, want %q", got.Snake, tt.wantSnake)
			}
			if got.Kebab != tt.wantKebab {
				t.Fatalf("kebab = %q, want %q", got.Kebab, tt.wantKebab)
			}
			if got.Pascal != tt.wantPascal {
				t.Fatalf("pascal = %q, want %q", got.Pascal, tt.wantPascal)
			}
		})
	}
}

func TestGenerateResourceScaffold(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "app", "goship")

	result, err := generateResourceScaffold(resourceGenerateOptions{
		Name:  "contact_form",
		Path:  basePath,
		Auth:  "public",
		Views: "templ",
	})
	if err != nil {
		t.Fatalf("generateResourceScaffold error: %v", err)
	}

	handlerPath := filepath.Join(basePath, "web", "routes", "contact_form.go")
	viewPath := filepath.Join(basePath, "views", "web", "pages", "contact_form.templ")

	if _, err := os.Stat(handlerPath); err != nil {
		t.Fatalf("expected handler file, stat error: %v", err)
	}
	if _, err := os.Stat(viewPath); err != nil {
		t.Fatalf("expected templ file, stat error: %v", err)
	}
	if len(result.CreatedFiles) != 2 {
		t.Fatalf("created files = %d, want 2", len(result.CreatedFiles))
	}
	if !strings.Contains(result.RouteSnippet, `g.GET("/contact-form", contactForm.Get)`) {
		t.Fatalf("unexpected route snippet: %s", result.RouteSnippet)
	}
}

func TestGenerateResourceScaffold_NoneViews(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "app", "goship")

	result, err := generateResourceScaffold(resourceGenerateOptions{
		Name:  "inbox",
		Path:  basePath,
		Auth:  "auth",
		Views: "none",
	})
	if err != nil {
		t.Fatalf("generateResourceScaffold error: %v", err)
	}

	if len(result.CreatedFiles) != 1 {
		t.Fatalf("created files = %d, want 1", len(result.CreatedFiles))
	}
	if !strings.Contains(result.RouteSnippet, `onboardedGroup.GET("/inbox", inbox.Get)`) {
		t.Fatalf("unexpected auth route snippet: %s", result.RouteSnippet)
	}
}

func TestGenerateResourceScaffold_Validation(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "app", "goship")

	tests := []resourceGenerateOptions{
		{Name: "x", Path: basePath, Auth: "private", Views: "templ"},
		{Name: "x", Path: basePath, Auth: "public", Views: "jsx"},
		{Name: "", Path: basePath, Auth: "public", Views: "none"},
	}

	for _, tt := range tests {
		if _, err := generateResourceScaffold(tt); err == nil {
			t.Fatalf("expected error for opts: %+v", tt)
		}
	}
}

func TestGenerateResourceScaffold_RefuseOverwrite(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, "app", "goship")

	_, err := generateResourceScaffold(resourceGenerateOptions{
		Name:  "contact",
		Path:  basePath,
		Auth:  "public",
		Views: "none",
	})
	if err != nil {
		t.Fatalf("first generation should succeed: %v", err)
	}
	_, err = generateResourceScaffold(resourceGenerateOptions{
		Name:  "contact",
		Path:  basePath,
		Auth:  "public",
		Views: "none",
	})
	if err == nil {
		t.Fatalf("expected overwrite protection error")
	}
}

func TestWireRouteSnippet(t *testing.T) {
	base := `package goship

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}
`
	routerPath := filepath.Join(t.TempDir(), "router.go")
	if err := os.WriteFile(routerPath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	publicSnippet := "\t// ship:generated:contact\n\tcontact := routes.NewContactRoute(ctr)\n\tg.GET(\"/contact\", contact.Get).Name = routeNames.RouteNameContact\n"
	if err := wireRouteSnippet(routerPath, "public", publicSnippet); err != nil {
		t.Fatalf("wire public failed: %v", err)
	}
	// Idempotent second call.
	if err := wireRouteSnippet(routerPath, "public", publicSnippet); err != nil {
		t.Fatalf("wire public second call failed: %v", err)
	}

	authSnippet := "\t// ship:generated:inbox\n\tinbox := routes.NewInboxRoute(ctr)\n\tonboardedGroup.GET(\"/inbox\", inbox.Get).Name = routeNames.RouteNameInbox\n"
	if err := wireRouteSnippet(routerPath, "auth", authSnippet); err != nil {
		t.Fatalf("wire auth failed: %v", err)
	}

	contentBytes, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)
	if strings.Count(content, "ship:generated:contact") != 1 {
		t.Fatalf("expected one contact insertion, got content:\n%s", content)
	}
	if !strings.Contains(content, "ship:generated:inbox") {
		t.Fatalf("expected auth insertion, got content:\n%s", content)
	}
}

func TestWireRouteSnippet_MarkerErrors(t *testing.T) {
	routerPath := filepath.Join(t.TempDir(), "router.go")
	if err := os.WriteFile(routerPath, []byte("package goship\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := wireRouteSnippet(routerPath, "public", "\tfoo := bar\n")
	if err == nil {
		t.Fatalf("expected marker error")
	}

	err = wireRouteSnippet(routerPath, "internal", "\tfoo := bar\n")
	if err == nil {
		t.Fatalf("expected auth group validation error")
	}
}
