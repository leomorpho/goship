package mailer

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/smtp"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/resend/resend-go/v2"
)

func TestLogMailClientSend(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&out, nil))

	cfg := &config.Config{}
	cfg.Mail.FromAddress = "noreply@example.com"
	client, err := NewMailClient(cfg, NewLogMailClient(logger))
	if err != nil {
		t.Fatalf("NewMailClient() error = %v", err)
	}

	err = client.Compose().
		To("user@example.com").
		Subject("Welcome").
		Body("hello").
		Send(context.Background())
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	logOutput := out.String()
	if !strings.Contains(logOutput, "email sent (log driver)") {
		t.Fatalf("expected log output to include marker, got %q", logOutput)
	}
}

func TestSMTPMailClientSendUsesConfiguredTransport(t *testing.T) {
	original := smtpSendMail
	defer func() { smtpSendMail = original }()

	var gotAddr string
	var gotFrom string
	var gotTo []string
	var gotBody string
	smtpSendMail = func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		gotAddr = addr
		gotFrom = from
		gotTo = append([]string(nil), to...)
		gotBody = string(msg)
		return nil
	}

	cfg := &config.Config{}
	cfg.Mail.FromAddress = "noreply@example.com"
	client, err := NewMailClient(cfg, NewSMTPMailClient("localhost", 1025))
	if err != nil {
		t.Fatalf("NewMailClient() error = %v", err)
	}

	err = client.Compose().
		To("user@example.com").
		Subject("Welcome").
		Body("hello").
		Send(context.Background())
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotAddr != "localhost:1025" {
		t.Fatalf("unexpected smtp addr: %q", gotAddr)
	}
	if gotFrom != "noreply@example.com" {
		t.Fatalf("unexpected from: %q", gotFrom)
	}
	if len(gotTo) != 1 || gotTo[0] != "user@example.com" {
		t.Fatalf("unexpected recipients: %#v", gotTo)
	}
	if !strings.Contains(gotBody, "Subject: Welcome") {
		t.Fatalf("unexpected smtp payload: %q", gotBody)
	}
}

type fakeResendSender struct {
	req *resend.SendEmailRequest
	err error
}

func (f *fakeResendSender) SendWithContext(_ context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
	f.req = params
	if f.err != nil {
		return nil, f.err
	}
	return &resend.SendEmailResponse{Id: "id-1"}, nil
}

func TestResendMailClientSend(t *testing.T) {
	cfg := &config.Config{}
	cfg.Mail.FromAddress = "noreply@example.com"
	fake := &fakeResendSender{}
	client, err := NewMailClient(cfg, NewResendMailClientWithSender(fake))
	if err != nil {
		t.Fatalf("NewMailClient() error = %v", err)
	}

	err = client.Compose().
		To("user@example.com").
		Subject("Welcome").
		Body("hello").
		Send(context.Background())
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if fake.req == nil {
		t.Fatal("expected resend request")
	}
	if fake.req.Subject != "Welcome" {
		t.Fatalf("unexpected subject: %q", fake.req.Subject)
	}
	if len(fake.req.To) != 1 || fake.req.To[0] != "user@example.com" {
		t.Fatalf("unexpected recipient: %#v", fake.req.To)
	}
}

func TestResendMailClientSendError(t *testing.T) {
	cfg := &config.Config{}
	cfg.Mail.FromAddress = "noreply@example.com"
	fake := &fakeResendSender{err: errors.New("network down")}
	client, err := NewMailClient(cfg, NewResendMailClientWithSender(fake))
	if err != nil {
		t.Fatalf("NewMailClient() error = %v", err)
	}

	err = client.Compose().
		To("user@example.com").
		Subject("Welcome").
		Body("hello").
		Send(context.Background())
	if err == nil {
		t.Fatal("expected send error")
	}
	if !strings.Contains(err.Error(), "network down") {
		t.Fatalf("unexpected error: %v", err)
	}
}
