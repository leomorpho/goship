package controllers

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/runtimeconfig"
	frameworkauthcontext "github.com/leomorpho/goship/framework/web/authcontext"
	layouts "github.com/leomorpho/goship/framework/web/layouts/gen"
	pages "github.com/leomorpho/goship/framework/web/pages/gen"
	routeNames "github.com/leomorpho/goship/framework/web/routenames"
	"github.com/leomorpho/goship/framework/web/templates"
	"github.com/leomorpho/goship/framework/web/ui"
	viewmodels "github.com/leomorpho/goship/framework/web/viewmodels"

	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	profilesvc "github.com/leomorpho/goship/modules/profile"
)

func NewPreferencesRoute(
	ctr ui.Controller,
	profileService *profilesvc.ProfileService,
	pushNotificationsRepo *notifications.PwaPushService,
	notificationPermissionService *notifications.NotificationPermissionService,
	subscriptionsService *paidsubscriptions.Service,
	smsSenderService *notifications.SMSSender,
) preferences {
	return preferences{
		ctr:                           ctr,
		profileService:                *profileService,
		pushNotificationsRepo:         pushNotificationsRepo,
		notificationPermissionService: notificationPermissionService,
		subscriptionsService:          subscriptionsService,
		smsSenderService:              smsSenderService,
	}
}

func (p *preferences) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.Settings(&page)
	page.Name = templates.PagePreferences

	data, err := p.currentPreferencesData(ctx)
	if err != nil {
		return err
	}

	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	profile, err := p.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	subscribedEndpoints, err := p.pushNotificationsRepo.GetPushSubscriptionEndpoints(ctx.Request().Context(), profile.ID)
	if err != nil {
		return err
	}

	permissions, err := p.notificationPermissionService.GetPermissions(ctx.Request().Context(), profile.ID)
	if err != nil {
		return err
	}

	notificationPermissions := viewmodels.NewNotificationPermissionsData()
	notificationPermissions.VapidPublicKey = p.ctr.Container.Config.App.VapidPublicKey
	notificationPermissions.PermissionDailyNotif = permissions[notifications.PermissionDailyReminder]
	notificationPermissions.PermissionPartnerActivity = permissions[notifications.PermissionNewFriendActivity]
	notificationPermissions.SubscribedEndpoints = subscribedEndpoints
	notificationPermissions.PhoneSubscriptionEnabled = profile.PhoneNumberE164 != "" && profile.PhoneVerified
	notificationPermissions.NotificationTypeQueryParamKey = domain.PermissionNotificationType
	notificationPermissions.AddPushSubscriptionEndpoint = p.subscriptionEndpoint(ctx, &page, routeNames.RouteNameRegisterSubscription, notifications.PlatformPWAPush.Value)
	notificationPermissions.DeletePushSubscriptionEndpoint = p.subscriptionEndpoint(ctx, &page, routeNames.RouteNameDeleteSubscription, notifications.PlatformPWAPush.Value)
	notificationPermissions.AddFCMPushSubscriptionEndpoint = p.subscriptionEndpoint(ctx, &page, routeNames.RouteNameRegisterSubscription, notifications.PlatformFCMPush.Value)
	notificationPermissions.DeleteFCMPushSubscriptionEndpoint = p.subscriptionEndpoint(ctx, &page, routeNames.RouteNameDeleteSubscription, notifications.PlatformFCMPush.Value)
	notificationPermissions.AddEmailSubscriptionEndpoint = p.subscriptionEndpoint(ctx, &page, routeNames.RouteNameRegisterSubscription, notifications.PlatformEmail.Value)
	notificationPermissions.DeleteEmailSubscriptionEndpoint = p.subscriptionEndpoint(ctx, &page, routeNames.RouteNameDeleteSubscription, notifications.PlatformEmail.Value)
	notificationPermissions.AddSmsSubscriptionEndpoint = p.subscriptionEndpoint(ctx, &page, routeNames.RouteNameRegisterSubscription, notifications.PlatformSMS.Value)
	notificationPermissions.DeleteSmsSubscriptionEndpoint = p.subscriptionEndpoint(ctx, &page, routeNames.RouteNameDeleteSubscription, notifications.PlatformSMS.Value)

	data.NotificationPermissionsData = notificationPermissions

	page.Data = data
	page.HTMX.Request.Boosted = true

	if page.IsFullyOnboarded {
		page.ShowBottomNavbar = true
		page.SelectedBottomNavbarItem = domain.BottomNavbarItemSettings
	}

	return p.ctr.RenderPage(ctx, page)
}

func (p *preferences) currentPreferencesData(ctx echo.Context) (viewmodels.PreferencesData, error) {
	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return viewmodels.NewPreferencesData(), err
	}
	profile, err := p.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return viewmodels.NewPreferencesData(), err
	}

	birthdateStr := profile.Birthdate.UTC().Format("2006-01-02")
	activePlan, subscriptionExpiredOn, isTrial, err := p.subscriptionsService.GetCurrentlyActiveProduct(ctx.Request().Context(), profile.ID)
	if err != nil {
		return viewmodels.NewPreferencesData(), err
	}

	activePlanKey := p.subscriptionsService.ActivePlanKey(activePlan)

	data := viewmodels.NewPreferencesData()
	data.Bio = profile.Bio
	data.PhoneNumberInE164Format = profile.PhoneNumberE164
	data.CountryCode = profile.CountryCode
	data.SelfBirthdate = birthdateStr
	data.IsProfileFullyOnboarded = profile.FullyOnboarded
	data.DefaultBio = domain.DefaultBio
	data.DefaultBirthdate = domain.DefaultBirthdate.Format("2006-01-02")
	data.IsPaymentsEnabled = p.ctr.Container.Config.App.OperationalConstants.PaymentsEnabled
	data.ActiveSubscriptionPlanKey = activePlanKey
	data.ActiveSubscriptionPlanIsPaid = p.subscriptionsService.IsPaidPlanKey(activePlanKey)
	data.IsTrial = isTrial
	data.ManagedMode = p.ctr.Container.Config.Managed.RuntimeReport.Mode == runtimeconfig.ModeManaged
	data.ManagedAuthority = p.ctr.Container.Config.Managed.RuntimeReport.Authority
	data.ManagedSettings = viewmodels.ManagedSettingsFromConfig(p.ctr.Container.Config)

	if subscriptionExpiredOn != nil {
		data.HasMonthlySubscriptionExpiry = true
		data.MonthlySybscriptionExpiration = subscriptionExpiredOn.Format("2006-01-02T15:04:05.999999999Z07:00")
	}

	return data, nil
}

func (p *preferences) subscriptionEndpoint(ctx echo.Context, page *ui.Page, routeName string, platform string) string {
	return fmt.Sprintf("%s%s", p.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(routeName, platform)) + "?csrf=" + page.CSRF
}
