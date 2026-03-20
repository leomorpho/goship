package policies

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDoctorChecks_WarnsWhenAPIRouteRendersHTML(t *testing.T) {
	root := t.TempDir()
	writeDoctorFixture(t, root)

	router := `package goship

func registerExternalRoutes(e any) {
	v1 := e.Group("/api/v1") // ship:routes:api:v1:start
	v1.GET("/posts", posts.Index)
	// ship:routes:api:v1:end
}
`
	if err := os.WriteFile(filepath.Join(root, "app", "router.go"), []byte(router), 0o644); err != nil {
		t.Fatal(err)
	}

	controller := `package controllers

func (p *posts) Index(ctx Context) error {
	return ctx.HTML(200, "<html></html>")
}
`
	if err := os.WriteFile(filepath.Join(root, "app", "web", "controllers", "posts.go"), []byte(controller), 0o644); err != nil {
		t.Fatal(err)
	}

	issues := RunDoctorChecks(root)
	found := false
	for _, issue := range issues {
		if issue.Code == "DX026" {
			found = true
			if issue.Severity != "warning" {
				t.Fatalf("severity = %q, want warning", issue.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("issues = %+v, want DX026 warning", issues)
	}
}
