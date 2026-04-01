package notifications

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	paidsubscriptions "github.com/leomorpho/goship/v2-modules/paidsubscriptions"
	"github.com/leomorpho/goship/v2/framework/core"
)

const ModuleID = "notifications"

// Services bundles all notification-related services exposed by this installable module.
type Services struct {
	Notifier                    *NotifierService
	Permission                  *NotificationPermissionService
	PwaPush                     *PwaPushService
	FcmPush                     *FcmPushService
	SMSSender                   *SMSSender
	PlannedNotificationsService *PlannedNotificationsService
}

// RuntimeDeps are the composition-root inputs required to build module services.
type RuntimeDeps struct {
	InitContext                         context.Context
	DB                                  *sql.DB
	DBDialect                           string
	PubSub                              PubSub
	Jobs                                core.Jobs
	SubscriptionService                 *paidsubscriptions.Service
	VapidPublicKey                      string
	VapidPrivateKey                     string
	MailFromAddress                     string
	FirebaseJSONAccessKeys              *[]byte
	SMSRegion                           string
	SMSSenderID                         string
	SMSValidationCodeExpirationMinutes  int
	GetNumNotificationsForProfileByIDFn func(context.Context, int) (int, error)
}

// New constructs all notification services from explicit composition-root dependencies.
func New(deps RuntimeDeps) (*Services, error) {
	if deps.DB == nil {
		return nil, errors.New("notifications module requires DB")
	}
	var err error
	permissionService := NewSQLNotificationPermissionService(deps.DB, deps.DBDialect)
	plannedNotificationsService := NewSQLPlannedNotificationsService(deps.DB, deps.DBDialect, deps.SubscriptionService)
	var pwaPushService *PwaPushService
	var fcmPushService *FcmPushService
	pwaPushService = NewSQLPwaPushService(
		deps.DB,
		deps.DBDialect,
		deps.VapidPublicKey,
		deps.VapidPrivateKey,
		deps.MailFromAddress,
	)
	fcmPushService, err = NewSQLFcmPushService(
		deps.DB,
		deps.DBDialect,
		deps.FirebaseJSONAccessKeys,
	)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(deps.SMSRegion)
	if region == "" {
		region = "us-east-1"
	}
	initCtx := deps.InitContext
	if initCtx == nil {
		initCtx = context.Background()
	}
	smsSenderService, err := NewSQLSMSSender(
		initCtx,
		deps.DB,
		deps.DBDialect,
		region,
		deps.SMSSenderID,
		deps.SMSValidationCodeExpirationMinutes,
	)
	if err != nil {
		return nil, err
	}

	notificationStore, err := NewSQLNotificationStoreWithSchema(deps.DB, deps.DBDialect)
	if err != nil {
		return nil, err
	}
	getNumNotificationsForProfileByID := deps.GetNumNotificationsForProfileByIDFn
	if getNumNotificationsForProfileByID == nil {
		getNumNotificationsForProfileByID = notificationStore.GetCountOfUnseenNotifications
	}
	notifierService := NewNotifierService(
		deps.PubSub,
		deps.Jobs,
		notificationStore,
		permissionService,
		pwaPushService,
		fcmPushService,
		getNumNotificationsForProfileByID,
	)

	return &Services{
		Notifier:                    notifierService,
		Permission:                  permissionService,
		PwaPush:                     pwaPushService,
		FcmPush:                     fcmPushService,
		SMSSender:                   smsSenderService,
		PlannedNotificationsService: plannedNotificationsService,
	}, nil
}
