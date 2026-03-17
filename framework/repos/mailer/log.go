package mailer

import "log/slog"

type LogMailClient struct {
	logger *slog.Logger
}

func NewLogMailClient(logger *slog.Logger) *LogMailClient {
	return &LogMailClient{logger: logger}
}

func (c *LogMailClient) Send(email *mail) error {
	logger := c.logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("email sent (log driver)",
		"from", email.from,
		"to", email.to,
		"subject", email.subject,
	)
	return nil
}
