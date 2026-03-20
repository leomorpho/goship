package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/leomorpho/goship/app/foundation"
)

type SendTestEmailCommand struct {
	Container *foundation.Container
}

func (c *SendTestEmailCommand) Name() string {
	return "send:test-email"
}

func (c *SendTestEmailCommand) Description() string {
	return "Send a basic test email to verify mailer configuration."
}

func (c *SendTestEmailCommand) Run(ctx context.Context, args []string) error {
	if c == nil || c.Container == nil || c.Container.Mail == nil {
		return errors.New("container mail client is not initialized")
	}

	to, subject, dryRun, err := parseSendTestEmailArgs(args)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Printf("[dry-run] would send test email to %s with subject %q\n", to, subject)
		return nil
	}

	if err := c.Container.Mail.
		Compose().
		To(to).
		Subject(subject).
		Body("This is a GoShip test email.").
		Send(ctx); err != nil {
		return err
	}

	fmt.Printf("sent test email to %s\n", to)
	return nil
}

func parseSendTestEmailArgs(args []string) (string, string, bool, error) {
	to := ""
	subject := "GoShip test email"
	dryRun := false

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--dry-run":
			dryRun = true
		case strings.HasPrefix(args[i], "--to="):
			to = strings.TrimSpace(strings.TrimPrefix(args[i], "--to="))
		case args[i] == "--to":
			if i+1 >= len(args) {
				return "", "", false, errors.New("missing value for --to")
			}
			i++
			to = strings.TrimSpace(args[i])
		case strings.HasPrefix(args[i], "--subject="):
			subject = strings.TrimSpace(strings.TrimPrefix(args[i], "--subject="))
		case args[i] == "--subject":
			if i+1 >= len(args) {
				return "", "", false, errors.New("missing value for --subject")
			}
			i++
			subject = strings.TrimSpace(args[i])
		default:
			return "", "", false, fmt.Errorf("unknown option: %s", args[i])
		}
	}

	if to == "" {
		return "", "", false, errors.New("usage: send:test-email --to <email> [--subject <text>] [--dry-run]")
	}

	return to, subject, dryRun, nil
}
