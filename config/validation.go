package config

import (
	"fmt"
	"strings"
)

type ValidationIssue struct {
	Field   string
	Message string
}

func (i ValidationIssue) Error() string {
	if strings.TrimSpace(i.Field) == "" {
		return strings.TrimSpace(i.Message)
	}
	return fmt.Sprintf("%s: %s", i.Field, strings.TrimSpace(i.Message))
}

func ValidateConfigSemantics(cfg Config) []ValidationIssue {
	issues := make([]ValidationIssue, 0, 8)

	mailDriver := strings.ToLower(strings.TrimSpace(cfg.Mail.Driver))
	switch mailDriver {
	case "", "log", "mailpit":
	case "smtp":
		issues = appendMissingIssue(issues, "mail.from_address", cfg.Mail.FromAddress, "must be set when mail driver is smtp")
		issues = appendMissingIssue(issues, "mail.smtp.host", cfg.Mail.SMTP.Host, "must be set when mail driver is smtp")
		if cfg.Mail.SMTP.Port <= 0 {
			issues = append(issues, ValidationIssue{
				Field:   "mail.smtp.port",
				Message: "must be greater than zero when mail driver is smtp",
			})
		}
	case "resend":
		issues = appendMissingIssue(issues, "mail.from_address", cfg.Mail.FromAddress, "must be set when mail driver is resend")
		resendKey := firstNonEmpty(cfg.Mail.Resend.APIKey, cfg.Mail.ResendAPIKey)
		issues = appendPlaceholderIssue(issues, "mail.resend.api_key", resendKey, "must be set when mail driver is resend")
	default:
		issues = append(issues, ValidationIssue{
			Field:   "mail.driver",
			Message: fmt.Sprintf("unsupported mail driver %q", cfg.Mail.Driver),
		})
	}

	if strings.EqualFold(string(cfg.Storage.Driver), string(StorageDriverMinIO)) {
		issues = appendMissingIssue(issues, "storage.app_bucket_name", cfg.Storage.AppBucketName, "must be set when storage driver is minio")
		issues = appendMissingIssue(issues, "storage.static_files_bucket_name", cfg.Storage.StaticFilesBucketName, "must be set when storage driver is minio")
		issues = appendMissingIssue(issues, "storage.s3_endpoint", cfg.Storage.S3Endpoint, "must be set when storage driver is minio")
		issues = appendPlaceholderIssue(issues, "storage.s3_access_key", cfg.Storage.S3AccessKey, "must be set when storage driver is minio")
		issues = appendPlaceholderIssue(issues, "storage.s3_secret_key", cfg.Storage.S3SecretKey, "must be set when storage driver is minio")
	}

	if usesPostgres(cfg) {
		issues = appendMissingIssue(issues, "database.hostname", cfg.Database.Hostname, "must be set when using postgres")
		issues = appendMissingIssue(issues, "database.user", cfg.Database.User, "must be set when using postgres")
		issues = appendMissingIssue(issues, "database.password", cfg.Database.Password, "must be set when using postgres")
		issues = appendMissingIssue(issues, "database.database_name_prod", cfg.Database.DatabaseNameProd, "must be set when using postgres")
		if cfg.Database.Port == 0 {
			issues = append(issues, ValidationIssue{
				Field:   "database.port",
				Message: "must be greater than zero when using postgres",
			})
		}
	}

	if cfg.Backup.S3.Enabled || strings.EqualFold(strings.TrimSpace(cfg.Backup.Driver), "s3") {
		issues = appendMissingIssue(issues, "backup.s3.endpoint", cfg.Backup.S3.Endpoint, "must be set when S3 backups are enabled")
		issues = appendMissingIssue(issues, "backup.s3.region", cfg.Backup.S3.Region, "must be set when S3 backups are enabled")
		issues = appendMissingIssue(issues, "backup.s3.bucket", cfg.Backup.S3.Bucket, "must be set when S3 backups are enabled")
		issues = appendPlaceholderIssue(issues, "backup.s3.access_key", cfg.Backup.S3.AccessKey, "must be set when S3 backups are enabled")
		issues = appendPlaceholderIssue(issues, "backup.s3.secret_key", cfg.Backup.S3.SecretKey, "must be set when S3 backups are enabled")
	}

	if cfg.App.OperationalConstants.PaymentsEnabled {
		issues = appendPlaceholderIssue(issues, "app.public_stripe_key", cfg.App.PublicStripeKey, "must be replaced before enabling payments")
		issues = appendPlaceholderIssue(issues, "app.private_stripe_key", cfg.App.PrivateStripeKey, "must be replaced before enabling payments")
		issues = appendPlaceholderIssue(issues, "app.stripe_webhook_secret", cfg.App.StripeWebhookSecret, "must be replaced before enabling payments")
		issues = appendPlaceholderIssue(issues, "app.product_pro_code", cfg.App.OperationalConstants.ProductProCode, "must be replaced before enabling payments")
	}

	if cfg.Managed.Enabled {
		issues = appendMissingIssue(issues, "managed.authority", cfg.Managed.Authority, "must be set when managed mode is enabled")
		issues = appendMissingIssue(issues, "managed.hooks_secret", cfg.Managed.HooksSecret, "must be set when managed mode is enabled")
	}

	return issues
}

func usesPostgres(cfg Config) bool {
	if strings.EqualFold(string(cfg.Database.Driver), string(DBDriverPostgres)) {
		return true
	}
	if strings.EqualFold(cfg.Adapters.DB, string(DBDriverPostgres)) {
		return true
	}
	return strings.EqualFold(string(cfg.Database.DbMode), string(DBModeStandalone))
}

func appendMissingIssue(issues []ValidationIssue, field, value, msg string) []ValidationIssue {
	if strings.TrimSpace(value) != "" {
		return issues
	}
	return append(issues, ValidationIssue{
		Field:   field,
		Message: msg,
	})
}

func appendPlaceholderIssue(issues []ValidationIssue, field, value, msg string) []ValidationIssue {
	if !isPlaceholderValue(value) {
		return issues
	}
	return append(issues, ValidationIssue{
		Field:   field,
		Message: msg,
	})
}

func isPlaceholderValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	placeholders := []string{
		"...",
		"changeme",
		"replace-me",
		"replace_this",
		"placeholder",
	}
	for _, placeholder := range placeholders {
		if strings.Contains(lower, placeholder) {
			return true
		}
	}
	prefixes := []string{
		"pk_...",
		"sk_...",
		"whsec_...",
		"price_...",
		"0072...",
		"k001...",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
