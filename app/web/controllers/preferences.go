package controllers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/routenames"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	viewmodels "github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"
	"github.com/leomorpho/goship/framework/runtimeconfig"

	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	profilesvc "github.com/leomorpho/goship/modules/profile"
)

type (
	profilePrefsRoute struct {
		ctr            ui.Controller
		profileService *profilesvc.ProfileService
	}

	profileBioFormData struct {
		Bio        string `form:"bio" validate:"required"`
		Submission ui.FormSubmission
	}
)

func NewProfilePrefsRoute(ctr ui.Controller, profileService *profilesvc.ProfileService) profilePrefsRoute {
	return profilePrefsRoute{
		ctr:            ctr,
		profileService: profileService,
	}
}

func (p *profilePrefsRoute) GetBio(ctx echo.Context) error {
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	prof, err := p.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.AboutMe(&page)
	page.Name = templates.PagePreferences

	form := viewmodels.NewProfileBioFormData()
	form.Bio = prof.Bio
	page.Form = form

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.ProfileBioFormData)
	}

	return p.ctr.RenderPage(ctx, page)
}

func (p *profilePrefsRoute) UpdateBio(ctx echo.Context) error {
	// Create a new instance of geolocationPoint to hold the incoming data
	var profileBioData viewmodels.ProfileBioFormData
	ctx.Set(context.FormKey, &profileBioData)

	if err := ctx.Bind(&profileBioData); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid bio data")
	}

	if err := profileBioData.Submission.Process(ctx, profileBioData); err != nil {
		return p.ctr.Fail(err, "unable to process form submission")
	}
	if profileBioData.Submission.HasErrors() {
		return p.GetBio(ctx)
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if err := p.profileService.UpdateProfileBio(ctx.Request().Context(), profileID, profileBioData.Bio); err != nil {
		return err
	}

	return p.GetBio(ctx)
}

type preferences struct {
	ctr                           ui.Controller
	profileService                profilesvc.ProfileService
	pushNotificationsRepo         *notifications.PwaPushService
	notificationPermissionService *notifications.NotificationPermissionService
	subscriptionsService          *paidsubscriptions.Service
	smsSenderService              *notifications.SMSSender
}

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

func (g *preferences) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.Settings(&page)
	page.Name = templates.PagePreferences

	var data viewmodels.PreferencesData
	var err error

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	profile, err := g.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	data, err = g.getCurrPreferencesData(ctx)

	if err != nil {
		return err
	}

	subscribedEndpoints, err := g.pushNotificationsRepo.GetPushSubscriptionEndpoints(ctx.Request().Context(), profile.ID)
	if err != nil {
		return err
	}

	addPushSubscriptionEndpoint := fmt.Sprintf("%s%s",
		g.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameRegisterSubscription, domain.NotificationPlatformPush.Value)) + "?csrf=" + page.CSRF
	deletePushSubscriptionEndpoint := fmt.Sprintf("%s%s",
		g.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameDeleteSubscription, domain.NotificationPlatformPush.Value)) + "?csrf=" + page.CSRF

	addFCMPushSubscriptionEndpoint := fmt.Sprintf("%s%s",
		g.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameRegisterSubscription, domain.NotificationPlatformFCMPush.Value)) + "?csrf=" + page.CSRF
	deleteFCMPushSubscriptionEndpoint := fmt.Sprintf("%s%s",
		g.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameDeleteSubscription, domain.NotificationPlatformFCMPush.Value)) + "?csrf=" + page.CSRF

	addEmailSubscriptionEndpoint := fmt.Sprintf("%s%s",
		g.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameRegisterSubscription, domain.NotificationPlatformEmail.Value)) + "?csrf=" + page.CSRF
	deleteEmailSubscriptionEndpoint := fmt.Sprintf("%s%s",
		g.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameDeleteSubscription, domain.NotificationPlatformEmail.Value)) + "?csrf=" + page.CSRF

	addSmsSubscriptionEndpoint := fmt.Sprintf("%s%s",
		g.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameRegisterSubscription, domain.NotificationPlatformSMS.Value)) + "?csrf=" + page.CSRF
	deleteSmsSubscriptionEndpoint := fmt.Sprintf("%s%s",
		g.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameDeleteSubscription, domain.NotificationPlatformSMS.Value)) + "?csrf=" + page.CSRF

	permissions, err := g.notificationPermissionService.GetPermissions(ctx.Request().Context(), profile.ID)
	if err != nil {
		return err
	}

	notificationPermissions := viewmodels.NewNotificationPermissionsData()
	notificationPermissions.VapidPublicKey = g.ctr.Container.Config.App.VapidPublicKey
	notificationPermissions.PermissionDailyNotif = permissions[domain.NotificationPermissionDailyReminder]
	notificationPermissions.PermissionPartnerActivity = permissions[domain.NotificationPermissionNewFriendActivity]
	notificationPermissions.SubscribedEndpoints = subscribedEndpoints
	notificationPermissions.PhoneSubscriptionEnabled = profile.PhoneNumberE164 != "" && profile.PhoneVerified
	notificationPermissions.NotificationTypeQueryParamKey = domain.PermissionNotificationType
	notificationPermissions.AddPushSubscriptionEndpoint = addPushSubscriptionEndpoint
	notificationPermissions.DeletePushSubscriptionEndpoint = deletePushSubscriptionEndpoint
	notificationPermissions.AddFCMPushSubscriptionEndpoint = addFCMPushSubscriptionEndpoint
	notificationPermissions.DeleteFCMPushSubscriptionEndpoint = deleteFCMPushSubscriptionEndpoint
	notificationPermissions.AddEmailSubscriptionEndpoint = addEmailSubscriptionEndpoint
	notificationPermissions.DeleteEmailSubscriptionEndpoint = deleteEmailSubscriptionEndpoint
	notificationPermissions.AddSmsSubscriptionEndpoint = addSmsSubscriptionEndpoint
	notificationPermissions.DeleteSmsSubscriptionEndpoint = deleteSmsSubscriptionEndpoint

	data.NotificationPermissionsData = notificationPermissions

	page.Data = data
	page.HTMX.Request.Boosted = true

	if page.IsFullyOnboarded {
		page.ShowBottomNavbar = true
		page.SelectedBottomNavbarItem = domain.BottomNavbarItemSettings
	}

	return g.ctr.RenderPage(ctx, page)
}

func (g *preferences) getCurrPreferencesData(ctx echo.Context) (viewmodels.PreferencesData, error) {
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return viewmodels.NewPreferencesData(), err
	}
	profile, err := g.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return viewmodels.NewPreferencesData(), err
	}

	// Make sure to check if birthdate is non-nil
	birthdateStr := profile.Birthdate.UTC().Format("2006-01-02")

	activePlan, subscriptionExpiredOn, isTrial, err := g.subscriptionsService.GetCurrentlyActiveProduct(
		ctx.Request().Context(), profile.ID,
	)

	if err != nil {
		return viewmodels.NewPreferencesData(), err
	}
	activePlanKey := activePlanKey(activePlan)

	data := viewmodels.NewPreferencesData()
	data.Bio = profile.Bio
	data.PhoneNumberInE164Format = profile.PhoneNumberE164
	data.CountryCode = profile.CountryCode
	data.SelfBirthdate = birthdateStr
	data.IsProfileFullyOnboarded = profile.FullyOnboarded
	data.DefaultBio = domain.DefaultBio
	data.DefaultBirthdate = domain.DefaultBirthdate.Format("2006-01-02")
	data.IsPaymentsEnabled = g.ctr.Container.Config.App.OperationalConstants.PaymentsEnabled
	data.ActiveSubscriptionPlanKey = activePlanKey
	data.ActiveSubscriptionPlanIsPaid = isPaidPlanKey(activePlanKey)
	data.IsTrial = isTrial
	data.ManagedMode = g.ctr.Container.Config.Managed.RuntimeReport.Mode == runtimeconfig.ModeManaged
	data.ManagedAuthority = g.ctr.Container.Config.Managed.RuntimeReport.Authority
	data.ManagedSettings = managedSettingsViewData(g.ctr.Container.Config)

	if subscriptionExpiredOn != nil {
		data.HasMonthlySubscriptionExpiry = true
		data.MonthlySybscriptionExpiration = subscriptionExpiredOn.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	return data, nil
}

func managedSettingsViewData(cfg *config.Config) []viewmodels.ManagedSettingControl {
	if cfg == nil {
		return []viewmodels.ManagedSettingControl{}
	}

	statuses := cfg.ManagedSettingStatuses()
	controls := make([]viewmodels.ManagedSettingControl, 0, len(statuses))
	for _, status := range statuses {
		control := viewmodels.NewManagedSettingControl()
		control.Key = status.Key
		control.Label = status.Label
		control.Value = status.Value
		control.Source = string(status.Source)
		control.Access = string(status.Access)
		controls = append(controls, control)
	}
	return controls
}

func (p *preferences) GetPhoneComponent(ctx echo.Context) error {
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	profile, err := p.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.EditPhonePage(&page)
	page.Name = templates.PagePhoneNumber
	page.HTMX.Request.Boosted = true

	data := viewmodels.NewPhoneNumber()
	data.CountryCode = profile.CountryCode
	data.PhoneNumberE164 = profile.PhoneNumberE164
	data.PhoneVerified = profile.PhoneVerified
	page.Data = data

	return p.ctr.RenderPage(ctx, page)
}

func (p *preferences) GetPhoneVerificationComponent(ctx echo.Context) error {
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	profile, err := p.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PagePhoneNumber
	page.Form = viewmodels.NewPhoneNumberVerification()
	page.Component = pages.PhoneVerificationField(&page)
	data := viewmodels.NewSmsVerificationCodeInfo()
	data.ExpirationInMinutes = p.ctr.Container.Config.Phone.ValidationCodeExpirationMinutes
	page.Data = data

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.PhoneNumberVerification)
	}

	_, err = p.smsSenderService.CreateConfirmationCode(ctx.Request().Context(), profile.ID, profile.PhoneNumberE164)
	if err != nil {
		slog.Error("failed to send verification code", "error", err)
		uxflashmessages.Danger(ctx, "Failed to send verification code 😨")
		return p.ctr.RenderPage(ctx, page)
	}

	return p.ctr.RenderPage(ctx, page)
}

func (p *preferences) SubmitPhoneVerificationCode(ctx echo.Context) error {

	var form viewmodels.PhoneNumberVerification
	ctx.Set(context.FormKey, &form)

	// Parse the form values
	if err := ctx.Bind(&form); err != nil {
		return p.ctr.Fail(err, "unable to parse verification code form")
	}

	if err := form.Submission.Process(ctx, form); err != nil {
		return p.ctr.Fail(err, "unable to process form submission")
	}

	if form.Submission.HasErrors() {
		return p.GetPhoneVerificationComponent(ctx)
	}

	if form.VerificationCode == "" {
		form.Submission.SetFieldError("VerificationCode", "Invalid code")
		uxflashmessages.Danger(ctx, "Invalid code. Please try again.")
		return p.GetPhoneVerificationComponent(ctx)
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	profile, err := p.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	valid, err := p.smsSenderService.VerifyConfirmationCode(ctx.Request().Context(), profile.ID, form.VerificationCode)
	if err != nil || !valid {

		form.Submission.SetFieldError("VerificationCode", "Invalid code")
		uxflashmessages.Danger(ctx, "Invalid code. Please try again.")
		return p.GetPhoneVerificationComponent(ctx)
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PagePhoneNumber
	page.Form = viewmodels.NewPhoneNumberVerification()
	page.Component = pages.PhoneVerificationField(&page)

	uxflashmessages.Success(ctx, "Success! Your phone number was confirmed.")

	return p.GetPhoneVerificationComponent(ctx)
}

type phoneNumberFormData struct {
	PhoneNumberE164Format string `form:"phone_number_e164" validate:"required"`
	CountryCode           string `form:"country_code" validate:"required"`
	Submission            ui.FormSubmission
}

func (p *preferences) SavePhoneInfo(ctx echo.Context) error {
	// Create a new instance of geolocationPoint to hold the incoming data
	var phoneNumberFormData phoneNumberFormData
	ctx.Set(context.FormKey, &phoneNumberFormData)

	if err := ctx.Bind(&phoneNumberFormData); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid bio data")
	}

	if err := phoneNumberFormData.Submission.Process(ctx, phoneNumberFormData); err != nil {
		return p.ctr.Fail(err, "unable to process form submission")
	}

	if phoneNumberFormData.Submission.HasErrors() {
		return p.ctr.Redirect(ctx, "preferences")
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	return p.profileService.UpdateProfilePhone(
		ctx.Request().Context(),
		profileID,
		phoneNumberFormData.CountryCode,
		phoneNumberFormData.PhoneNumberE164Format,
	)
}

func (p *preferences) GetDisplayName(ctx echo.Context) error {
	userIDRaw := ctx.Get(context.AuthenticatedUserIDKey)
	userID, ok := userIDRaw.(int)
	if !ok || userID <= 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "authenticated user id missing from context")
	}
	displayName, err := p.ctr.Container.Auth.GetUserDisplayNameByUserID(ctx, userID)
	if err != nil {
		return p.ctr.Fail(err, "unable to load display name")
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.DisplayName(&page)
	page.Name = templates.PageDisplayName
	form := viewmodels.NewDisplayNameForm()
	form.DisplayName = displayName
	page.Form = form

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.DisplayNameForm)
	}

	return p.ctr.RenderPage(ctx, page)
}

func (p *preferences) SaveDisplayName(ctx echo.Context) error {
	// Create a new instance of geolocationPoint to hold the incoming data
	var displayNameFormData viewmodels.DisplayNameForm
	ctx.Set(context.FormKey, &displayNameFormData)

	if err := ctx.Bind(&displayNameFormData); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid display name data")
	}

	if err := displayNameFormData.Submission.Process(ctx, displayNameFormData); err != nil {
		return p.ctr.Fail(err, "unable to process form submission")
	}

	if displayNameFormData.Submission.HasErrors() {
		return p.GetDisplayName(ctx)
	}

	userIDRaw := ctx.Get(context.AuthenticatedUserIDKey)
	userID, ok := userIDRaw.(int)
	if !ok || userID <= 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "authenticated user id missing from context")
	}

	if err := p.ctr.Container.Auth.SetUserDisplayNameByUserID(ctx, userID, displayNameFormData.DisplayName); err != nil {
		return err
	}

	return p.GetDisplayName(ctx)
}

type onboarding struct {
	ctr            ui.Controller
	profileService *profilesvc.ProfileService
}

func NewOnboardingRoute(ctr ui.Controller, profileService *profilesvc.ProfileService) onboarding {
	return onboarding{
		ctr:            ctr,
		profileService: profileService,
	}
}

func (p *onboarding) Get(ctx echo.Context) error {
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if err := p.profileService.MarkProfileFullyOnboarded(ctx.Request().Context(), profileID); err != nil {
		return err
	}

	return p.ctr.RedirectWithDetails(ctx, routenames.RouteNameHomeFeed, "?just_finished_onboarding=true", http.StatusFound)
}
