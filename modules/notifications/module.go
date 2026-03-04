package notifications

import (
	"context"
	"strings"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/framework/core"
)

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
	ORM                                 *ent.Client
	PubSub                              core.PubSub
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
	permissionService := NewNotificationPermissionService(deps.ORM)
	pwaPushService := NewPwaPushService(
		deps.ORM,
		deps.VapidPublicKey,
		deps.VapidPrivateKey,
		deps.MailFromAddress,
	)
	fcmPushService, err := NewFcmPushService(deps.ORM, deps.FirebaseJSONAccessKeys)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(deps.SMSRegion)
	if region == "" {
		region = "us-east-1"
	}
	smsSenderService, err := NewSMSSender(
		deps.ORM,
		region,
		deps.SMSSenderID,
		deps.SMSValidationCodeExpirationMinutes,
	)
	if err != nil {
		return nil, err
	}

	notificationStore := NewNotificationStore(deps.ORM)
	notifierService := NewNotifierService(
		deps.PubSub,
		notificationStore,
		pwaPushService,
		fcmPushService,
		deps.GetNumNotificationsForProfileByIDFn,
	)

	return &Services{
		Notifier:                    notifierService,
		Permission:                  permissionService,
		PwaPush:                     pwaPushService,
		FcmPush:                     fcmPushService,
		SMSSender:                   smsSenderService,
		PlannedNotificationsService: NewPlannedNotificationsService(deps.ORM, deps.SubscriptionService),
	}, nil
}
