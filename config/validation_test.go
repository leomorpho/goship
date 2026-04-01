package config

import "testing"

func TestValidateConfigSemantics_FlagsBrokenDrivers(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Adapters.DB = "postgres"
	cfg.Database.Driver = DBDriverPostgres
	cfg.Database.DatabaseNameProd = ""
	cfg.Mail.Driver = "smtp"
	cfg.Mail.FromAddress = ""
	cfg.Mail.SMTP.Host = ""
	cfg.Mail.SMTP.Port = 0
	cfg.Storage.Driver = StorageDriverMinIO
	cfg.Storage.S3AccessKey = "0072..."
	cfg.Storage.S3SecretKey = "K001..."
	cfg.Backup.S3.Enabled = true
	cfg.Backup.S3.AccessKey = "0072..."
	cfg.Backup.S3.SecretKey = "K001..."
	cfg.App.OperationalConstants.PaymentsEnabled = true

	issues := ValidateConfigSemantics(cfg)
	if len(issues) == 0 {
		t.Fatal("expected semantic validation issues, got none")
	}

	wantFields := map[string]bool{
		"database.database_name_prod": false,
		"mail.from_address":           false,
		"mail.smtp.host":              false,
		"storage.s3_access_key":       false,
		"backup.s3.access_key":        false,
		"app.public_stripe_key":       false,
	}
	for _, issue := range issues {
		if _, ok := wantFields[issue.Field]; ok {
			wantFields[issue.Field] = true
		}
	}
	for field, found := range wantFields {
		if !found {
			t.Fatalf("expected issue for %s; got %+v", field, issues)
		}
	}
}
