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
	applyBackupDefaults(&c)
	applyRuntimeDefaults(&c)
	if err := applyUIProviderDefaults(&c); err != nil {
		return c, err
	}
	applyProcessesProfileIfUnset(&c, hasExplicitProcessSelection(repoPresence, envPresence, managedSet))
	c.Managed.RuntimeReport = runtimeconfig.BuildReport(runtimeconfig.LayerInputs{
		Defaults:        managedKeyValues(normalizedDefaultConfigForReporting()),
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
			SentryDsn:                        "",
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
		Runtime: RuntimeConfig{},
		Managed: ManagedConfig{
			HooksMaxSkewSeconds:  300,
			HooksNonceTTLSeconds: 300,
		},
		UI:        UIConfig{},
		Processes: ProcessesConfig{},
		Adapters: AdaptersConfig{
			DB:     "sqlite",
			Cache:  "otter",
			Jobs:   "backlite",
			PubSub: "inproc",
		},
		Metrics: MetricsConfig{
			Enabled:  true,
			Path:     "/metrics",
			Exporter: "prometheus",
			Format:   "prometheus-text",
		},
		I18n: I18nConfig{
			Enabled:         true,
			DefaultLanguage: "en",
			StrictMode:      "off",
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
			Path:                   ".local/db/main.db",
			DbMode:                 DBModeEmbedded,
			EmbeddedDriver:         "sqlite",
			EmbeddedConnection:     ".local/db/main.db?_journal=WAL&_timeout=5000&_fk=true",
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
		Backup: BackupConfig{
			Driver:        "sqlite-file",
			SchemaVersion: "v1",
			SQLitePath:    "",
			S3: BackupS3Config{
				Enabled:   false,
				Endpoint:  "s3.us-west-002.backblazeb2.com",
				Region:    "us-west-002",
				Bucket:    "goship-backups",
				Prefix:    "snapshots",
				AccessKey: "0072...",
				SecretKey: "K001...",
				UseSSL:    true,
			},
		},
		OAuth: OAuthConfig{
			GitHub:  OAuthGitHubConfig{},
			Google:  OAuthGoogleConfig{},
			Discord: OAuthDiscordConfig{},
		},
		AI: AIConfig{
			Driver: "anthropic",
			Anthropic: AIAnthropicConfig{
				DefaultModel: "claude-haiku-4-5-20251001",
			},
			OpenAI: AIOpenAIConfig{
				DefaultModel: "gpt-4o-mini",
			},
			OpenRouter: AIOpenRouterConfig{
				DefaultModel: "anthropic/claude-haiku-4-5-20251001",
			},
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
		c.Database.Path = ".local/db/main.db"
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

func applyUIProviderDefaults(c *Config) error {
	if c == nil {
		return nil
	}
	provider := strings.ToLower(strings.TrimSpace(string(c.UI.Provider)))
	switch provider {
	case "":
		c.UI.Provider = UIProviderFranken
	case string(UIProviderFranken):
		c.UI.Provider = UIProviderFranken
	case string(UIProviderDaisy):
		c.UI.Provider = UIProviderDaisy
	case string(UIProviderBare):
		c.UI.Provider = UIProviderBare
	default:
		return fmt.Errorf("unsupported ui provider %q", c.UI.Provider)
	}
	return nil
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
		p = ".local/db/main.db"
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
		c.Runtime.Profile = RuntimeProfileSingleNode
	}
}

func applyBackupDefaults(c *Config) {
	if c == nil {
		return
	}

	c.Backup.Driver = strings.TrimSpace(c.Backup.Driver)
	if c.Backup.Driver == "" {
		c.Backup.Driver = "sqlite-file"
	}
	c.Backup.SchemaVersion = strings.TrimSpace(c.Backup.SchemaVersion)
	if c.Backup.SchemaVersion == "" {
		c.Backup.SchemaVersion = "v1"
	}
	c.Backup.SQLitePath = strings.TrimSpace(c.Backup.SQLitePath)
	if c.Backup.SQLitePath == "" {
		c.Backup.SQLitePath = strings.TrimSpace(c.Database.Path)
	}
	if c.Backup.SQLitePath == "" {
		c.Backup.SQLitePath = ".local/db/main.db"
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
