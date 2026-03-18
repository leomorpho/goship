package mailer_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	emailviews "github.com/leomorpho/goship/app/views/emails/gen"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/framework/repos/mailer"
)

func TestRenderEmailRequiresComponent(t *testing.T) {
	_, _, err := mailer.RenderEmail(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when component is nil")
	}
}

func TestRenderEmailReturnsHTMLAndText(t *testing.T) {
	component := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<h1>Hello&nbsp;friend</h1><style>.x{color:red}</style><p>Reset your password</p>`)
		return err
	})

	html, text, err := mailer.RenderEmail(context.Background(), component)
	if err != nil {
		t.Fatalf("render email: %v", err)
	}
	if html == "" {
		t.Fatal("expected non-empty html output")
	}
	if text == "" {
		t.Fatal("expected non-empty text output")
	}
	if strings.Contains(text, "<") {
		t.Fatalf("text output should not contain html tags: %q", text)
	}
	if text != "Hello friend Reset your password" {
		t.Fatalf("text output = %q, want %q", text, "Hello friend Reset your password")
	}
}

func TestRenderEmailTemplatesReturnNonEmptyOutput(t *testing.T) {
	tplTests := []struct {
		name      string
		component templ.Component
	}{
		{name: "test", component: emailviews.TestEmail()},
		{name: "subscription_confirmation", component: emailviews.SubscriptionConfirmation(&ui.Page{})},
		{
			name: "registration_confirmation",
			component: emailviews.RegistrationConfirmation(&ui.Page{
				Data: viewmodels.EmailDefaultData{
					AppName:          "GoShip",
					SupportEmail:     "support@example.com",
					ConfirmationLink: "https://example.com/confirm",
				},
			}),
		},
		{
			name: "password_reset",
			component: emailviews.PasswordReset(&ui.Page{
				Data: viewmodels.EmailPasswordResetData{
					AppName:           "GoShip",
					SupportEmail:      "support@example.com",
					ProfileName:       "Test User",
					PasswordResetLink: "https://example.com/reset",
				},
			}),
		},
		{
			name: "email_update",
			component: emailviews.EmailUpdate(&ui.Page{
				Data: viewmodels.EmailUpdate{
					AppName:      "GoShip",
					SupportEmail: "support@example.com",
				},
			}),
		},
	}

	for _, tc := range tplTests {
		t.Run(tc.name, func(t *testing.T) {
			html, text, err := mailer.RenderEmail(context.Background(), tc.component)
			if err != nil {
				t.Fatalf("render email template: %v", err)
			}
			if strings.TrimSpace(html) == "" {
				t.Fatal("expected non-empty html output")
			}
			if strings.TrimSpace(text) == "" {
				t.Fatal("expected non-empty text output")
			}
		})
	}
}
