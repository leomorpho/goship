package mailer

import (
	"bytes"
	"context"
	"errors"

	"github.com/a-h/templ"

	"github.com/mikestefanello/pagoda/config"
)

type (
	// MailClient provides a client for sending email
	// This is purposely not completed because there are many different methods and services
	// for sending email, many of which are very different. Choose what works best for you
	// and populate the methods below
	MailClient struct {
		// config stores application configuration
		config *config.Config
		// MailSender provides the actual implementation to send emails
		MailSender MailClientInterface
	}

	LayoutComponent func(content templ.Component) templ.Component

	// mail represents an email to be sent
	mail struct {
		client    *MailClient
		from      string
		to        string
		subject   string
		body      string
		layout    LayoutComponent
		component templ.Component
	}

	MailClientInterface interface {
		Send(email *mail) error
	}
)

// NewMailClient creates a new MailClient
func NewMailClient(cfg *config.Config, sender MailClientInterface) (*MailClient, error) {
	return &MailClient{
		config:     cfg,
		MailSender: sender,
	}, nil
}

// Compose creates a new email
func (m *MailClient) Compose() *mail {
	return &mail{
		client: m,
		from:   m.config.Mail.FromAddress,
	}
}

// send attempts to send the email
func (m *MailClient) send(email *mail, ctx context.Context) error {
	switch {
	case email.to == "":
		return errors.New("email cannot be sent without a to address")
	case email.body == "" && email.component == nil:
		return errors.New("email cannot be sent without a body or component")
	}

	// Check if a component was supplied
	if email.component != nil {
		// Render the templates for the Email
		buf := &bytes.Buffer{}

		// If the email layout is set, that will be used to wrap the email component
		component := email.component
		if email.layout != nil {
			component = email.layout(component)
		}
		if err := component.Render(ctx, buf); err != nil {
			return err
		}

		email.body = buf.String()
	}

	// Delegate sending to the mailSender
	return m.MailSender.Send(email)
}

// From sets the email from address
func (m *mail) From(from string) *mail {
	m.from = from
	return m
}

// To sets the email address this email will be sent to
func (m *mail) To(to string) *mail {
	m.to = to
	return m
}

// Subject sets the subject line of the email
func (m *mail) Subject(subject string) *mail {
	m.subject = subject
	return m
}

// Body sets the body of the email
// This is not required and will be ignored if a template via Template()
func (m *mail) Body(body string) *mail {
	m.body = body
	return m
}

// Component sets the template component to be used to produce the body of the email
func (m *mail) Component(component templ.Component) *mail {
	m.component = component
	return m
}

// TemplateLayout sets the layout component that will wrap the template component specified when calling Component()
func (m *mail) TemplateLayout(layout LayoutComponent) *mail {
	m.layout = layout
	return m
}

// Send attempts to send the email
func (m *mail) Send(ctx context.Context) error {
	return m.client.send(m, ctx)
}
