package bootstrap

import (
	"fmt"

	notifications "github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
)

type FirstPartyServices struct {
	PaidSubscriptions *paidsubscriptions.Service
	Notifications     *notifications.Services
}

// BuildFirstPartyServices centralizes first-party battery/runtime composition
// shared by web, worker, and test bootstrap paths.
func BuildFirstPartyServices(c *Container, process JobsProcess) (FirstPartyServices, error) {
	if c == nil {
		return FirstPartyServices{}, fmt.Errorf("missing runtime container")
	}

	plansCatalog, err := paidsubscriptions.BuildDefaultCatalog()
	if err != nil {
		return FirstPartyServices{}, err
	}
	paidSubscriptionsService := paidsubscriptions.NewServiceWithCatalog(paidsubscriptions.NewSQLStore(
		c.Database,
		c.Config.Adapters.DB,
		c.Config.App.OperationalConstants.ProTrialTimespanInDays,
		c.Config.App.OperationalConstants.PaymentFailedGracePeriodInDays,
	), plansCatalog)

	runtime, err := WireJobsRuntime(c.Config, c.Database, process)
	if err != nil {
		return FirstPartyServices{}, err
	}
	c.CoreJobs = runtime.Jobs
	c.CoreJobsInspector = runtime.Inspector

	var firebaseJSONAccessKeys *[]byte
	if len(c.Config.App.FirebaseJSONAccessKeys) > 0 {
		firebaseJSONAccessKeys = &c.Config.App.FirebaseJSONAccessKeys
	}
	notificationServices, err := notifications.New(notifications.RuntimeDeps{
		DB:                                 c.Database,
		DBDialect:                          c.Config.Adapters.DB,
		PubSub:                             AdaptNotificationsPubSub(c.CorePubSub),
		Jobs:                               AdaptNotificationsJobs(c.CoreJobs),
		SubscriptionService:                paidSubscriptionsService,
		VapidPublicKey:                     c.Config.App.VapidPublicKey,
		VapidPrivateKey:                    c.Config.App.VapidPrivateKey,
		MailFromAddress:                    c.Config.Mail.FromAddress,
		FirebaseJSONAccessKeys:             firebaseJSONAccessKeys,
		SMSRegion:                          c.Config.Phone.Region,
		SMSSenderID:                        c.Config.Phone.SenderID,
		SMSValidationCodeExpirationMinutes: c.Config.Phone.ValidationCodeExpirationMinutes,
	})
	if err != nil {
		return FirstPartyServices{}, err
	}

	return FirstPartyServices{
		PaidSubscriptions: paidSubscriptionsService,
		Notifications:     notificationServices,
	}, nil
}
