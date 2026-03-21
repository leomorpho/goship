package generators

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeMailerContract_RedSpec(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))

	makeHelp, err := os.ReadFile(filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "help.go"))
	if err != nil {
		t.Fatal(err)
	}
	cliDispatch, err := os.ReadFile(filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go"))
	if err != nil {
		t.Fatal(err)
	}
	cliRef, err := os.ReadFile(filepath.Join(root, "docs", "reference", "01-cli.md"))
	if err != nil {
		t.Fatal(err)
	}
	mailPreview, err := os.ReadFile(filepath.Join(root, "framework", "web", "controllers", "mail_preview.go"))
	if err != nil {
		t.Fatal(err)
	}

	for _, required := range []string{
		"ship make:mailer <Name>",
		"Generate a mailer scaffold",
	} {
		if !strings.Contains(string(makeHelp), required) {
			t.Fatalf("make help should advertise %q for the make:mailer generator", required)
		}
		if !strings.Contains(string(cliRef), required) {
			t.Fatalf("cli reference should advertise %q for the make:mailer generator", required)
		}
	}

	for _, required := range []string{
		`case "mailer":`,
		"runMakeMailer(",
	} {
		if !strings.Contains(string(cliDispatch), required) {
			t.Fatalf("cli dispatch should include %q for ship make:mailer", required)
		}
	}

	for _, required := range []string{
		"/dev/mail/",
		"Email previews",
		"templ.Component",
	} {
		if !strings.Contains(string(mailPreview), required) {
			t.Fatalf("mailer contract should align with preview surface token %q", required)
		}
	}
}
