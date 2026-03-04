package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// TemplateExt stores the extension used for the template files
	TemplateExt = ".gohtml"

	// StaticDir stores the directory served as static assets for the example app.
	StaticDir = "apps/site/static"

	// StaticPrefix stores the URL prefix used when serving static files
	StaticPrefix = "files"
)

type app string
type environment string
type dbmode string
type runtimeprofile string

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
		Profile runtimeprofile
	}

	ProcessesConfig struct {
		Web       bool
		Worker    bool
		Scheduler bool
		CoLocated bool
	}

	AdaptersConfig struct {
		DB     string
		Cache  string
		Jobs   string
		PubSub string
	}

	// HTTPConfig stores HTTP configuration
	HTTPConfig struct {
		Hostname     string
		Port         uint16
		Domain       string
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
		SseKeepAlive time.Duration
		TLS          struct {
			Enabled     bool
			Certificate string
			Key         string
		}
	}

	// AppConfig stores application configuration
	AppConfig struct {
		Name          app
		SupportEmail  string
		Environment   environment
		EncryptionKey string
		Timeout       time.Duration
		PasswordToken struct {
			Expiration time.Duration
			Length     int
		}
		EmailVerificationTokenExpiration time.Duration
		OperationalConstants             OperationalConstants
		PageSize                         int
		VapidPublicKey                   string
		VapidPrivateKey                  string
		SentryDsn                        string
		TestSentryUrl                    string
		PublicStripeKey                  string
		PrivateStripeKey                 string
		StripeWebhookSecret              string
		StripeWebhookPath                string
		AppEncryptionKey                 string
		FirebaseBase64AccessKeys         string
		FirebaseJSONAccessKeys           []byte
	}

	OperationalConstants struct {
		NewsletterSignupEnabled                           bool
		UserSignupEnabled                                 bool
		UserSignupEnabledOnLandingPage                    bool
		QuestionInteractionValidLifetimeInDays            int
		NumMinSharedAnswersForPrivateMessages             int
		NotifEmojiDebounceTime                            time.Duration
		NotifyNewAnswerFromUnansweredQuestionDebounceTime time.Duration
		MinAnswerLen                                      int
		PaymentsEnabled                                   bool
		ProTrialTimespanInDays                            int
		ProductProCode                                    string
		ProductProPrice                                   float32
		PaymentFailedGracePeriodInDays                    int
		DeleteStaleNotificationAfterDays                  int
		MaxLikedQuestionHistoryFreePlan                   int
	}

	// CacheConfig stores the cache configuration
	CacheConfig struct {
		Hostname     string
		Port         uint16
		Password     string
		Database     int
		TestDatabase int
		Expiration   struct {
			StaticFile time.Duration
			Page       time.Duration
		}
	}

	// DatabaseConfig stores the database configuration
	DatabaseConfig struct {
		DbMode                 dbmode
		EmbeddedDriver         string
		EmbeddedConnection     string
		EmbeddedTestConnection string
		// TODO: eventually separate in-memory (SQLite) and standalone DB configs
		Hostname          string
		Port              uint16
		User              string
		Password          string
		DatabaseNameLocal string
		DatabaseNameProd  string
		TestDatabase      string
		SslCertPath       string
		SslMode           string
	}

	// MailConfig stores the mail configuration
	MailConfig struct {
		Hostname     string
		HttpPort     uint16
		SmtpPort     uint16
		User         string
		Password     string
		FromAddress  string
		ResendAPIKey string
	}

	PhoneConfig struct {
		SenderID                        string
		Region                          string
		ValidationCodeExpirationMinutes int
	}

	RecommenderConfig struct {
		NumProfilesToMatchAtOnce int
	}

	StorageConfig struct {
		AppBucketName             string
		StaticFilesBucketName     string
		S3Endpoint                string
		S3AccessKey               string
		S3SecretKey               string
		S3UseSSL                  bool
		ProfilePhotoMaxFileSizeMB int64
		PhotosMaxFileSizeMB       int64
	}
)

// GetConfig loads and returns configuration
func GetConfig() (Config, error) {
	var c Config
	roots := configSearchRoots()
	v := viper.New()
	v.SetConfigType("yaml")

	// Base config is required.
	if err := mergeNamedConfig(v, roots, "application.yaml", true); err != nil {
		return c, err
	}

	// Unmarshal once to detect default environment before environment-specific overrides.
	if err := v.Unmarshal(&c); err != nil {
		return c, err
	}
	env := resolveEnvironment(c.App.Environment)

	// Layer per-environment overrides.
	if err := mergeNamedConfig(v, roots, filepath.Join("environments", string(env)+".yaml"), true); err != nil {
		return c, err
	}

	// Production supports env var overrides with PAGODA_ prefix.
	if env == EnvProduction {
		// Load env variables for production
		v.SetEnvPrefix("pagoda")
		v.AutomaticEnv()
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	}

	if err := v.Unmarshal(&c); err != nil {
		return c, err
	}
	c.App.Environment = env
	applyRuntimeDefaults(&c)

	if err := applyProcessesProfile(&c, roots); err != nil {
		return c, err
	}

	if c.App.FirebaseBase64AccessKeys != "" {
		jsonCreds, err := base64.StdEncoding.DecodeString(c.App.FirebaseBase64AccessKeys)
		if err != nil {
			return c, fmt.Errorf("error decoding firebase credentials: %w", err)
		}
		c.App.FirebaseJSONAccessKeys = jsonCreds
	}

	return c, nil
}

func configSearchRoots() []string {
	return []string{
		".",
		"config",
		"../config",
		"../../config",
		"../../../config",
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

func applyRuntimeDefaults(c *Config) {
	if c.Runtime.Profile == "" {
		c.Runtime.Profile = RuntimeProfileServerDB
	}
}

func mergeNamedConfig(dst *viper.Viper, roots []string, relPath string, required bool) error {
	for _, root := range roots {
		absPath := filepath.Clean(filepath.Join(root, relPath))
		info, err := os.Stat(absPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("stat config file %q: %w", absPath, err)
		}
		if info.IsDir() {
			continue
		}

		src := viper.New()
		src.SetConfigFile(absPath)
		src.SetConfigType("yaml")
		if err := src.ReadInConfig(); err != nil {
			return fmt.Errorf("read config file %q: %w", absPath, err)
		}
		if err := dst.MergeConfigMap(src.AllSettings()); err != nil {
			return fmt.Errorf("merge config file %q: %w", absPath, err)
		}
		return nil
	}

	if required {
		return fmt.Errorf("required config file %q not found in search paths", relPath)
	}
	return nil
}

type processProfilesFile struct {
	Profiles map[runtimeprofile]ProcessesConfig `mapstructure:"profiles"`
}

func applyProcessesProfile(c *Config, roots []string) error {
	pv := viper.New()
	pv.SetConfigType("yaml")
	if err := mergeNamedConfig(pv, roots, "processes.yaml", true); err != nil {
		return err
	}

	var profiles processProfilesFile
	if err := pv.Unmarshal(&profiles); err != nil {
		return fmt.Errorf("unmarshal processes profiles: %w", err)
	}
	if len(profiles.Profiles) == 0 {
		return fmt.Errorf("processes.yaml missing profiles")
	}

	if p, ok := profiles.Profiles[c.Runtime.Profile]; ok {
		c.Processes = p
		return nil
	}
	return fmt.Errorf("runtime profile %q not found in processes.yaml", c.Runtime.Profile)
}
