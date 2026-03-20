package core

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	backupcontract "github.com/leomorpho/goship/framework/backup"
)

// TxFunc is executed inside a store transaction boundary.
type TxFunc func(ctx context.Context) error

// Store is the app-facing database boundary.
type Store interface {
	Ping(ctx context.Context) error
	WithTx(ctx context.Context, fn TxFunc) error
}

// Cache is the app-facing cache boundary.
type Cache interface {
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	InvalidatePrefix(ctx context.Context, prefix string) error
	Close() error
}

// MessageHandler processes pubsub payloads.
type MessageHandler func(ctx context.Context, topic string, payload []byte) error

// Subscription represents an active pubsub subscription.
type Subscription interface {
	Close() error
}

// PubSub is the app-facing pubsub boundary.
type PubSub interface {
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler) (Subscription, error)
	Close() error
}

// JobHandler processes an enqueued job payload.
type JobHandler func(ctx context.Context, payload []byte) error

// EnqueueOptions controls queue execution behavior.
type EnqueueOptions struct {
	Queue      string
	RunAt      time.Time
	MaxRetries int
	Timeout    time.Duration
	Retention  time.Duration
	Priority   int
}

// JobCapabilities declares backend-supported jobs features.
type JobCapabilities struct {
	Delayed    bool
	Retries    bool
	Cron       bool
	Priority   bool
	DeadLetter bool
	Dashboard  bool
}

type JobStatus string

const (
	JobStatusQueued  JobStatus = "queued"
	JobStatusRunning JobStatus = "running"
	JobStatusDone    JobStatus = "done"
	JobStatusFailed  JobStatus = "failed"
)

type JobRecord struct {
	ID         string
	Queue      string
	Name       string
	Payload    []byte
	Status     JobStatus
	Attempt    int
	MaxRetries int
	RunAt      time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastError  string
}

type JobListFilter struct {
	Queue    string
	Statuses []JobStatus
	Limit    int
	Offset   int
}

// Missing returns required capability names not supported by the current backend.
func (c JobCapabilities) Missing(required JobCapabilities) []string {
	missing := make([]string, 0, 6)
	if required.Delayed && !c.Delayed {
		missing = append(missing, "delayed")
	}
	if required.Retries && !c.Retries {
		missing = append(missing, "retries")
	}
	if required.Cron && !c.Cron {
		missing = append(missing, "cron")
	}
	if required.Priority && !c.Priority {
		missing = append(missing, "priority")
	}
	if required.DeadLetter && !c.DeadLetter {
		missing = append(missing, "dead_letter")
	}
	if required.Dashboard && !c.Dashboard {
		missing = append(missing, "dashboard")
	}
	return missing
}

// ValidateJobCapabilities fails fast when required job features are not available.
func ValidateJobCapabilities(required, available JobCapabilities) error {
	missing := available.Missing(required)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing required jobs capabilities: %s", strings.Join(missing, ", "))
}

// Jobs is the app-facing background jobs boundary.
type Jobs interface {
	Register(name string, handler JobHandler) error
	Enqueue(ctx context.Context, name string, payload []byte, opts EnqueueOptions) (jobID string, err error)
	StartWorker(ctx context.Context) error
	StartScheduler(ctx context.Context) error
	Stop(ctx context.Context) error
	Capabilities() JobCapabilities
}

type JobsInspector interface {
	List(ctx context.Context, filter JobListFilter) ([]JobRecord, error)
	Get(ctx context.Context, id string) (JobRecord, bool, error)
}

// I18n is the app-facing localization boundary.
type I18n interface {
	DefaultLanguage() string
	SupportedLanguages() []string
	NormalizeLanguage(raw string) string
	T(ctx context.Context, key string, templateData ...map[string]any) string
	TC(ctx context.Context, key string, count any, templateData ...map[string]any) string
	TS(ctx context.Context, key string, choice string, templateData ...map[string]any) string
}

// PutObjectInput describes an object upload request.
type PutObjectInput struct {
	Bucket      string
	Key         string
	ContentType string
	Reader      io.Reader
	Size        int64
	Metadata    map[string]string
}

// StoredObject describes an object written to blob storage.
type StoredObject struct {
	Bucket string
	Key    string
	ETag   string
	Size   int64
}

// BlobStorage is the app-facing object storage boundary.
type BlobStorage interface {
	Put(ctx context.Context, in PutObjectInput) (StoredObject, error)
	Delete(ctx context.Context, bucket, key string) error
	PresignGet(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
}

// BackupDriver creates framework backup manifests from runtime state.
type BackupDriver interface {
	Create(ctx context.Context, req backupcontract.CreateRequest) (backupcontract.Manifest, error)
}

// RestoreDriver validates and applies restore operations from backup manifests.
type RestoreDriver interface {
	Restore(ctx context.Context, req backupcontract.RestoreRequest) error
}

// MailAddress represents an email identity.
type MailAddress struct {
	Email string
	Name  string
}

// MailAttachment represents one attachment for a message.
type MailAttachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// MailMessage is the app-facing portable mail payload.
type MailMessage struct {
	From        MailAddress
	To          []MailAddress
	CC          []MailAddress
	BCC         []MailAddress
	ReplyTo     *MailAddress
	Subject     string
	TextBody    string
	HTMLBody    string
	Headers     map[string]string
	Attachments []MailAttachment
}

// Mailer is the app-facing mail delivery boundary.
type Mailer interface {
	Send(ctx context.Context, msg MailMessage) error
}

// Module is the installable framework module contract.
type Module interface {
	ID() string
	Migrations() fs.FS
}

// Router is the minimal Echo routing surface exposed to modules.
type Router interface {
	Group(prefix string, middleware ...echo.MiddlewareFunc) *echo.Group
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RoutableModule registers HTTP routes in addition to base module metadata.
type RoutableModule interface {
	Module
	RegisterRoutes(r Router) error
}
