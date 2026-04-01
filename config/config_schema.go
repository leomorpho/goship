package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/leomorpho/goship/v2/config/runtimeconfig"
)

const (
	// TemplateExt stores the extension used for the template files
	TemplateExt = ".gohtml"

	// StaticDir stores the directory served as static assets for the example app.
	StaticDir = "static"

	// StaticPrefix stores the URL prefix used when serving static files
	StaticPrefix = "files"
)

type app string
type environment string
type dbmode string
type runtimeprofile string
type dbdriver string
type storagedriver string
type uiprovider string

const (
	// EnvLocal represents the local environment
	EnvLocal environment = "local"

	// EnvTest represents the test environment
	EnvTest environment = "test"

	// EnvDevelop represents the development environment
	EnvDevelop environment = "dev"

	// EnvStaging represents the staging environment
	EnvStaging environment = "staging"

	// EnvQA represents the qa environment
	EnvQA environment = "qa"

	// EnvProduction represents the production environment
	EnvProduction environment = "prod"

	// DBModeEmbedded represents an embedded DB being used as a storage backend
	DBModeEmbedded dbmode = "embedded"

	// DBModeStandalone represents a standalone DB as being used as a storage backend
	DBModeStandalone dbmode = "standalone"

	// RuntimeProfileServerDB is the default profile using an external DB server.
	RuntimeProfileServerDB runtimeprofile = "server-db"

	// RuntimeProfileSingleNode is the profile targeting single-node embedding.
	RuntimeProfileSingleNode runtimeprofile = "single-node"

	// RuntimeProfileDistributed is the profile targeting distributed processes.
	RuntimeProfileDistributed runtimeprofile = "distributed"

	// DBDriverPostgres uses an external Postgres server.
	DBDriverPostgres dbdriver = "postgres"

	// DBDriverSQLite uses an embedded SQLite database.
	DBDriverSQLite dbdriver = "sqlite"

	// StorageDriverLocal uses the local filesystem via Afero.
	StorageDriverLocal storagedriver = "local"

	// StorageDriverMinIO uses MinIO/S3 compatible storage.
	StorageDriverMinIO storagedriver = "minio"

	// UIProviderFranken is the default GoShip UI provider.
	UIProviderFranken uiprovider = "franken"

	// UIProviderDaisy selects the Daisy UI provider.
	UIProviderDaisy uiprovider = "daisy"

	// UIProviderBare selects the minimal bare UI provider.
	UIProviderBare uiprovider = "bare"
)

// SwitchEnvironment sets the environment variable used to dictate which environment the application is
// currently running in.
// This must be called prior to loading the configuration in order for it to take effect.
func SwitchEnvironment(env environment) {
	if err := os.Setenv("PAGODA_APP_ENVIRONMENT", string(env)); err != nil {
		panic(err)
	}
}

type (
	// Config stores complete configuration
	Config struct {
		HTTP        HTTPConfig
		App         AppConfig
		Log         LogConfig
		Security    SecurityConfig
		Runtime     RuntimeConfig
		Managed     ManagedConfig
		UI          UIConfig
		Processes   ProcessesConfig
		Adapters    AdaptersConfig
		Metrics     MetricsConfig
		I18n        I18nConfig
		Cache       CacheConfig
		Database    DatabaseConfig
		Mail        MailConfig
		Phone       PhoneConfig
		Recommender RecommenderConfig
		Storage     StorageConfig
		Backup      BackupConfig
		OAuth       OAuthConfig
		AI          AIConfig
	}

	LogConfig struct {
		Level  string `env:"PAGODA_LOG_LEVEL" env-default:"info"`
		Format string `env:"PAGODA_LOG_FORMAT" env-default:"text"`
	}

	SecurityConfig struct {
		Headers SecurityHeadersConfig
	}

	SecurityHeadersConfig struct {
		Enabled bool   `env:"PAGODA_SECURITY_HEADERS_ENABLED,SECURITY_HEADERS_ENABLED" env-default:"true"`
		HSTS    bool   `env:"PAGODA_SECURITY_HEADERS_HSTS,SECURITY_HEADERS_HSTS" env-default:"false"`
		CSP     string `env:"PAGODA_SECURITY_HEADERS_CSP,SECURITY_HEADERS_CSP"`
	}

	RuntimeConfig struct {
		Profile runtimeprofile `env:"PAGODA_RUNTIME_PROFILE"`
	}

	ManagedConfig struct {
		Enabled              bool                 `env:"PAGODA_MANAGED_MODE"`
		Authority            string               `env:"PAGODA_MANAGED_AUTHORITY"`
		OverridesJSON        string               `env:"PAGODA_MANAGED_OVERRIDES"`
		HooksSecret          string               `env:"PAGODA_MANAGED_HOOKS_SECRET"`
		HooksPreviousSecret  string               `env:"PAGODA_MANAGED_HOOKS_PREVIOUS_SECRET"`
		HooksMaxSkewSeconds  int                  `env:"PAGODA_MANAGED_HOOKS_MAX_SKEW_SECONDS" env-default:"300"`
		HooksNonceTTLSeconds int                  `env:"PAGODA_MANAGED_HOOKS_NONCE_TTL_SECONDS" env-default:"300"`
		RuntimeReport        runtimeconfig.Report `env:"-"`
	}

	UIConfig struct {
		Provider uiprovider `env:"PAGODA_UI_PROVIDER"`
	}

	ProcessesConfig struct {
		Web       bool `env:"PAGODA_PROCESSES_WEB"`
		Worker    bool `env:"PAGODA_PROCESSES_WORKER"`
		Scheduler bool `env:"PAGODA_PROCESSES_SCHEDULER"`
		CoLocated bool `env:"PAGODA_PROCESSES_COLOCATED"`
	}

	AdaptersConfig struct {
		DB     string `env:"PAGODA_ADAPTERS_DB"`
		Cache  string `env:"PAGODA_ADAPTERS_CACHE,PAGODA_CACHE_DRIVER"`
		Jobs   string `env:"PAGODA_ADAPTERS_JOBS,PAGODA_JOBS_DRIVER"`
		PubSub string `env:"PAGODA_ADAPTERS_PUBSUB"`
	}

	MetricsConfig struct {
		Enabled  bool   `env:"PAGODA_METRICS_ENABLED" env-default:"true"`
		Path     string `env:"PAGODA_METRICS_PATH" env-default:"/metrics"`
		Exporter string `env:"PAGODA_METRICS_EXPORTER" env-default:"prometheus"`
		Format   string `env:"PAGODA_METRICS_FORMAT" env-default:"prometheus-text"`
	}

	I18nConfig struct {
		Enabled         bool   `env:"PAGODA_I18N_ENABLED" env-default:"true"`
		DefaultLanguage string `env:"PAGODA_I18N_DEFAULT_LANGUAGE" env-default:"en"`
		StrictMode      string `env:"PAGODA_I18N_STRICT_MODE" env-default:"off"`
	}

	// HTTPConfig stores HTTP configuration
	HTTPConfig struct {
		Hostname     string        `env:"PAGODA_HTTP_HOSTNAME"`
		Port         uint16        `env:"PAGODA_HTTP_PORT"`
		Domain       string        `env:"PAGODA_HTTP_DOMAIN"`
		ReadTimeout  time.Duration `env:"PAGODA_HTTP_READTIMEOUT"`
		WriteTimeout time.Duration `env:"PAGODA_HTTP_WRITETIMEOUT"`
		IdleTimeout  time.Duration `env:"PAGODA_HTTP_IDLETIMEOUT"`
		SseKeepAlive time.Duration `env:"PAGODA_HTTP_SSEKEEPALIVE"`
		TLS          TLSConfig
	}

	TLSConfig struct {
		Enabled     bool   `env:"PAGODA_HTTP_TLS_ENABLED"`
		Certificate string `env:"PAGODA_HTTP_TLS_CERTIFICATE"`
		Key         string `env:"PAGODA_HTTP_TLS_KEY"`
	}

	// AppConfig stores application configuration
	AppConfig struct {
		Name                             app           `env:"PAGODA_APP_NAME"`
		SupportEmail                     string        `env:"PAGODA_APP_SUPPORTEMAIL"`
		Environment                      environment   `env:"PAGODA_APP_ENVIRONMENT"`
		EncryptionKey                    string        `env:"PAGODA_APP_ENCRYPTIONKEY"`
		Timeout                          time.Duration `env:"PAGODA_APP_TIMEOUT"`
		PasswordToken                    PasswordTokenConfig
		EmailVerificationTokenExpiration time.Duration `env:"PAGODA_APP_EMAILVERIFICATIONTOKENEXPIRATION"`
		OperationalConstants             OperationalConstants
		PageSize                         int    `env:"PAGODA_APP_PAGESIZE"`
		VapidPublicKey                   string `env:"PAGODA_APP_VAPIDPUBLICKEY"`
		VapidPrivateKey                  string `env:"PAGODA_APP_VAPIDPRIVATEKEY"`
		SentryDsn                        string `env:"PAGODA_APP_SENTRYDSN,SENTRY_DSN"`
		TestSentryUrl                    string `env:"PAGODA_APP_TESTSENTRYURL"`
		PublicStripeKey                  string `env:"PAGODA_APP_PUBLICSTRIPEKEY"`
		PrivateStripeKey                 string `env:"PAGODA_APP_PRIVATESTRIPEKEY"`
		StripeWebhookSecret              string `env:"PAGODA_APP_STRIPEWEBHOOKSECRET"`
		StripeWebhookPath                string `env:"PAGODA_APP_STRIPEWEBHOOKPATH"`
		AppEncryptionKey                 string `env:"PAGODA_APP_APPENCRYPTIONKEY"`
		FirebaseBase64AccessKeys         string `env:"PAGODA_APP_FIREBASEBASE64ACCESSKEYS"`
		FirebaseJSONAccessKeys           []byte
	}

	PasswordTokenConfig struct {
		Expiration time.Duration `env:"PAGODA_APP_PASSWORDTOKEN_EXPIRATION,PAGODA_APP_PASSWORDTOKENEXPIRATION"`
		Length     int           `env:"PAGODA_APP_PASSWORDTOKEN_LENGTH,PAGODA_APP_PASSWORDTOKENLENGTH"`
	}

	OperationalConstants struct {
		NewsletterSignupEnabled                           bool          `env:"PAGODA_APP_OPERATIONALCONSTANTS_NEWSLETTERSIGNUPENABLED,PAGODA_APP_OPERATIONALCONSTANTS_NEWSLETTER_SIGNUP_ENABLED"`
		UserSignupEnabled                                 bool          `env:"PAGODA_APP_OPERATIONALCONSTANTS_USERSIGNUPENABLED,PAGODA_APP_OPERATIONALCONSTANTS_USERSIGNUPENABLED"`
		UserSignupEnabledOnLandingPage                    bool          `env:"PAGODA_APP_OPERATIONALCONSTANTS_USERSIGNUPENABLEDONLANDINGPAGE"`
		QuestionInteractionValidLifetimeInDays            int           `env:"PAGODA_APP_OPERATIONALCONSTANTS_QUESTIONINTERACTIONVALIDLIFETIMEINDAYS"`
		NumMinSharedAnswersForPrivateMessages             int           `env:"PAGODA_APP_OPERATIONALCONSTANTS_NUMMINSHAREDANSWERSFORPRIVATEMESSAGES"`
		NotifEmojiDebounceTime                            time.Duration `env:"PAGODA_APP_OPERATIONALCONSTANTS_NOTIFEMOJIDEBOUNCETIME"`
		NotifyNewAnswerFromUnansweredQuestionDebounceTime time.Duration `env:"PAGODA_APP_OPERATIONALCONSTANTS_NOTIFYNEWANSWERFROMUNANSWEREDQUESTIONDEBOUNCETIME"`
		MinAnswerLen                                      int           `env:"PAGODA_APP_OPERATIONALCONSTANTS_MINANSWERLEN"`
		PaymentsEnabled                                   bool          `env:"PAGODA_APP_OPERATIONALCONSTANTS_PAYMENTSENABLED"`
		ProTrialTimespanInDays                            int           `env:"PAGODA_APP_OPERATIONALCONSTANTS_PROTRIALTIMESPANINDAYS"`
		ProductProCode                                    string        `env:"PAGODA_APP_OPERATIONALCONSTANTS_PRODUCTPROCODE"`
		ProductProPrice                                   float32       `env:"PAGODA_APP_OPERATIONALCONSTANTS_PRODUCTPROPRICE"`
		PaymentFailedGracePeriodInDays                    int           `env:"PAGODA_APP_OPERATIONALCONSTANTS_PAYMENTFAILEDGRACEPERIODINDAYS"`
		DeleteStaleNotificationAfterDays                  int           `env:"PAGODA_APP_OPERATIONALCONSTANTS_DELETESTALENOTIFICATIONAFTERDAYS"`
		MaxLikedQuestionHistoryFreePlan                   int           `env:"PAGODA_APP_OPERATIONALCONSTANTS_MAXLIKEDQUESTIONHISTORYFREEPLAN"`
	}

	// CacheConfig stores the cache configuration
	CacheConfig struct {
		Hostname     string                `env:"PAGODA_CACHE_HOSTNAME"`
		Port         uint16                `env:"PAGODA_CACHE_PORT"`
		Password     string                `env:"PAGODA_CACHE_PASSWORD"`
		Database     int                   `env:"PAGODA_CACHE_DATABASE"`
		TestDatabase int                   `env:"PAGODA_CACHE_TESTDATABASE"`
		Expiration   CacheExpirationConfig `env:"-"`
	}

	CacheExpirationConfig struct {
		StaticFile time.Duration `env:"PAGODA_CACHE_EXPIRATIONSTATICFILE,PAGODA_CACHE_EXPIRATION_STATICFILE"`
		Page       time.Duration `env:"PAGODA_CACHE_EXPIRATIONPAGE,PAGODA_CACHE_EXPIRATION_PAGE"`
	}

	// DatabaseConfig stores the database configuration
	DatabaseConfig struct {
		Driver                 dbdriver `env:"PAGODA_DATABASE_DRIVER,PAGODA_DB_DRIVER"`
		Path                   string   `env:"PAGODA_DATABASE_PATH,PAGODA_DB_PATH"`
		DbMode                 dbmode   `env:"PAGODA_DATABASE_DBMODE"`
		EmbeddedDriver         string   `env:"PAGODA_DATABASE_EMBEDDEDDRIVER"`
		EmbeddedConnection     string   `env:"PAGODA_DATABASE_EMBEDDEDCONNECTION"`
		EmbeddedTestConnection string   `env:"PAGODA_DATABASE_EMBEDDEDTESTCONNECTION"`
		Hostname               string   `env:"PAGODA_DATABASE_HOSTNAME"`
		Port                   uint16   `env:"PAGODA_DATABASE_PORT"`
		User                   string   `env:"PAGODA_DATABASE_USER"`
		Password               string   `env:"PAGODA_DATABASE_PASSWORD"`
		DatabaseNameLocal      string   `env:"PAGODA_DATABASE_DATABASENAMELOCAL"`
		DatabaseNameProd       string   `env:"PAGODA_DATABASE_DATABASENAMEPROD"`
		TestDatabase           string   `env:"PAGODA_DATABASE_TESTDATABASE"`
		SslCertPath            string   `env:"PAGODA_DATABASE_SSLCERTPATH"`
		SslMode                string   `env:"PAGODA_DATABASE_SSLMODE"`
	}

	// MailConfig stores the mail configuration
	MailConfig struct {
		Driver      string `env:"PAGODA_MAIL_DRIVER,MAIL_DRIVER" env-default:"log"`
		FromName    string `env:"PAGODA_MAIL_FROM_NAME,MAIL_FROM_NAME" env-default:"GoShip App"`
		FromAddress string `env:"PAGODA_MAIL_FROMADDRESS,MAIL_FROM_ADDRESS"`
		Hostname    string `env:"PAGODA_MAIL_HOSTNAME,MAIL_HOSTNAME"`
		HttpPort    uint16 `env:"PAGODA_MAIL_HTTPPORT,MAIL_HTTPPORT"`
		SmtpPort    uint16 `env:"PAGODA_MAIL_SMTPPORT,MAIL_SMTP_PORT"`
		User        string `env:"PAGODA_MAIL_USER,MAIL_USER"`
		Password    string `env:"PAGODA_MAIL_PASSWORD,MAIL_PASSWORD"`
		SMTP        struct {
			Host string `env:"PAGODA_MAIL_SMTP_HOST,MAIL_SMTP_HOST"`
			Port int    `env:"PAGODA_MAIL_SMTP_PORT,MAIL_SMTP_PORT" env-default:"587"`
			User string `env:"PAGODA_MAIL_SMTP_USER,MAIL_SMTP_USER"`
			Pass string `env:"PAGODA_MAIL_SMTP_PASS,MAIL_SMTP_PASS"`
			TLS  bool   `env:"PAGODA_MAIL_SMTP_TLS,MAIL_SMTP_TLS" env-default:"true"`
		}
		Resend struct {
			APIKey string `env:"PAGODA_MAIL_RESEND_API_KEY,MAIL_RESEND_API_KEY"`
		}
		ResendAPIKey string `env:"PAGODA_MAIL_RESENDAPIKEY,MAIL_RESENDAPIKEY"`
	}

	PhoneConfig struct {
		SenderID                        string `env:"PAGODA_PHONE_SENDERID"`
		Region                          string `env:"PAGODA_PHONE_REGION"`
		ValidationCodeExpirationMinutes int    `env:"PAGODA_PHONE_VALIDATIONCODEEXPIRATIONMINUTES"`
	}

	RecommenderConfig struct {
		NumProfilesToMatchAtOnce int `env:"PAGODA_RECOMMENDER_NUMPROFILESTOMATCHATONCE"`
	}

	StorageConfig struct {
		Driver                    storagedriver `env:"PAGODA_STORAGE_DRIVER"`
		LocalStoragePath          string        `env:"PAGODA_STORAGE_LOCALSTORAGEPATH" env-default:"./uploads"`
		AppBucketName             string        `env:"PAGODA_STORAGE_APPBUCKETNAME"`
		StaticFilesBucketName     string        `env:"PAGODA_STORAGE_STATICFILESBUCKETNAME"`
		S3Endpoint                string        `env:"PAGODA_STORAGE_S3ENDPOINT"`
		S3AccessKey               string        `env:"PAGODA_STORAGE_S3ACCESSKEY"`
		S3SecretKey               string        `env:"PAGODA_STORAGE_S3SECRETKEY"`
		S3UseSSL                  bool          `env:"PAGODA_STORAGE_S3USESSL"`
		ProfilePhotoMaxFileSizeMB int64         `env:"PAGODA_STORAGE_PROFILEPHOTOMAXFILESIZEMB"`
		PhotosMaxFileSizeMB       int64         `env:"PAGODA_STORAGE_PHOTOSMAXFILESIZEMB"`
	}

	BackupConfig struct {
		Driver        string `env:"PAGODA_BACKUP_DRIVER" env-default:"sqlite-file"`
		SchemaVersion string `env:"PAGODA_BACKUP_SCHEMA_VERSION" env-default:"v1"`
		SQLitePath    string `env:"PAGODA_BACKUP_SQLITE_PATH"`
		S3            BackupS3Config
	}

	BackupS3Config struct {
		Enabled   bool   `env:"PAGODA_BACKUP_S3_ENABLED"`
		Endpoint  string `env:"PAGODA_BACKUP_S3_ENDPOINT"`
		Region    string `env:"PAGODA_BACKUP_S3_REGION"`
		Bucket    string `env:"PAGODA_BACKUP_S3_BUCKET"`
		Prefix    string `env:"PAGODA_BACKUP_S3_PREFIX"`
		AccessKey string `env:"PAGODA_BACKUP_S3_ACCESSKEY"`
		SecretKey string `env:"PAGODA_BACKUP_S3_SECRETKEY"`
		UseSSL    bool   `env:"PAGODA_BACKUP_S3_USESSL" env-default:"true"`
	}

	OAuthConfig struct {
		GitHub  OAuthGitHubConfig
		Google  OAuthGoogleConfig
		Discord OAuthDiscordConfig
	}

	OAuthGitHubConfig struct {
		ClientID     string `env:"OAUTH_GITHUB_CLIENT_ID"`
		ClientSecret string `env:"OAUTH_GITHUB_CLIENT_SECRET"`
	}

	OAuthGoogleConfig struct {
		ClientID     string `env:"OAUTH_GOOGLE_CLIENT_ID"`
		ClientSecret string `env:"OAUTH_GOOGLE_CLIENT_SECRET"`
	}

	OAuthDiscordConfig struct {
		ClientID     string `env:"OAUTH_DISCORD_CLIENT_ID"`
		ClientSecret string `env:"OAUTH_DISCORD_CLIENT_SECRET"`
	}

	AIConfig struct {
		Driver     string `env:"AI_DRIVER" env-default:"anthropic"`
		Anthropic  AIAnthropicConfig
		OpenAI     AIOpenAIConfig
		OpenRouter AIOpenRouterConfig
	}

	AIAnthropicConfig struct {
		APIKey       string `env:"ANTHROPIC_API_KEY"`
		DefaultModel string `env:"ANTHROPIC_DEFAULT_MODEL" env-default:"claude-haiku-4-5-20251001"`
	}

	AIOpenAIConfig struct {
		APIKey       string `env:"OPENAI_API_KEY"`
		DefaultModel string `env:"OPENAI_DEFAULT_MODEL" env-default:"gpt-4o-mini"`
	}

	AIOpenRouterConfig struct {
		APIKey       string `env:"OPENROUTER_API_KEY"`
		DefaultModel string `env:"OPENROUTER_DEFAULT_MODEL" env-default:"anthropic/claude-haiku-4-5-20251001"`
		SiteURL      string `env:"OPENROUTER_SITE_URL"`
		SiteName     string `env:"OPENROUTER_SITE_NAME"`
	}
)

type managedKeySpec struct {
	envVars []string
	get     func(Config) string
	set     func(*Config, string) error
}

var managedOverrideSpecs = map[string]managedKeySpec{
	"runtime.profile": {
		envVars: []string{"PAGODA_RUNTIME_PROFILE"},
		get: func(cfg Config) string {
			return string(cfg.Runtime.Profile)
		},
		set: func(cfg *Config, raw string) error {
			profile := normalizeRuntimeProfile(raw)
			if profile == "" {
				return fmt.Errorf("unsupported runtime profile %q", raw)
			}
			cfg.Runtime.Profile = profile
			return nil
		},
	},
	"processes.web": {
		envVars: []string{"PAGODA_PROCESSES_WEB"},
		get: func(cfg Config) string {
			return strconv.FormatBool(cfg.Processes.Web)
		},
		set: func(cfg *Config, raw string) error {
			return setManagedBool(raw, &cfg.Processes.Web, "processes.web")
		},
	},
	"processes.worker": {
		envVars: []string{"PAGODA_PROCESSES_WORKER"},
		get: func(cfg Config) string {
			return strconv.FormatBool(cfg.Processes.Worker)
		},
		set: func(cfg *Config, raw string) error {
			return setManagedBool(raw, &cfg.Processes.Worker, "processes.worker")
		},
	},
	"processes.scheduler": {
		envVars: []string{"PAGODA_PROCESSES_SCHEDULER"},
		get: func(cfg Config) string {
			return strconv.FormatBool(cfg.Processes.Scheduler)
		},
		set: func(cfg *Config, raw string) error {
			return setManagedBool(raw, &cfg.Processes.Scheduler, "processes.scheduler")
		},
	},
	"processes.colocated": {
		envVars: []string{"PAGODA_PROCESSES_COLOCATED"},
		get: func(cfg Config) string {
			return strconv.FormatBool(cfg.Processes.CoLocated)
		},
		set: func(cfg *Config, raw string) error {
			return setManagedBool(raw, &cfg.Processes.CoLocated, "processes.colocated")
		},
	},
	"adapters.db": {
		envVars: []string{"PAGODA_ADAPTERS_DB"},
		get: func(cfg Config) string {
			return strings.TrimSpace(cfg.Adapters.DB)
		},
		set: func(cfg *Config, raw string) error {
			return setManagedString(raw, &cfg.Adapters.DB, "adapters.db")
		},
	},
	"adapters.cache": {
		envVars: []string{"PAGODA_ADAPTERS_CACHE", "PAGODA_CACHE_DRIVER"},
		get: func(cfg Config) string {
			return strings.TrimSpace(cfg.Adapters.Cache)
		},
		set: func(cfg *Config, raw string) error {
			return setManagedString(raw, &cfg.Adapters.Cache, "adapters.cache")
		},
	},
	"adapters.jobs": {
		envVars: []string{"PAGODA_ADAPTERS_JOBS", "PAGODA_JOBS_DRIVER"},
		get: func(cfg Config) string {
			return strings.TrimSpace(cfg.Adapters.Jobs)
		},
		set: func(cfg *Config, raw string) error {
			return setManagedString(raw, &cfg.Adapters.Jobs, "adapters.jobs")
		},
	},
	"adapters.pubsub": {
		envVars: []string{"PAGODA_ADAPTERS_PUBSUB"},
		get: func(cfg Config) string {
			return strings.TrimSpace(cfg.Adapters.PubSub)
		},
		set: func(cfg *Config, raw string) error {
			return setManagedString(raw, &cfg.Adapters.PubSub, "adapters.pubsub")
		},
	},
	"metrics.enabled": {
		envVars: []string{"PAGODA_METRICS_ENABLED"},
		get: func(cfg Config) string {
			return strconv.FormatBool(cfg.Metrics.Enabled)
		},
		set: func(cfg *Config, raw string) error {
			return setManagedBool(raw, &cfg.Metrics.Enabled, "metrics.enabled")
		},
	},
	"database.driver": {
		envVars: []string{"PAGODA_DATABASE_DRIVER", "PAGODA_DB_DRIVER"},
		get: func(cfg Config) string {
			return string(cfg.Database.Driver)
		},
		set: func(cfg *Config, raw string) error {
			driver := normalizeDBDriver(raw)
			if driver == "" {
				return fmt.Errorf("unsupported database driver %q", raw)
			}
			cfg.Database.Driver = dbdriver(driver)
			return nil
		},
	},
	"database.path": {
		envVars: []string{"PAGODA_DATABASE_PATH", "PAGODA_DB_PATH"},
		get: func(cfg Config) string {
			return strings.TrimSpace(cfg.Database.Path)
		},
		set: func(cfg *Config, raw string) error {
			return setManagedString(raw, &cfg.Database.Path, "database.path")
		},
	},
	"storage.driver": {
		envVars: []string{"PAGODA_STORAGE_DRIVER"},
		get: func(cfg Config) string {
			return string(cfg.Storage.Driver)
		},
		set: func(cfg *Config, raw string) error {
			driver := normalizeStorageDriver(raw)
			if driver == "" {
				return fmt.Errorf("unsupported storage driver %q", raw)
			}
			cfg.Storage.Driver = driver
			return nil
		},
	},
}
