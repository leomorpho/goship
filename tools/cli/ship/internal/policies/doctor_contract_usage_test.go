package policies

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDoctorChecks_ContractUsage(t *testing.T) {
	t.Run("bind without contracts type emits warning", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		path := filepath.Join(root, "app", "web", "controllers", "raw_bind.go")
		content := `package controllers

import "github.com/labstack/echo/v4"

type rawForm struct {
	Email string ` + "`form:\"email\"`" + `
}

type rawBindRoute struct{}

func (rawBindRoute) Post(ctx echo.Context) error {
	var form rawForm
	if err := ctx.Bind(&form); err != nil {
		return err
	}
	return nil
}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX027")
	})

	t.Run("bind with app/contracts type is allowed", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		contractsPath := filepath.Join(root, "app", "contracts")
		if err := os.MkdirAll(contractsPath, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(contractsPath, "request.go"), []byte("package contracts\ntype SignupForm struct{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		controllerPath := filepath.Join(root, "app", "web", "controllers", "contract_bind.go")
		controllerContent := `package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/contracts"
)

type contractRoute struct{}

func (contractRoute) Post(ctx echo.Context) error {
	form := contracts.SignupForm{}
	if err := ctx.Bind(&form); err != nil {
		return err
	}
	return nil
}
`
		if err := os.WriteFile(controllerPath, []byte(controllerContent), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX027" && issue.File == "app/web/controllers/contract_bind.go" {
				t.Fatalf("unexpected DX027 issue for contracts binding: %+v", issue)
			}
		}
	})

	t.Run("FormValue usage emits warning", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		path := filepath.Join(root, "app", "web", "controllers", "form_value.go")
		content := `package controllers

import "github.com/labstack/echo/v4"

type formValueRoute struct{}

func (formValueRoute) Post(ctx echo.Context) error {
	_ = ctx.FormValue("email")
	return nil
}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX027")
	})

	t.Run("contact controller uses app contracts request type", func(t *testing.T) {
		root := findRepoRoot(t)
		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX027" && issue.File == "app/web/controllers/contact.go" {
				t.Fatalf("unexpected DX027 issue for contact controller: %+v", issue)
			}
		}
	})

	t.Run("email subscribe controller uses app contracts request type", func(t *testing.T) {
		root := findRepoRoot(t)
		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX027" && issue.File == "app/web/controllers/email_subscribe.go" {
				t.Fatalf("unexpected DX027 issue for email subscribe controller: %+v", issue)
			}
		}
	})

	t.Run("managed hooks controller uses app contracts request types", func(t *testing.T) {
		root := findRepoRoot(t)
		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX027" && issue.File == "app/web/controllers/managed_hooks.go" {
				t.Fatalf("unexpected DX027 issue for managed hooks controller: %+v", issue)
			}
		}
	})

	t.Run("preferences controller uses app contracts request types", func(t *testing.T) {
		root := findRepoRoot(t)
		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX027" && issue.File == "app/web/controllers/preferences.go" {
				t.Fatalf("unexpected DX027 issue for preferences controller: %+v", issue)
			}
		}
	})
}
