package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-playground/validator/v10"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/robfig/cron/v3"
	"github.com/stripe/stripe-go/v78"
	_ "modernc.org/sqlite"

	"github.com/leomorpho/goship-modules/notifications"
	"github.com/leomorpho/goship/config"
	dbqueries "github.com/leomorpho/goship/db/queries"
	cacherepo "github.com/leomorpho/goship/framework/cache"
	"github.com/leomorpho/goship/framework/core"
	adapters "github.com/leomorpho/goship/framework/core/adapters"
	"github.com/leomorpho/goship/framework/events"
	eventtypes "github.com/leomorpho/goship/framework/events/types"
	"github.com/leomorpho/goship/framework/health"
	"github.com/leomorpho/goship/framework/logging"
	"github.com/leomorpho/goship/framework/mailer"
	pubsubrepo "github.com/leomorpho/goship/framework/pubsub"
	"github.com/leomorpho/goship/framework/sse"
	"github.com/leomorpho/goship/modules/authsupport"
	i18nmodule "github.com/leomorpho/goship/modules/i18n"
)

// Container contains all services used by the application and provides an easy way to handle dependency
// injection including within tests
type Container struct {
	// Validator stores a validator
	Validator echo.Validator

	// Web stores the web framework
	Web *echo.Echo

	Logger echo.Logger

	// Config stores the application configuration
	Config *config.Config

	// Cache contains the cache client
	Cache *cacherepo.CacheClient

	// Database stores the connection to the database
	Database    *sql.DB
	databaseDSN string

	// Mail stores an email sending client
	Mail *mailer.MailClient

	// Auth stores an authentication client
	Auth *authsupport.AuthClient

	// I18n stores localized message resolution for request flows.
	I18n core.I18n

	// EventBus stores the synchronous domain event bus.
	EventBus *events.Bus

	// Notifier handles all notifications to clients
	Notifier *notifications.NotifierService

	// SSEHub stores the in-process SSE fan-out hub.
	SSEHub *sse.Hub

	// CoreCache exposes cache via the backend-agnostic core seam.
	CoreCache core.Cache
	// CoreJobs exposes jobs via the backend-agnostic core seam.
	CoreJobs core.Jobs
	// CoreJobsInspector exposes jobs inspection via the backend-agnostic core seam.
	CoreJobsInspector core.JobsInspector
	// CorePubSub exposes pubsub via the backend-agnostic core seam.
	CorePubSub core.PubSub

	// Health stores the framework-default liveness/readiness registry.
	Health *health.Registry

	// Scheduler stores cron-based app schedule registration.
	Scheduler *cron.Cron

	// Adapters stores resolved adapter selection/capabilities for runtime use.
	Adapters adapters.Resolved
}

type requestValidator struct {
	validator *validator.Validate
}

func newRequestValidator() *requestValidator {
	return &requestValidator{validator: validator.New()}
}

func (v *requestValidator) Validate(i any) error {
	if v == nil || v.validator == nil {
		return nil
	}
	return v.validator.Struct(i)
}

// NewContainer creates and initializes a new runtime container.
func NewContainer(registerSchedules func(*cron.Cron, func() core.Jobs)) *Container {
	c := new(Container)
	c.initConfig()
	c.I18n = i18nmodule.New(c.Config)
	c.validateAdapterPlan()
	c.initValidator()
	c.initWeb()
	c.initOptionalServices()
	c.initDatabase()
	c.initSchema()
	c.initAuth()
	c.initMail()
	c.initEventBus()
	c.initPaymentProcessor()
	c.initScheduler(registerSchedules)
	// ship:container:start
	// ship:container:end
	c.initCoreAdapters()
	c.initHealth()
	c.validateStartupContract()
	c.initSSEHub()
	return c
}

func (c *Container) initOptionalServices() {
	if c.shouldInitCache() {
		c.initCache()
	}
}

func (c *Container) shouldInitCache() bool {
	if c == nil {
		return false
	}

	switch c.Adapters.Selection.Cache {
	case "redis", "otter", "memory":
		return true
	default:
		return false
	}
}

func (c *Container) validateAdapterPlan() {
	resolved, err := ResolveAdapterPlan(c.Config)
	if err != nil {
		panic(fmt.Sprintf("startup configuration failure: invalid adapter plan (%v). Check %s.", err, adapterPlanEnvHint(err)))
	}
	c.Adapters = resolved
}

func adapterPlanEnvHint(err error) string {
	if err == nil {
		return "PAGODA_ADAPTERS_DB/PAGODA_ADAPTERS_CACHE/PAGODA_ADAPTERS_JOBS/PAGODA_ADAPTERS_PUBSUB"
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "db adapter"):
		return "PAGODA_ADAPTERS_DB"
	case strings.Contains(msg, "cache adapter"):
		return "PAGODA_ADAPTERS_CACHE"
	case strings.Contains(msg, "jobs adapter"):
		return "PAGODA_ADAPTERS_JOBS"
	case strings.Contains(msg, "pubsub adapter") && strings.Contains(msg, "requires cache adapter"):
		return "PAGODA_ADAPTERS_PUBSUB and PAGODA_ADAPTERS_CACHE"
	case strings.Contains(msg, "pubsub adapter"):
		return "PAGODA_ADAPTERS_PUBSUB"
	default:
		return "PAGODA_ADAPTERS_DB/PAGODA_ADAPTERS_CACHE/PAGODA_ADAPTERS_JOBS/PAGODA_ADAPTERS_PUBSUB"
	}
}

// ResolveAdapterPlan validates adapter configuration and returns the resolved runtime selection.
func ResolveAdapterPlan(cfg *config.Config) (adapters.Resolved, error) {
	if cfg == nil {
		return adapters.Resolved{}, fmt.Errorf("invalid container state: nil config")
	}
	return adapters.ResolveFromConfig(cfg)
}

func (c *Container) initCoreAdapters() {
	c.CoreCache = adapters.NewCoreCacheAdapter(c.Cache)
	c.CoreJobs = adapters.NewCoreJobsAdapter(nil, c.Adapters.JobsCapabilities)
	c.CoreJobsInspector = adapters.NewCoreJobsInspectorAdapter(nil)

	var ps pubsubrepo.PubSubClient
	switch c.Adapters.Selection.PubSub {
	case "redis":
		if c.Cache != nil && c.Cache.Client != nil {
			ps = pubsubrepo.NewRedisPubSubClient(c.Cache.Client)
		} else {
			ps = pubsubrepo.NewInProcPubSubClient()
		}
	default:
		ps = pubsubrepo.NewInProcPubSubClient()
	}
	c.CorePubSub = adapters.NewCorePubSubAdapter(ps)
}

func (c *Container) initScheduler(registerSchedules func(*cron.Cron, func() core.Jobs)) {
	c.Scheduler = cron.New(cron.WithSeconds())
	if registerSchedules != nil {
		registerSchedules(c.Scheduler, func() core.Jobs {
			return c.CoreJobs
		})
	}
}

func (c *Container) initHealth() {
	if c == nil {
		return
	}

	requiredEnv := []health.EnvRequirement{
		{Name: "PAGODA_APP_ENVIRONMENT", Value: string(c.Config.App.Environment)},
		{Name: "PAGODA_ADAPTERS_DB", Value: c.Adapters.Selection.DB},
		{Name: "PAGODA_ADAPTERS_CACHE", Value: c.Adapters.Selection.Cache},
		{Name: "PAGODA_ADAPTERS_JOBS", Value: c.Adapters.Selection.Jobs},
		{Name: "PAGODA_ADAPTERS_PUBSUB", Value: c.Adapters.Selection.PubSub},
	}
	if c.Config.Database.DbMode == config.DBModeEmbedded {
		requiredEnv = append(requiredEnv, health.EnvRequirement{
			Name:  "PAGODA_DB_PATH",
			Value: c.Config.Database.Path,
		})
	} else {
		dbPort := ""
		if c.Config.Database.Port > 0 {
			dbPort = strconv.Itoa(int(c.Config.Database.Port))
		}
		requiredEnv = append(requiredEnv,
			health.EnvRequirement{Name: "PAGODA_DATABASE_HOSTNAME", Value: c.Config.Database.Hostname},
			health.EnvRequirement{Name: "PAGODA_DATABASE_PORT", Value: dbPort},
		)
	}

	c.Health = health.NewRegistry(
		health.NewDBChecker(c.Database, 2*time.Second),
		health.NewCacheChecker(c.CoreCache, 2*time.Second),
		health.NewJobsChecker(c.CoreJobsInspector, 2*time.Second),
		health.NewEnvChecker(requiredEnv...),
	)
}

func (c *Container) validateStartupContract() {
	if c == nil {
		panic("startup configuration failure: container is not configured")
	}
	if err := c.Health.ValidateStartupContract(); err != nil {
		panic(fmt.Sprintf("startup configuration failure: %v", err))
	}
}

// Shutdown shuts the Container down and disconnects all connections
func (c *Container) Shutdown() error {
	if c.Cache != nil {
		if err := c.Cache.Close(); err != nil {
			return err
		}
	}
	if c.Database != nil {
		if err := c.Database.Close(); err != nil {
			return err
		}
	}

	return nil
}

// initConfig initializes configuration
func (c *Container) initConfig() {
	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("startup configuration failure: could not load config from .env/environment (%v). Check required PAGODA_* settings and secret values.", err))
	}
	if issues := config.ValidateConfigSemantics(cfg); len(issues) > 0 {
		msgs := make([]string, 0, len(issues))
		for _, issue := range issues {
			msgs = append(msgs, issue.Error())
		}
		panic(fmt.Sprintf("startup configuration failure: %s", strings.Join(msgs, "; ")))
	}
	c.Config = &cfg
}

// initValidator initializes the validator
func (c *Container) initValidator() {
	c.Validator = newRequestValidator()
}

// initWeb initializes the web framework
func (c *Container) initWeb() {

	c.Web = echo.New()
	if sentryDsn := strings.TrimSpace(c.Config.App.SentryDsn); sentryDsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:         sentryDsn,
			Environment: string(c.Config.App.Environment),
		}); err != nil {
			panic(fmt.Sprintf("startup sentry initialization failure: %v", err))
		}
	}

	// Create an slog logger instance
	slogLogger := logging.NewLogger(c.Config.Log)

	c.Logger = logging.NewEchoLogger(slogLogger)
	c.Web.Logger = c.Logger
	c.Web.Validator = c.Validator
}

// initCache initializes the cache
func (c *Container) initCache() {
	var err error
	if c.Cache, err = cacherepo.NewClient(c.Config); err != nil {
		panic(fmt.Sprintf(
			"startup cache service failure: could not initialize cache adapter %q (%v). Check PAGODA_ADAPTERS_CACHE plus cache service settings (PAGODA_CACHE_HOSTNAME/PAGODA_CACHE_PORT).",
			c.Adapters.Selection.Cache,
			err,
		))
	}
}

func (c *Container) getDBAddr(dbName string) string {
	c.databaseDSN = fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		c.Config.Database.User,
		c.Config.Database.Password,
		c.Config.Database.Hostname,
		c.Config.Database.Port,
		dbName,
	)
	return c.databaseDSN
}

func (c *Container) getProdDBAddr(dbName string) string {

	c.databaseDSN = fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s&sslrootcert=%s",
		c.Config.Database.User,
		c.Config.Database.Password,
		c.Config.Database.Hostname,
		c.Config.Database.Port,
		dbName,
		c.Config.Database.SslMode,
		c.Config.Database.SslCertPath,
	)
	return c.databaseDSN
}

// initDatabase initializes the database
// If the environment is set to test, the test database will be used and will be dropped, recreated and migrated
func (c *Container) initDatabase() {
	var connection string
	var err error

	if c.Config.Database.DbMode == config.DBModeEmbedded {
		switch c.Config.App.Environment {
		case config.EnvTest:
			connection = c.Config.Database.EmbeddedTestConnection
			if err := resetEmbeddedTestDB(connection); err != nil {
				panic(fmt.Sprintf("startup database service failure: could not reset embedded test database (%v)", err))
			}
		default:
			connection = c.Config.Database.EmbeddedConnection
		}

		c.Database, err = OpenEmbeddedDB(c.Config.Database.EmbeddedDriver, connection)
		if err != nil {
			panic(fmt.Sprintf(
				"startup database service failure: could not initialize embedded database driver %q (%v). Check PAGODA_DATABASE_EMBEDDEDDRIVER and PAGODA_DATABASE_EMBEDDEDCONNECTION/PAGODA_DB_PATH.",
				c.Config.Database.EmbeddedDriver,
				err,
			))
		}
	} else {

		if c.Config.App.Environment == config.EnvProduction {
			c.Database, err = sql.Open("pgx", c.getProdDBAddr(c.Config.Database.DatabaseNameProd))
			if err != nil {
				panic(fmt.Sprintf("startup database service failure: could not open postgres connection for production (%v). Check PAGODA_DATABASE_HOSTNAME/PAGODA_DATABASE_PORT and credentials.", err))
			}
		} else {
			c.Database, err = sql.Open("pgx", c.getDBAddr(c.Config.Database.DatabaseNameLocal))
			if err != nil {
				panic(fmt.Sprintf("startup database service failure: could not open postgres connection (%v). Check PAGODA_DATABASE_HOSTNAME/PAGODA_DATABASE_PORT and credentials.", err))
			}
		}

		// Check if this is a test environment
		if c.Config.App.Environment == config.EnvTest {
			dropTestDatabase, err := dbqueries.Get("drop_database_postgres")
			if err != nil {
				panic(fmt.Sprintf("failed to load test DB drop query: %v", err))
			}
			// Drop the test database, ignoring errors in case it doesn't yet exist
			_, _ = c.Database.Exec(dropTestDatabase + c.Config.Database.TestDatabase)

			createTestDatabase, err := dbqueries.Get("create_database_postgres")
			if err != nil {
				panic(fmt.Sprintf("failed to load test DB create query: %v", err))
			}
			// Create the test database
			if _, err = c.Database.Exec(createTestDatabase + c.Config.Database.TestDatabase); err != nil {
				panic(fmt.Sprintf("startup database service failure: could not create test database %q (%v). Check PAGODA_DATABASE_HOSTNAME/PAGODA_DATABASE_PORT and privileges.", c.Config.Database.TestDatabase, err))
			}

			// Connect to the test database
			if err = c.Database.Close(); err != nil {
				panic(fmt.Sprintf("startup database service failure: could not close bootstrap database connection (%v)", err))
			}
			c.Database, err = sql.Open("pgx", c.getDBAddr(c.Config.Database.TestDatabase))
			if err != nil {
				panic(fmt.Sprintf("startup database service failure: could not open test database connection (%v). Check PAGODA_DATABASE_HOSTNAME/PAGODA_DATABASE_PORT and credentials.", err))
			}
		}
		// Create the pgvector extension
		createVectorExtension, err := dbqueries.Get("create_pgvector_extension_postgres")
		if err != nil {
			panic(fmt.Sprintf("failed to load pgvector extension query: %v", err))
		}
		_, err = c.Database.Exec(createVectorExtension)
		if err != nil {
			panic(fmt.Sprintf("startup database service failure: could not enable pgvector extension (%v). Check postgres service availability plus PAGODA_DATABASE_HOSTNAME/PAGODA_DATABASE_PORT and extension privileges.", err))
		}
	}
}

// initSchema runs DB migrations for the app.
func (c *Container) initSchema() {
	migrationsDir, err := ResolveMigrationsDir()
	if err != nil {
		panic(fmt.Sprintf("failed to resolve migrations directory: %v", err))
	}
	driver := "postgres"
	if c.Config.Database.DbMode == config.DBModeEmbedded {
		driver = NormalizeSQLiteDriver(c.Config.Database.EmbeddedDriver)
	}
	if IsSQLiteDriver(driver) {
		if err := EnsureEmbeddedSQLiteSchema(c.Database); err != nil {
			panic(fmt.Sprintf("failed to initialize embedded sqlite schema: %v", err))
		}
		return
	}
	if err := ApplySQLMigrations(c.Database, migrationsDir, driver); err != nil {
		panic(fmt.Sprintf("failed to run SQL migrations: %v", err))
	}
}

// initAuth initializes the authentication client
func (c *Container) initAuth() {
	c.Auth = authsupport.NewAuthClient(c.Config, authsupport.SelectStore(c.Config, c.Database))
}

// initMail initialize the mail client
func (c *Container) initMail() {
	var err error
	mailClientImplementation := c.resolveMailImplementation()
	c.Mail, err = mailer.NewMailClient(c.Config, mailClientImplementation)
	if err != nil {
		panic(fmt.Sprintf("failed to create mail client: %v", err))
	}
}

func (c *Container) resolveMailImplementation() mailer.MailClientInterface {
	driver := strings.ToLower(strings.TrimSpace(c.Config.Mail.Driver))
	if driver == "" {
		driver = "log"
	}

	switch driver {
	case "resend":
		apiKey := strings.TrimSpace(c.Config.Mail.Resend.APIKey)
		if apiKey == "" {
			apiKey = strings.TrimSpace(c.Config.Mail.ResendAPIKey)
		}
		if apiKey == "" {
			panic("startup mail secret missing: resend mail driver requires PAGODA_MAIL_RESEND_API_KEY (or PAGODA_MAIL_RESENDAPIKEY)")
		}
		return mailer.NewResendMailClient(apiKey)
	case "smtp":
		host := strings.TrimSpace(c.Config.Mail.SMTP.Host)
		if host == "" {
			host = strings.TrimSpace(c.Config.Mail.Hostname)
		}
		if host == "" {
			host = "localhost"
		}
		port := c.Config.Mail.SMTP.Port
		if port == 0 {
			port = int(c.Config.Mail.SmtpPort)
		}
		if port == 0 {
			port = 1025
		}
		return mailer.NewSMTPMailClientWithAuth(host, port, c.Config.Mail.SMTP.User, c.Config.Mail.SMTP.Pass)
	case "log":
		return mailer.NewLogMailClient(slog.Default())
	default:
		panic(fmt.Sprintf("startup mail configuration failure: unsupported mail driver %q (set PAGODA_MAIL_DRIVER to log, smtp, or resend)", c.Config.Mail.Driver))
	}
}

func (c *Container) initEventBus() {
	c.EventBus = events.NewBus()

	logger := logging.NewLogger(c.Config.Log)
	events.Subscribe(c.EventBus, func(_ context.Context, event eventtypes.UserRegistered) error {
		logger.Info("domain event published", "event", "UserRegistered", "user_id", event.UserID)
		return nil
	})
	events.Subscribe(c.EventBus, func(_ context.Context, event eventtypes.UserLoggedIn) error {
		logger.Info("domain event published", "event", "UserLoggedIn", "user_id", event.UserID, "ip", event.IP)
		return nil
	})
	events.Subscribe(c.EventBus, func(_ context.Context, event eventtypes.UserLoggedOut) error {
		logger.Info("domain event published", "event", "UserLoggedOut", "user_id", event.UserID)
		return nil
	})
}

func (c *Container) initSSEHub() {
	c.SSEHub = sse.NewHub(c.CorePubSub)
}

func (c *Container) initPaymentProcessor() {
	stripe.Key = c.Config.App.PrivateStripeKey
}
