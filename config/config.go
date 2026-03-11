package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
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
		Runtime     RuntimeConfig
		Processes   ProcessesConfig
		Adapters    AdaptersConfig
		Cache       CacheConfig
		Database    DatabaseConfig
		Mail        MailConfig
		Phone       PhoneConfig
		Recommender RecommenderConfig
		Storage     StorageConfig
	}

	RuntimeConfig struct {
		Profile runtimeprofile `env:"PAGODA_RUNTIME_PROFILE"`
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
		Hostname     string `env:"PAGODA_MAIL_HOSTNAME"`
		HttpPort     uint16 `env:"PAGODA_MAIL_HTTPPORT"`
		SmtpPort     uint16 `env:"PAGODA_MAIL_SMTPPORT"`
		User         string `env:"PAGODA_MAIL_USER"`
		Password     string `env:"PAGODA_MAIL_PASSWORD"`
		FromAddress  string `env:"PAGODA_MAIL_FROMADDRESS"`
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
		AppBucketName             string `env:"PAGODA_STORAGE_APPBUCKETNAME"`
		StaticFilesBucketName     string `env:"PAGODA_STORAGE_STATICFILESBUCKETNAME"`
		S3Endpoint                string `env:"PAGODA_STORAGE_S3ENDPOINT"`
		S3AccessKey               string `env:"PAGODA_STORAGE_S3ACCESSKEY"`
		S3SecretKey               string `env:"PAGODA_STORAGE_S3SECRETKEY"`
		S3UseSSL                  bool   `env:"PAGODA_STORAGE_S3USESSL"`
		ProfilePhotoMaxFileSizeMB int64  `env:"PAGODA_STORAGE_PROFILEPHOTOMAXFILESIZEMB"`
		PhotosMaxFileSizeMB       int64  `env:"PAGODA_STORAGE_PHOTOSMAXFILESIZEMB"`
	}
)

// GetConfig loads and returns configuration.
func GetConfig() (Config, error) {
	c := defaultConfig()
	if path, ok := findDotEnvFromWD(); ok {
		if err := cleanenv.ReadConfig(path, &c); err != nil {
			return c, fmt.Errorf("read .env: %w", err)
		}
	}
	if err := cleanenv.ReadEnv(&c); err != nil {
		return c, err
	}

	c.App.Environment = resolveEnvironment(c.App.Environment)
	applyLegacyEnvAliases(&c)
	applyDatabaseDriverConfig(&c)
	applyRuntimeDefaults(&c)
	applyProcessesProfile(&c)

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
		Runtime:   RuntimeConfig{},
		Processes: ProcessesConfig{},
		Adapters: AdaptersConfig{
			DB:     "postgres",
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
		return environment(env)
	}
	if configured != "" {
		return configured
	}
	return EnvLocal
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
	if c == nil || anyEnvIsSet(
		"PAGODA_PROCESSES_WEB",
		"PAGODA_PROCESSES_WORKER",
		"PAGODA_PROCESSES_SCHEDULER",
		"PAGODA_PROCESSES_COLOCATED",
	) {
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
