package policies

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDoctorChecks_ContractUsage(t *testing.T) {
	t.Run("bind into untyped map emits warning", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		path := filepath.Join(root, "app", "web", "controllers", "raw_bind.go")
		content := `package controllers

import "github.com/labstack/echo/v4"

type rawBindRoute struct{}

func (rawBindRoute) Post(ctx echo.Context) error {
	form := map[string]any{}
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

	t.Run("bind with local typed request struct is allowed", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		controllerPath := filepath.Join(root, "app", "web", "controllers", "local_bind.go")
		controllerContent := `package controllers

import "github.com/labstack/echo/v4"

type localSignupRequest struct {
	Email string ` + "`form:\"email\" validate:\"required,email\"`" + `
}

type localRequestRoute struct{}

func (localRequestRoute) Post(ctx echo.Context) error {
	req := localSignupRequest{}
	if err := ctx.Bind(&req); err != nil {
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
			if issue.Code == "DX027" && issue.File == "app/web/controllers/local_bind.go" {
				t.Fatalf("unexpected DX027 issue for local typed request binding: %+v", issue)
			}
		}
	})

	t.Run("bind with module-owned contracts type is allowed", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		moduleContractsPath := filepath.Join(root, "modules", "payments", "contracts")
		if err := os.MkdirAll(moduleContractsPath, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(moduleContractsPath, "request.go"), []byte("package contracts\ntype CheckoutForm struct{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		controllerPath := filepath.Join(root, "app", "web", "controllers", "module_contract_bind.go")
		controllerContent := `package controllers

import (
	"github.com/labstack/echo/v4"
	paymentcontracts "example.com/root/modules/payments/contracts"
)

type moduleContractRoute struct{}

func (moduleContractRoute) Post(ctx echo.Context) error {
	form := paymentcontracts.CheckoutForm{}
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
			if issue.Code == "DX027" && issue.File == "app/web/controllers/module_contract_bind.go" {
				t.Fatalf("unexpected DX027 issue for module contracts binding: %+v", issue)
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
