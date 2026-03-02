package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"os"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"

	atlas "ariga.io/atlas/sql/schema"
	"github.com/getsentry/sentry-go"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v78"
	"github.com/ziflex/lecho/v3"

	"github.com/mikestefanello/pagoda/config"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/pkg/repos/mailer"
	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	"github.com/mikestefanello/pagoda/pkg/repos/pubsub"
	storagerepo "github.com/mikestefanello/pagoda/pkg/repos/storage"

	// Required by ent
	"github.com/mikestefanello/pagoda/ent/migrate"
	_ "github.com/mikestefanello/pagoda/ent/runtime"
	"github.com/mikestefanello/pagoda/ent/user"
)

type SentryHook struct{}

func (h *SentryHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if level >= zerolog.ErrorLevel {
		// Optionally, you can add more context to the Sentry event here
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetLevel(sentry.LevelError)
			scope.SetTag("logger", "zerolog")
			sentry.CaptureMessage(msg)
		})
	}
}

// Container contains all services used by the application and provides an easy way to handle dependency
// injection including within tests
type Container struct {
	// Validator stores a validator
	Validator *Validator

	// Web stores the web framework
	Web *echo.Echo

	Logger *lecho.Logger

	// Config stores the application configuration
	Config *config.Config

	// Cache contains the cache client
	Cache *CacheClient

	// Database stores the connection to the database
	Database    *sql.DB
	databaseDSN string

	// ORM stores a client to the ORM
	ORM *ent.Client

	// Mail stores an email sending client
	Mail *mailer.MailClient

	// Auth stores an authentication client
	Auth *AuthClient

	// Notifier handles all notifications to clients
	Notifier *notifierrepo.NotifierRepo

	// Tasks stores the task client
	Tasks *TaskClient
}

// NewContainer creates and initializes a new Container
func NewContainer() *Container {
	c := new(Container)
	c.initConfig()
	c.initValidator()
	c.initWeb()
	// c.initCache()
	c.initDatabase()
	c.initORM()
	c.initAuth()
	// c.initNotifier()
	c.initMail()
	c.initPaymentProcessor()
	// c.initTasks()
	return c
}

// Shutdown shuts the Container down and disconnects all connections
func (c *Container) Shutdown() error {
	if c.Tasks != nil {
		if err := c.Tasks.Close(); err != nil {
			return err
		}
	}
	if c.Cache != nil {
		if err := c.Cache.Close(); err != nil {
			return err
		}
	}
	if c.ORM != nil {
		if err := c.ORM.Close(); err != nil {
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

	// Create a zerolog logger instance
	zerologLogger := zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

	// Add the SentryHook
	// zerologLogger = zerologLogger.Hook(&SentryHook{})

	// Configure logging
	switch c.Config.App.Environment {
	case config.EnvProduction:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	}
	c.Logger = lecho.From(zerologLogger)
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
			// Drop the test database, ignoring errors in case it doesn't yet exist
			_, _ = c.Database.Exec("DROP DATABASE " + c.Config.Database.TestDatabase)

			// Create the test database
			if _, err = c.Database.Exec("CREATE DATABASE " + c.Config.Database.TestDatabase); err != nil {
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
		_, err = c.Database.Exec("CREATE EXTENSION IF NOT EXISTS vector")
		if err != nil {
			panic(fmt.Sprintf("failed to enable pgvector: %v", err))
		}
	}
}

// openEmbeddedDB opens a database connection
func openEmbeddedDB(driver, connection string) (*sql.DB, error) {
	// Helper to automatically create the directories that the specified sqlite file
	// should reside in, if one
	if driver == "sqlite3" {
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

// initORM initializes the ORM
func (c *Container) initORM() {
	var activeDialect string
	if c.Config.Database.DbMode == config.DBModeEmbedded {
		activeDialect = c.Config.Database.EmbeddedDriver
	} else {
		activeDialect = dialect.Postgres
	}

	drv := entsql.OpenDB(activeDialect, c.Database)
	c.ORM = ent.NewClient(ent.Driver(drv))
	if err := c.ORM.Schema.Create(
		context.Background(),
		schema.WithDiffHook(renameColumnHook),
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true),
	); err != nil {
		panic(fmt.Sprintf("failed to create database schema: %v", err))
	}
}

// TODO: not quite sure why I added the below ent hook. Might be removable for streamlining.
func renameColumnHook(next schema.Differ) schema.Differ {
	return schema.DiffFunc(func(current, desired *atlas.Schema) ([]atlas.Change, error) {
		changes, err := next.Diff(current, desired)
		if err != nil {
			return nil, err
		}
		for _, c := range changes {
			m, ok := c.(*atlas.ModifyTable)
			// Skip if the change is not a ModifyTable,
			// or if the table is not the "users" table.
			if !ok || m.T.Name != user.Table {
				continue
			}
			changes := atlas.Changes(m.Changes)
			switch i, j := changes.IndexDropColumn("old_name"), changes.IndexAddColumn("new_name"); {
			case i != -1 && j != -1:
				// Append a new renaming change.
				changes = append(changes, &atlas.RenameColumn{
					From: changes[i].(*atlas.DropColumn).C,
					To:   changes[j].(*atlas.AddColumn).C,
				})
				// Remove the drop and add changes.
				changes.RemoveIndex(i, j)
				m.Changes = changes
			case i != -1 || j != -1:
				return nil, errors.New("old_name and new_name must be present or absent")
			}
		}
		return changes, nil
	})
}

// initAuth initializes the authentication client
func (c *Container) initAuth() {
	c.Auth = NewAuthClient(c.Config, c.ORM)
}

func (c *Container) initNotifier() {
	pubsubRepo := pubsub.NewRedisPubSubClient(c.Cache.Client)
	notificationStorageRepo := notifierrepo.NewNotificationStorageRepo(c.ORM)
	pwaPushNotificationsRepo := notifierrepo.NewPwaPushNotificationsRepo(
		c.ORM, c.Config.App.VapidPublicKey, c.Config.App.VapidPrivateKey, c.Config.Mail.FromAddress,
	)
	fcmPushNotificationsRepo, err := notifierrepo.NewFcmPushNotificationsRepo(
		c.ORM, &c.Config.App.FirebaseJSONAccessKeys)
	if err != nil {
		log.Fatal().Err(err)
	}
	storageRepo := storagerepo.NewStorageClient(c.Config, c.ORM)
	profileRepo := *profilerepo.NewProfileRepo(c.ORM, storageRepo, nil)
	c.Notifier = notifierrepo.NewNotifierRepo(
		pubsubRepo, notificationStorageRepo, pwaPushNotificationsRepo, fcmPushNotificationsRepo, profileRepo.GetCountOfUnseenNotifications)
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

func (c *Container) initPaymentProcessor() {
	stripe.Key = c.Config.App.PrivateStripeKey
}

// initTasks initializes the task client
func (c *Container) initTasks() {
	c.Tasks = NewTaskClient(c.Config)
}
