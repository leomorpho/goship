package controllers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/web/routenames"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/profile"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app/notifications"
	"github.com/leomorpho/goship/app/profiles"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/rs/zerolog/log"
)

type (
	profilePrefsRoute struct {
		ctr ui.Controller
		orm *ent.Client
	}

	profileBioFormData struct {
		Bio        string `form:"bio" validate:"required"`
		Submission ui.FormSubmission
	}
)

func NewProfilePrefsRoute(ctr ui.Controller, orm *ent.Client) profilePrefsRoute {
	return profilePrefsRoute{
		ctr: ctr,
		orm: orm,
	}
}

func (p *profilePrefsRoute) GetBio(ctx echo.Context) error {
	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	prof := usr.QueryProfile().Select(profile.FieldBio).FirstX(ctx.Request().Context())

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.AboutMe(&page)
	page.Name = templates.PagePreferences

	page.Form = &viewmodels.ProfileBioFormData{
		Bio: prof.Bio,
	}

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

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	_, err := p.orm.Profile.UpdateOneID(profile.ID).SetBio(profileBioData.Bio).Save(ctx.Request().Context())
	if err != nil {
		return err
	}

	return p.GetBio(ctx)
}

type preferences struct {
	ctr                            ui.Controller
	profileRepo                    profiles.ProfileRepo
	pushNotificationsRepo          *notifications.PwaPushNotificationsRepo
	notificationSendPermissionRepo *notifications.NotificationSendPermissionRepo
	subscriptionsRepo              *paidsubscriptions.SubscriptionsRepo
	smsSenderRepo                  *notifications.SMSSender
}

func NewPreferencesRoute(
	ctr ui.Controller,
	profileRepo *profiles.ProfileRepo,
	pushNotificationsRepo *notifications.PwaPushNotificationsRepo,
	notificationSendPermissionRepo *notifications.NotificationSendPermissionRepo,
	subscriptionsRepo *paidsubscriptions.SubscriptionsRepo,
	smsSenderRepo *notifications.SMSSender,
) preferences {
	return preferences{
		ctr:                            ctr,
		profileRepo:                    *profileRepo,
		pushNotificationsRepo:          pushNotificationsRepo,
		notificationSendPermissionRepo: notificationSendPermissionRepo,
		subscriptionsRepo:              subscriptionsRepo,
		smsSenderRepo:                  smsSenderRepo,
	}
}

func (g *preferences) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.Settings(&page)
	page.Name = templates.PagePreferences

	var data *viewmodels.PreferencesData
	var err error

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

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

	permissions, err := g.notificationSendPermissionRepo.GetPermissions(ctx.Request().Context(), profile.ID)
	if err != nil {
		return err
	}

	notificationPermissions := viewmodels.NotificationPermissionsData{
		VapidPublicKey:                g.ctr.Container.Config.App.VapidPublicKey,
		PermissionDailyNotif:          permissions[domain.NotificationPermissionDailyReminder],
		PermissionPartnerActivity:     permissions[domain.NotificationPermissionNewFriendActivity],
		SubscribedEndpoints:           subscribedEndpoints,
		PhoneSubscriptionEnabled:      profile.PhoneNumberE164 != "" && profile.PhoneVerified,
		NotificationTypeQueryParamKey: domain.PermissionNotificationType,

		AddPushSubscriptionEndpoint:    addPushSubscriptionEndpoint,
		DeletePushSubscriptionEndpoint: deletePushSubscriptionEndpoint,

		AddFCMPushSubscriptionEndpoint:    addFCMPushSubscriptionEndpoint,
		DeleteFCMPushSubscriptionEndpoint: deleteFCMPushSubscriptionEndpoint,

		AddEmailSubscriptionEndpoint:    addEmailSubscriptionEndpoint,
		DeleteEmailSubscriptionEndpoint: deleteEmailSubscriptionEndpoint,

		AddSmsSubscriptionEndpoint:    addSmsSubscriptionEndpoint,
		DeleteSmsSubscriptionEndpoint: deleteSmsSubscriptionEndpoint,
	}

	data.NotificationPermissionsData = notificationPermissions

	page.Data = data
	page.HTMX.Request.Boosted = true

	if page.IsFullyOnboarded {
		page.ShowBottomNavbar = true
		page.SelectedBottomNavbarItem = domain.BottomNavbarItemSettings
	}

	return g.ctr.RenderPage(ctx, page)
}

func (g *preferences) getCurrPreferencesData(ctx echo.Context) (*viewmodels.PreferencesData, error) {

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)

	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	// Make sure to check if birthdate is non-nil
	birthdateStr := profile.Birthdate.UTC().Format("2006-01-02")

	activePlan, subscriptionExpiredOn, isTrial, err := g.subscriptionsRepo.GetCurrentlyActiveProduct(
		ctx.Request().Context(), profile.ID,
	)

	if err != nil {
		return nil, err
	}

	data := &viewmodels.PreferencesData{
		Bio:                     profile.Bio,
		PhoneNumberInE164Format: profile.PhoneNumberE164,
		CountryCode:             profile.CountryCode,
		SelfBirthdate:           birthdateStr,
		IsProfileFullyOnboarded: profiles.IsProfileFullyOnboarded(profile),
		DefaultBio:              domain.DefaultBio,
		DefaultBirthdate:        domain.DefaultBirthdate.Format("2006-01-02"),

		// if IsPaymentsEnabled is true, none of the subscription stuff matters and the entire app will be free
		IsPaymentsEnabled:      g.ctr.Container.Config.App.OperationalConstants.PaymentsEnabled,
		ActiveSubscriptionPlan: *activePlan,
		IsTrial:                isTrial,
	}

	if subscriptionExpiredOn != nil {
		data.MonthlySybscriptionExpiration = subscriptionExpiredOn
	}
	return data, nil
}

func (p *preferences) GetPhoneComponent(ctx echo.Context) error {
	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.EditPhonePage(&page)
	page.Name = templates.PagePhoneNumber
	page.HTMX.Request.Boosted = true

	page.Data = &viewmodels.PhoneNumber{
		CountryCode:     profile.CountryCode,
		PhoneNumberE164: profile.PhoneNumberE164,
		PhoneVerified:   profile.PhoneVerified,
	}

	return p.ctr.RenderPage(ctx, page)
}

func (p *preferences) GetPhoneVerificationComponent(ctx echo.Context) error {
	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PagePhoneNumber
	page.Form = &viewmodels.PhoneNumberVerification{}
	page.Component = pages.PhoneVerificationField(&page)
	page.Data = &viewmodels.SmsVerificationCodeInfo{
		ExpirationInMinutes: p.ctr.Container.Config.Phone.ValidationCodeExpirationMinutes,
	}

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.PhoneNumberVerification)
	}

	_, err := p.smsSenderRepo.CreateConfirmationCode(ctx.Request().Context(), profile.ID, profile.PhoneNumberE164)
	if err != nil {
		log.Error().Err(err).Msg("failed to send verification code.")
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

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	valid, err := p.smsSenderRepo.VerifyConfirmationCode(ctx.Request().Context(), profile.ID, form.VerificationCode)
	if err != nil || !valid {

		form.Submission.SetFieldError("VerificationCode", "Invalid code")
		uxflashmessages.Danger(ctx, "Invalid code. Please try again.")
		return p.GetPhoneVerificationComponent(ctx)
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PagePhoneNumber
	page.Form = &viewmodels.PhoneNumberVerification{}
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

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	_, err := p.ctr.Container.ORM.Profile.
		UpdateOneID(profile.ID).
		SetCountryCode(phoneNumberFormData.CountryCode).
		SetPhoneNumberE164(phoneNumberFormData.PhoneNumberE164Format).
		Save(ctx.Request().Context())

	return err
}

func (p *preferences) GetDisplayName(ctx echo.Context) error {
	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.DisplayName(&page)
	page.Name = templates.PageDisplayName
	page.Form = &viewmodels.DisplayNameForm{
		DisplayName: usr.Name,
	}

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

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)

	_, err := p.ctr.Container.ORM.User.
		UpdateOneID(usr.ID).
		SetName(displayNameFormData.DisplayName).
		Save(ctx.Request().Context())

	if err != nil {
		return err
	}

	return p.GetDisplayName(ctx)
}

type onboarding struct {
	ctr ui.Controller
	orm *ent.Client
}

func NewOnboardingRoute(
	ctr ui.Controller, orm *ent.Client,
) onboarding {
	return onboarding{
		ctr: ctr,
		orm: orm,
	}
}

func (p *onboarding) Get(ctx echo.Context) error {
	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	_, err := p.orm.Profile.
		UpdateOneID(profile.ID).
		SetFullyOnboarded(true).
		Save(ctx.Request().Context())
	if err != nil {
		return err
	}

	return p.ctr.RedirectWithDetails(ctx, routenames.RouteNameHomeFeed, "?just_finished_onboarding=true", http.StatusFound)
}
