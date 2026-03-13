package foundation

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/labstack/echo/v4"
	anthropicdriver "github.com/leomorpho/goship/modules/ai/drivers/anthropic"
	"github.com/stripe/stripe-go/v78"
	_ "modernc.org/sqlite"

	"github.com/leomorpho/goship-modules/notifications"
	"github.com/leomorpho/goship/config"
	dbqueries "github.com/leomorpho/goship/db/queries"
	"github.com/leomorpho/goship/framework/core"
	coreadapters "github.com/leomorpho/goship/framework/core/adapters"
	"github.com/leomorpho/goship/framework/logging"
	"github.com/leomorpho/goship/framework/repos/mailer"
	"github.com/leomorpho/goship/modules/ai"
)

// Container contains all services used by the application and provides an easy way to handle dependency
// injection including within tests
type Container struct {
	// Validator stores a validator
	Validator *Validator

	// Web stores the web framework
	Web *echo.Echo

	Logger echo.Logger

	// Config stores the application configuration
	Config *config.Config

	// Cache contains the cache client
	Cache *CacheClient

	// Database stores the connection to the database
	Database    *sql.DB
	databaseDSN string

	// Mail stores an email sending client
	Mail *mailer.MailClient

	// Auth stores an authentication client
	Auth *AuthClient

	// AI stores the app-facing AI service.
	AI *ai.Service

	// Notifier handles all notifications to clients
	Notifier *notifications.NotifierService

	// CoreCache exposes cache via the backend-agnostic core seam.
	CoreCache core.Cache
	// CoreJobs exposes jobs via the backend-agnostic core seam.
	CoreJobs core.Jobs
	// CoreJobsInspector exposes jobs inspection via the backend-agnostic core seam.
	CoreJobsInspector core.JobsInspector
	// CorePubSub exposes pubsub via the backend-agnostic core seam.
	CorePubSub core.PubSub

	// Adapters stores resolved adapter selection/capabilities for runtime use.
	Adapters coreadapters.Resolved
}

// NewContainer creates and initializes a new Container
func NewContainer() *Container {
	c := new(Container)
	c.initConfig()
	c.validateAdapterPlan()
	c.initValidator()
	c.initWeb()
	c.initOptionalServices()
	c.initDatabase()
	c.initSchema()
	c.initAuth()
	c.initMail()
	c.initAI()
	c.initPaymentProcessor()
	// ship:container:start
	// ship:container:end
	c.initCoreAdapters()
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
	if c == nil || c.Config == nil {
		panic("invalid container state: nil config")
	}

	resolved, err := coreadapters.ResolveFromConfig(c.Config)
	if err != nil {
		panic(fmt.Sprintf("invalid adapter plan: %v", err))
	}
	c.Adapters = resolved
}

func (c *Container) initCoreAdapters() {
	c.CoreCache = NewCoreCacheAdapter(c.Cache)
	c.CoreJobs = NewCoreJobsAdapter(nil, c.Adapters.JobsCapabilities)
	c.CoreJobsInspector = NewCoreJobsInspectorAdapter(nil)
	c.CorePubSub = NewCorePubSubAdapter(nil)
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
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	c.Config = &cfg
}

// initValidator initializes the validator
func (c *Container) initValidator() {
	c.Validator = NewValidator()
}

// initWeb initializes the web framework
func (c *Container) initWeb() {

	c.Web = echo.New()
	if c.Config.App.Environment == config.EnvProduction {
		// TODO: Haven't set up sentry for GoShip yet
		// sentryDsn := c.Config.App.SentryDsn
		// if len(sentryDsn) == 0 {
		// 	log.Fatal().Str("app", string(c.Config.App.Name)).Msg("sentry initialization failed due to empty DSN")
		// }
		// // To initialize Sentry's handler, you need to initialize Sentry itself beforehand
		// if err := sentry.Init(sentry.ClientOptions{
		// 	Dsn: sentryDsn,
		// 	BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		// 		if event.Level == sentry.LevelError {
		// 			for _, exception := range event.Exception {
		// 				if exception.Type == "sentry.usageError" {
		// 					return nil // Ignore this error
		// 				}
		// 			}
		// 		}

		// 		if event.Message == "CaptureMessage called with empty message" {
		// 			return nil // Ignore this error
		// 		}

		// 		return event
		// 	},
		// 	// Set TracesSampleRate to 1.0 to capture 100%
		// 	// of transactions for performance monitoring.
		// 	// We recommend adjusting this value in production,
		// 	TracesSampleRate: 0.05, // For dev, because otherwise I use all my sentry errors in days
		// 	EnableTracing:    true,
		// 	Release:          "v0.1",
		// }); err != nil {
		// 	log.Fatal().Err(err).Msg("sentry initialization failed")
		// }
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
	if c.Cache, err = NewCacheClient(c.Config); err != nil {
		panic(err)
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
			// TODO: Drop/recreate the DB, if this isn't in memory?
			connection = c.Config.Database.EmbeddedTestConnection
		default:
			connection = c.Config.Database.EmbeddedConnection
		}

		c.Database, err = openEmbeddedDB(c.Config.Database.EmbeddedDriver, connection)
		if err != nil {
			panic(err)
		}
	} else {

		if c.Config.App.Environment == config.EnvProduction {
			c.Database, err = sql.Open("pgx", c.getProdDBAddr(c.Config.Database.DatabaseNameProd))
			if err != nil {
				panic(fmt.Sprintf("failed to connect to database: %v", err))
			}
		} else {
			c.Database, err = sql.Open("pgx", c.getDBAddr(c.Config.Database.DatabaseNameLocal))
			if err != nil {
				panic(fmt.Sprintf("failed to connect to database: %v", err))
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
				panic(fmt.Sprintf("failed to create test database: %v", err))
			}

			// Connect to the test database
			if err = c.Database.Close(); err != nil {
				panic(fmt.Sprintf("failed to close database connection: %v", err))
			}
			c.Database, err = sql.Open("pgx", c.getDBAddr(c.Config.Database.TestDatabase))
			if err != nil {
				panic(fmt.Sprintf("failed to connect to database: %v", err))
			}
		}
		// Create the pgvector extension
		createVectorExtension, err := dbqueries.Get("create_pgvector_extension_postgres")
		if err != nil {
			panic(fmt.Sprintf("failed to load pgvector extension query: %v", err))
		}
		_, err = c.Database.Exec(createVectorExtension)
		if err != nil {
			panic(fmt.Sprintf("failed to enable pgvector: %v", err))
		}
	}
}

// openEmbeddedDB opens a database connection
func openEmbeddedDB(driver, connection string) (*sql.DB, error) {
	// Helper to automatically create the directories that the specified sqlite file
	// should reside in, if one
	if isSQLiteDriver(driver) {
		d := strings.Split(connection, "/")

		if len(d) > 1 {
			path := strings.Join(d[:len(d)-1], "/")

			if err := os.MkdirAll(path, 0755); err != nil {
				return nil, err
			}
		}
	}

	return sql.Open(driver, connection)
}

// initSchema runs DB migrations for the app.
func (c *Container) initSchema() {
	migrationsDir, err := resolveMigrationsDir()
	if err != nil {
		panic(fmt.Sprintf("failed to resolve migrations directory: %v", err))
	}
	driver := "postgres"
	if c.Config.Database.DbMode == config.DBModeEmbedded {
		driver = normalizeSQLiteDriver(c.Config.Database.EmbeddedDriver)
	}
	if isSQLiteDriver(driver) {
		if err := ensureEmbeddedSQLiteSchema(c.Database); err != nil {
			panic(fmt.Sprintf("failed to initialize embedded sqlite schema: %v", err))
		}
		return
	}
	if err := applySQLMigrations(c.Database, migrationsDir, driver); err != nil {
		panic(fmt.Sprintf("failed to run SQL migrations: %v", err))
	}
}

// initAuth initializes the authentication client
func (c *Container) initAuth() {
	c.Auth = NewAuthClient(c.Config, selectAuthStore(c.Config, c.Database))
}

// initMail initialize the mail client
func (c *Container) initMail() {
	var err error
	var mailClientImplementation mailer.MailClientInterface

	if c.Config.App.Environment == config.EnvProduction {
		mailClientImplementation = mailer.NewResendMailClient(c.Config.Mail.ResendAPIKey)
	} else {
		mailClientImplementation = mailer.NewSMTPMailClient("localhost", int(c.Config.Mail.SmtpPort))
	}
	c.Mail, err = mailer.NewMailClient(c.Config, mailClientImplementation)
	if err != nil {
		panic(fmt.Sprintf("failed to create mail client: %v", err))
	}
}

func (c *Container) initAI() {
	var provider ai.Provider

	switch strings.ToLower(strings.TrimSpace(c.Config.AI.Driver)) {
	case "", "anthropic":
		if strings.TrimSpace(c.Config.AI.Anthropic.APIKey) == "" {
			provider = ai.NewUnavailableProvider("missing ANTHROPIC_API_KEY")
		} else {
			provider = anthropicdriver.New(c.Config.AI.Anthropic.APIKey, c.Config.AI.Anthropic.DefaultModel)
		}
	default:
		provider = ai.NewUnavailableProvider(fmt.Sprintf("unsupported AI driver %q", c.Config.AI.Driver))
	}

	module := ai.NewModule(ai.NewService(provider, logging.NewLogger(c.Config.Log)))
	c.AI = module.Service()
}

func (c *Container) initPaymentProcessor() {
	stripe.Key = c.Config.App.PrivateStripeKey
}
