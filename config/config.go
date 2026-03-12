package config

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/leomorpho/goship/framework/runtimeconfig"
)

const (
	// TemplateExt stores the extension used for the template files
	TemplateExt = ".gohtml"

	// StaticDir stores the directory served as static assets for the example app.
	StaticDir = "app/static"

	// StaticPrefix stores the URL prefix used when serving static files
	StaticPrefix = "files"
)

type app string
type environment string
type dbmode string
type runtimeprofile string
type dbdriver string
type storagedriver string

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
		Processes   ProcessesConfig
		Adapters    AdaptersConfig
		Cache       CacheConfig
		Database    DatabaseConfig
		Mail        MailConfig
		Phone       PhoneConfig
		Recommender RecommenderConfig
		Storage     StorageConfig
	}

	LogConfig struct {
		Level  string `env:"PAGODA_LOG_LEVEL" env-default:"info"`
		Format string `env:"PAGODA_LOG_FORMAT" env-default:"text"`
	}

	SecurityConfig struct {
		Headers SecurityHeadersConfig
	}

	SecurityHeadersConfig struct {
		Enabled bool   `env:"PAGODA_SECURITY_HEADERS_ENABLED" env-default:"true"`
		HSTS    bool   `env:"PAGODA_SECURITY_HEADERS_HSTS" env-default:"false"`
		CSP     string `env:"PAGODA_SECURITY_HEADERS_CSP"`
	}

	RuntimeConfig struct {
		Profile runtimeprofile `env:"PAGODA_RUNTIME_PROFILE"`
	}

	ManagedConfig struct {
		Enabled       bool                 `env:"PAGODA_MANAGED_MODE"`
		Authority     string               `env:"PAGODA_MANAGED_AUTHORITY"`
		OverridesJSON string               `env:"PAGODA_MANAGED_OVERRIDES"`
		RuntimeReport runtimeconfig.Report `env:"-"`
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
		SentryDsn                        string `env:"PAGODA_APP_SENTRYDSN"`
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
		Driver      string `env:"PAGODA_MAIL_DRIVER" env-default:"log"`
		FromName    string `env:"PAGODA_MAIL_FROM_NAME" env-default:"GoShip App"`
		FromAddress string `env:"PAGODA_MAIL_FROMADDRESS"`
		Hostname    string `env:"PAGODA_MAIL_HOSTNAME"`
		HttpPort    uint16 `env:"PAGODA_MAIL_HTTPPORT"`
		SmtpPort    uint16 `env:"PAGODA_MAIL_SMTPPORT"`
		User        string `env:"PAGODA_MAIL_USER"`
		Password    string `env:"PAGODA_MAIL_PASSWORD"`
		SMTP        struct {
			Host string `env:"PAGODA_MAIL_SMTP_HOST"`
			Port int    `env:"PAGODA_MAIL_SMTP_PORT" env-default:"587"`
			User string `env:"PAGODA_MAIL_SMTP_USER"`
			Pass string `env:"PAGODA_MAIL_SMTP_PASS"`
			TLS  bool   `env:"PAGODA_MAIL_SMTP_TLS" env-default:"true"`
		}
		Resend struct {
			APIKey string `env:"PAGODA_MAIL_RESEND_API_KEY"`
		}
		ResendAPIKey string `env:"PAGODA_MAIL_RESENDAPIKEY"`
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

// GetConfig loads and returns configuration.
func GetConfig() (Config, error) {
	c := defaultConfig()
	dotEnvPath, hasDotEnv := findDotEnvFromWD()
	if hasDotEnv {
		if err := cleanenv.ReadConfig(dotEnvPath, &c); err != nil {
			return c, fmt.Errorf("read .env: %w", err)
		}
	}
	if err := cleanenv.ReadEnv(&c); err != nil {
		return c, err
	}

	c.App.Environment = resolveEnvironment(c.App.Environment)
	repoPresence, err := managedLayerPresenceFromDotEnvPath(dotEnvPath, hasDotEnv)
	if err != nil {
		return c, fmt.Errorf("read .env presence: %w", err)
	}
	envPresence := managedLayerPresenceFromEnv()
	managedOverrides, managedSet, err := resolveManagedOverrides(c.Managed)
	if err != nil {
		return c, err
	}
	if err := applyManagedOverrides(&c, managedOverrides); err != nil {
		return c, err
	}
	applyLegacyEnvAliases(&c)
	applyDatabaseDriverConfig(&c)
	applyRuntimeDefaults(&c)
	applyProcessesProfileIfUnset(&c, hasExplicitProcessSelection(repoPresence, envPresence, managedSet))
	c.Managed.RuntimeReport = runtimeconfig.BuildReport(runtimeconfig.LayerInputs{
		Defaults:        managedKeyValues(defaultConfig()),
		EffectiveValues: managedKeyValues(c),
		RepoSet:         repoPresence,
		EnvSet:          envPresence,
		ManagedSet:      managedSet,
		ManagedEnabled:  c.Managed.Enabled,
		Authority:       c.Managed.Authority,
	})

	if c.App.FirebaseBase64AccessKeys != "" {
		jsonCreds, err := base64.StdEncoding.DecodeString(c.App.FirebaseBase64AccessKeys)
		if err != nil {
			return c, fmt.Errorf("error decoding firebase credentials: %w", err)
		}
		c.App.FirebaseJSONAccessKeys = jsonCreds
	}

	return c, nil
}

func defaultConfig() Config {
	return Config{
		HTTP: HTTPConfig{
			Hostname:     "localhost",
			Port:         8000,
			Domain:       "http://localhost:8000",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  2 * time.Minute,
			SseKeepAlive: 20 * time.Second,
			TLS: TLSConfig{
				Enabled:     false,
				Certificate: "",
				Key:         "",
			},
		},
		App: AppConfig{
			Name:                             "GoShip",
			SupportEmail:                     "info@goship.run",
			Environment:                      EnvLocal,
			EncryptionKey:                    "?E(G+KbPeShVmYq3t6w9z$C&F)J@McQf",
			Timeout:                          20 * time.Second,
			EmailVerificationTokenExpiration: 12 * time.Hour,
			PageSize:                         3,
			VapidPublicKey:                   "",
			VapidPrivateKey:                  "",
			SentryDsn:                        "my-sentry-dsn-in-config",
			TestSentryUrl:                    "jmGg9OAe2dhR8SpUpgvXXgnB81AD1KUjqyVmCGQIMHoWCIHzQ5",
			PublicStripeKey:                  "pk_...",
			PrivateStripeKey:                 "sk_...",
			StripeWebhookSecret:              "whsec_...",
			StripeWebhookPath:                "/Q2HBfAY7iid59J1SUN8h1Y3WxJcPWA/payments/webhooks",
			AppEncryptionKey:                 "=",
			FirebaseBase64AccessKeys:         "",
			PasswordToken: PasswordTokenConfig{
				Expiration: 60 * time.Minute,
				Length:     64,
			},
			OperationalConstants: OperationalConstants{
				NewsletterSignupEnabled:                           true,
				UserSignupEnabled:                                 true,
				UserSignupEnabledOnLandingPage:                    true,
				QuestionInteractionValidLifetimeInDays:            7,
				NumMinSharedAnswersForPrivateMessages:             3,
				NotifEmojiDebounceTime:                            10 * time.Minute,
				NotifyNewAnswerFromUnansweredQuestionDebounceTime: 12 * time.Hour,
				MinAnswerLen:                                      3,
				PaymentsEnabled:                                   false,
				ProTrialTimespanInDays:                            15,
				ProductProCode:                                    "price_...",
				ProductProPrice:                                   1.49,
				PaymentFailedGracePeriodInDays:                    3,
				DeleteStaleNotificationAfterDays:                  15,
				MaxLikedQuestionHistoryFreePlan:                   3,
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Security: SecurityConfig{
			Headers: SecurityHeadersConfig{
				Enabled: true,
				HSTS:    false,
				CSP:     "",
			},
		},
		Runtime:   RuntimeConfig{},
		Managed:   ManagedConfig{},
		Processes: ProcessesConfig{},
		Adapters: AdaptersConfig{
			DB:     "sqlite",
			Cache:  "otter",
			Jobs:   "backlite",
			PubSub: "inproc",
		},
		Cache: CacheConfig{
			Hostname:     "localhost",
			Port:         6379,
			Password:     "",
			Database:     0,
			TestDatabase: 1,
			Expiration: CacheExpirationConfig{
				StaticFile: 0,
				Page:       0,
			},
		},
		Database: DatabaseConfig{
			Driver:                 "",
			Path:                   "dbs/main.db",
			DbMode:                 DBModeEmbedded,
			EmbeddedDriver:         "sqlite",
			EmbeddedConnection:     "dbs/main.db?_journal=WAL&_timeout=5000&_fk=true",
			EmbeddedTestConnection: ":memory:?_journal=WAL&_timeout=5000&_fk=true",
			Hostname:               "localhost",
			Port:                   5432,
			User:                   "admin",
			Password:               "admin",
			DatabaseNameLocal:      "goship_db",
			DatabaseNameProd:       "",
			TestDatabase:           "goship_testdb",
			SslCertPath:            "prod-ca-2021.cer",
			SslMode:                "require",
		},
		Mail: MailConfig{
			Hostname:     "localhost",
			HttpPort:     8025,
			SmtpPort:     1025,
			User:         "admin",
			Password:     "admin",
			FromAddress:  "info@goship.app",
			ResendAPIKey: "",
		},
		Phone: PhoneConfig{
			SenderID:                        "",
			Region:                          "",
			ValidationCodeExpirationMinutes: 15,
		},
		Recommender: RecommenderConfig{
			NumProfilesToMatchAtOnce: 100,
		},
		Storage: StorageConfig{
			Driver:                    StorageDriverLocal,
			LocalStoragePath:          "./uploads",
			AppBucketName:             "goship-dev",
			StaticFilesBucketName:     "goship-static",
			S3Endpoint:                "s3.us-west-002.backblazeb2.com",
			S3AccessKey:               "0072...",
			S3SecretKey:               "K001...",
			S3UseSSL:                  true,
			ProfilePhotoMaxFileSizeMB: 2,
			PhotosMaxFileSizeMB:       5,
		},
	}
}

func resolveEnvironment(configured environment) environment {
	if env := strings.TrimSpace(os.Getenv("PAGODA_APP_ENVIRONMENT")); env != "" {
		return normalizeEnvironment(environment(env))
	}
	// APP_ENV is supported as a compatibility alias.
	if env := strings.TrimSpace(os.Getenv("APP_ENV")); env != "" {
		return normalizeEnvironment(environment(env))
	}
	if configured != "" {
		return normalizeEnvironment(configured)
	}
	return EnvLocal
}

func normalizeEnvironment(env environment) environment {
	switch strings.ToLower(strings.TrimSpace(string(env))) {
	case "production":
		return EnvProduction
	case "development":
		return EnvDevelop
	case "testing":
		return EnvTest
	case "prod":
		return EnvProduction
	case "dev":
		return EnvDevelop
	case "test":
		return EnvTest
	case "local":
		return EnvLocal
	case "staging":
		return EnvStaging
	case "qa":
		return EnvQA
	default:
		return env
	}
}

func applyLegacyEnvAliases(c *Config) {
	if c == nil {
		return
	}
	if strings.TrimSpace(c.Adapters.Jobs) == "" {
		if value := strings.TrimSpace(os.Getenv("PAGODA_JOBS_DRIVER")); value != "" {
			c.Adapters.Jobs = value
		}
	}
	if strings.TrimSpace(c.Adapters.Cache) == "" {
		if value := strings.TrimSpace(os.Getenv("PAGODA_CACHE_DRIVER")); value != "" {
			c.Adapters.Cache = value
		}
	}
	if value := strings.TrimSpace(os.Getenv("PAGODA_DATABASE_DATABASE")); value != "" {
		if !envIsSet("PAGODA_DATABASE_DATABASENAMELOCAL") {
			c.Database.DatabaseNameLocal = value
		}
		if !envIsSet("PAGODA_DATABASE_DATABASENAMEPROD") {
			c.Database.DatabaseNameProd = value
		}
	}
}

func applyDatabaseDriverConfig(c *Config) {
	if c == nil {
		return
	}

	driver := normalizeDBDriver(string(c.Database.Driver))
	switch {
	case driver != "":
		c.Database.Driver = dbdriver(driver)
	case c.Database.DbMode == DBModeEmbedded:
		c.Database.Driver = DBDriverSQLite
	default:
		c.Database.Driver = DBDriverPostgres
	}

	if strings.TrimSpace(c.Database.Path) == "" {
		c.Database.Path = sqlitePathFromConnection(c.Database.EmbeddedConnection)
	}
	if strings.TrimSpace(c.Database.Path) == "" {
		c.Database.Path = "dbs/main.db"
	}

	switch c.Database.Driver {
	case DBDriverSQLite:
		c.Database.DbMode = DBModeEmbedded
		c.Database.EmbeddedDriver = string(DBDriverSQLite)
		c.Database.EmbeddedConnection = sqliteConnectionString(c.Database.Path)
		if strings.TrimSpace(c.Database.EmbeddedTestConnection) == "" {
			c.Database.EmbeddedTestConnection = ":memory:?_journal=WAL&_timeout=5000&_fk=true"
		}
	case DBDriverPostgres:
		c.Database.DbMode = DBModeStandalone
		if strings.TrimSpace(c.Database.EmbeddedDriver) == "" {
			c.Database.EmbeddedDriver = string(DBDriverSQLite)
		}
	default:
		c.Database.Driver = DBDriverSQLite
		c.Database.DbMode = DBModeEmbedded
		c.Database.EmbeddedDriver = string(DBDriverSQLite)
		c.Database.EmbeddedConnection = sqliteConnectionString(c.Database.Path)
	}
}

func normalizeDBDriver(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "postgres", "postgresql", "pgx":
		return string(DBDriverPostgres)
	case "sqlite", "sqlite3":
		return string(DBDriverSQLite)
	default:
		return ""
	}
}

func normalizeRuntimeProfile(raw string) runtimeprofile {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(RuntimeProfileServerDB):
		return RuntimeProfileServerDB
	case string(RuntimeProfileSingleNode):
		return RuntimeProfileSingleNode
	case string(RuntimeProfileDistributed):
		return RuntimeProfileDistributed
	default:
		return ""
	}
}

func normalizeStorageDriver(raw string) storagedriver {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(StorageDriverLocal):
		return StorageDriverLocal
	case string(StorageDriverMinIO):
		return StorageDriverMinIO
	default:
		return ""
	}
}

func setManagedString(raw string, target *string, key string) error {
	if target == nil {
		return fmt.Errorf("%s target is nil", key)
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		return fmt.Errorf("%s cannot be empty", key)
	}
	*target = value
	return nil
}

func setManagedBool(raw string, target *bool, key string) error {
	if target == nil {
		return fmt.Errorf("%s target is nil", key)
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid boolean value %q for %s", raw, key)
	}
	*target = value
	return nil
}

func managedOverrideAllowlist() map[string]struct{} {
	allowlist := make(map[string]struct{}, len(managedOverrideSpecs))
	for key := range managedOverrideSpecs {
		allowlist[key] = struct{}{}
	}
	return allowlist
}

func managedKeyValues(cfg Config) map[string]string {
	values := make(map[string]string, len(managedOverrideSpecs))
	for key, spec := range managedOverrideSpecs {
		values[key] = strings.TrimSpace(spec.get(cfg))
	}
	return values
}

func resolveManagedOverrides(managed ManagedConfig) (map[string]string, map[string]bool, error) {
	managedSet := map[string]bool{}
	if !managed.Enabled {
		return map[string]string{}, managedSet, nil
	}

	overrides, err := runtimeconfig.ParseManagedOverrides(managed.OverridesJSON)
	if err != nil {
		return nil, nil, err
	}

	rejected := runtimeconfig.RejectUnknownKeys(overrides, managedOverrideAllowlist())
	if len(rejected) > 0 {
		return nil, nil, fmt.Errorf("managed overrides contain non-allowlisted keys: %s", strings.Join(rejected, ", "))
	}

	for key := range overrides {
		managedSet[key] = true
	}
	return overrides, managedSet, nil
}

func applyManagedOverrides(cfg *Config, overrides map[string]string) error {
	if cfg == nil || len(overrides) == 0 {
		return nil
	}

	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		spec, ok := managedOverrideSpecs[key]
		if !ok {
			return fmt.Errorf("unsupported managed override key %q", key)
		}
		if err := spec.set(cfg, overrides[key]); err != nil {
			return fmt.Errorf("apply managed override %q: %w", key, err)
		}
	}

	return nil
}

func managedLayerPresenceFromDotEnvPath(path string, hasDotEnv bool) (map[string]bool, error) {
	if !hasDotEnv || strings.TrimSpace(path) == "" {
		return map[string]bool{}, nil
	}
	return managedLayerPresenceFromDotEnv(path)
}

func managedLayerPresenceFromDotEnv(path string) (map[string]bool, error) {
	names, err := envNamesFromFile(path)
	if err != nil {
		return nil, err
	}
	return managedLayerPresenceFromNames(names), nil
}

func managedLayerPresenceFromEnv() map[string]bool {
	names := map[string]struct{}{}
	for _, raw := range os.Environ() {
		name, _, ok := strings.Cut(raw, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		names[name] = struct{}{}
	}
	return managedLayerPresenceFromNames(names)
}

func managedLayerPresenceFromNames(names map[string]struct{}) map[string]bool {
	presence := map[string]bool{}
	for key, spec := range managedOverrideSpecs {
		for _, envVar := range spec.envVars {
			if _, ok := names[envVar]; ok {
				presence[key] = true
				break
			}
		}
	}
	return presence
}

func envNamesFromFile(path string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	names := map[string]struct{}{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		name, _, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		names[name] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return names, nil
}

func hasExplicitProcessSelection(repoPresence, envPresence, managedSet map[string]bool) bool {
	keys := []string{
		"processes.web",
		"processes.worker",
		"processes.scheduler",
		"processes.colocated",
	}
	for _, key := range keys {
		if repoPresence[key] || envPresence[key] || managedSet[key] {
			return true
		}
	}
	return false
}

func sqlitePathFromConnection(conn string) string {
	v := strings.TrimSpace(conn)
	if v == "" {
		return ""
	}
	if idx := strings.Index(v, "?"); idx >= 0 {
		v = v[:idx]
	}
	return strings.TrimSpace(v)
}

func sqliteConnectionString(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		p = "dbs/main.db"
	}
	return p + "?_journal=WAL&_timeout=5000&_fk=true"
}

func applyRuntimeDefaults(c *Config) {
	if c == nil || c.Runtime.Profile != "" {
		return
	}

	switch c.App.Environment {
	case EnvProduction:
		c.Runtime.Profile = RuntimeProfileDistributed
	default:
		c.Runtime.Profile = RuntimeProfileServerDB
	}
}

func applyProcessesProfile(c *Config) {
	applyProcessesProfileIfUnset(c, anyEnvIsSet(
		"PAGODA_PROCESSES_WEB",
		"PAGODA_PROCESSES_WORKER",
		"PAGODA_PROCESSES_SCHEDULER",
		"PAGODA_PROCESSES_COLOCATED",
	))
}

func applyProcessesProfileIfUnset(c *Config, explicitProcessConfig bool) {
	if c == nil || explicitProcessConfig {
		return
	}

	switch c.Runtime.Profile {
	case RuntimeProfileSingleNode:
		c.Processes = ProcessesConfig{
			Web:       true,
			Worker:    true,
			Scheduler: true,
			CoLocated: true,
		}
	case RuntimeProfileDistributed:
		c.Processes = ProcessesConfig{
			Web:       true,
			Worker:    false,
			Scheduler: false,
			CoLocated: false,
		}
	default:
		c.Processes = ProcessesConfig{
			Web:       true,
			Worker:    false,
			Scheduler: false,
			CoLocated: false,
		}
	}
}

func anyEnvIsSet(names ...string) bool {
	for _, name := range names {
		if envIsSet(name) {
			return true
		}
	}
	return false
}

func envIsSet(name string) bool {
	value, ok := os.LookupEnv(name)
	return ok && strings.TrimSpace(value) != ""
}
