package ship

import (
	"strings"
	"testing"
)

func TestParseMakeControllerArgs(t *testing.T) {
	opts, err := parseMakeControllerArgs([]string{"Posts", "--actions", "index,show", "--auth", "auth", "--wire"})
	if err != nil {
		t.Fatalf("parseMakeControllerArgs error = %v", err)
	}
	if opts.Name != "Posts" {
		t.Fatalf("name = %q, want Posts", opts.Name)
	}
	if len(opts.Actions) != 2 || opts.Actions[0] != "index" || opts.Actions[1] != "show" {
		t.Fatalf("actions = %v, want [index show]", opts.Actions)
	}
	if opts.Auth != "auth" {
		t.Fatalf("auth = %q, want auth", opts.Auth)
	}
	if !opts.Wire {
		t.Fatal("wire = false, want true")
	}
}

func TestParseMakeControllerArgs_InvalidAction(t *testing.T) {
	_, err := parseMakeControllerArgs([]string{"Posts", "--actions", "index,publish"})
	if err == nil {
		t.Fatal("expected invalid action error")
	}
}

func TestNormalizeControllerName(t *testing.T) {
	names, err := normalizeControllerName("BlogPostsController")
	if err != nil {
		t.Fatalf("normalizeControllerName error = %v", err)
	}
	if names.FileName != "blog_posts.go" {
		t.Fatalf("file name = %q, want blog_posts.go", names.FileName)
	}
	if names.BaseKebab != "blog-posts" {
		t.Fatalf("kebab = %q, want blog-posts", names.BaseKebab)
	}
}

func TestRenderControllerRouteSnippet(t *testing.T) {
	names := controllerNames{
		BaseSnake: "posts",
		BaseKebab: "posts",
		BaseTitle: "Posts",
		VarName:   "posts",
	}
	snippet := renderControllerRouteSnippet(names, []string{"index", "show", "create"}, "public")
	if !strings.Contains(snippet, `g.GET("/posts", posts.Index)`) {
		t.Fatalf("missing index route:\n%s", snippet)
	}
	if !strings.Contains(snippet, `g.GET("/posts/:id", posts.Show)`) {
		t.Fatalf("missing show route:\n%s", snippet)
	}
	if !strings.Contains(snippet, `g.POST("/posts", posts.Create)`) {
		t.Fatalf("missing create route:\n%s", snippet)
	}
}
