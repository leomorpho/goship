package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/leomorpho/goship/app/web/routenames"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/dberrors"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"

	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	profilesvc "github.com/leomorpho/goship/app/profile"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/emails/gen"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
)

type (
	register struct {
		ctr                           ui.Controller
		profileService                profilesvc.ProfileService
		subscriptionsService          *paidsubscriptions.Service
		notificationPermissionService *notifications.NotificationPermissionService
	}
)

func NewRegisterRoute(
	ctr ui.Controller,
	profileService profilesvc.ProfileService,
	subscriptionsService *paidsubscriptions.Service,
	notificationPermissionService *notifications.NotificationPermissionService,
) register {
	return register{
		ctr:                           ctr,
		profileService:                profileService,
		subscriptionsService:          subscriptionsService,
		notificationPermissionService: notificationPermissionService,
	}
}

func (c *register) Get(ctx echo.Context) error {
	mode := ctx.QueryParam("mode")

	page := ui.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageRegister
	page.Component = pages.Register(&page)
	page.Title = "Register"
	page.Form = &viewmodels.RegisterForm{}

	// Get the current time
	currentTime := time.Now()
	// Subtract 18 years from the current time
	yearsAgo := currentTime.AddDate(-18, 0, 0)
	// Format the date as yyyy-mm-dd
	formattedDate := yearsAgo.Format("2006-01-02")

	page.Data = viewmodels.RegisterData{
		UserSignupEnabled:  c.ctr.Container.Config.App.OperationalConstants.UserSignupEnabled,
		RelationshipStatus: mode,
		MinDate:            formattedDate,
	}
	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.RegisterForm)
	}
	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}

func (c *register) Post(ctx echo.Context) error {
	var form viewmodels.RegisterForm
	ctx.Set(context.FormKey, &form)

	// Parse the form values
	if err := ctx.Bind(&form); err != nil {
		return c.ctr.Fail(err, "unable to parse register form")
	}

	if err := form.Submission.Process(ctx, form); err != nil {
		return c.ctr.Fail(err, "unable to process form submission")
	}

	if form.Submission.HasErrors() {
		return c.Get(ctx)
	}
	// Hash the password
	pwHash, err := c.ctr.Container.Auth.HashPassword(form.Password)
	if err != nil {
		return c.ctr.Fail(err, "unable to hash password")
	}

	// Convert Birthdate from string to time.Time
	layout := "2006-01-02" // This is the layout string for "YYYY-MM-DD"
	birthdate, err := time.ParseInLocation(layout, form.Birthdate, time.UTC)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid birthdate format")
	}
	// Convert to UTC
	birthdate = birthdate.UTC().Truncate(24 * time.Hour)

	// Check if user is at least 18 years old
	eighteenYearsAgo := time.Now().UTC().AddDate(-18, 0, 0)
	if birthdate.After(eighteenYearsAgo) {
		// User is younger than 18
		uxflashmessages.Warning(ctx, "You must be 18+ to register.")
		return c.Get(ctx) // Re-render the register page with the warning message
	}

	// Start a transaction
	tx, err := c.ctr.Container.ORM.Tx(ctx.Request().Context())
	if err != nil {
		return c.ctr.Fail(err, "failed to start transaction")
	}

	// Attempt creating the user
	u, err := tx.User.
		Create().
		SetName(form.Name).
		SetEmail(form.Email).
		SetPassword(pwHash).
		Save(ctx.Request().Context())

	if err != nil {
		tx.Rollback()
		switch {
		case dberrors.IsConstraint(err):
			uxflashmessages.Warning(ctx, "A user with this email address already exists. Please log in.")
			return c.ctr.Redirect(ctx, routeNames.RouteNameLogin)
		default:
			return c.ctr.Fail(err, "unable to create user")
		}
	}

	profile, err := tx.Profile.
		Create().
		SetUser(u).
		SetBio(domain.DefaultBio).
		SetBirthdate(birthdate).
		SetAge(profilesvc.CalculateAge(birthdate)).
		Save(ctx.Request().Context())

	if err != nil {
		tx.Rollback()
		ctx.Logger().Errorf("failed to create profile: %v", err)
		uxflashmessages.Info(ctx, "unable to create user")
		return c.ctr.Redirect(ctx, routenames.RouteNameLogin)
	}

	err = c.subscriptionsService.CreateSubscription(ctx.Request().Context(), tx, profile.ID)
	if err != nil {
		tx.Rollback()
		ctx.Logger().Errorf("failed to create trial pro subscription for profile: %v", err)
	}

	// If all operations were successful, commit the transaction
	err = tx.Commit()
	if err != nil {
		return c.ctr.Fail(err, "failed to commit transaction")
	}

	ctx.Logger().Infof("user and profile created successfully: %s", u.Name)

	for _, perm := range domain.NotificationPermissions.Members() {
		err := c.notificationPermissionService.CreatePermission(
			ctx.Request().Context(), profile.ID, perm, &domain.NotificationPlatformEmail)
		if err != nil {
			log.Error().Err(err).Int("profileID", profile.ID).Msg("failed to create notification permission")
		}
	}

	// Log the user in
	err = c.ctr.Container.Auth.Login(ctx, u.ID)
	if err != nil {
		ctx.Logger().Errorf("unable to log in: %v", err)
		uxflashmessages.Info(ctx, "Your account has been created.")
		return c.ctr.Redirect(ctx, routeNames.RouteNameLogin)
	}

	uxflashmessages.Success(ctx, "Your account has been created. You are now logged in. 👌")

	// Send the verification email
	c.sendVerificationEmail(ctx, u.Email)

	redirect, err := redirectAfterLogin(ctx)
	if err != nil {
		return err
	}
	if redirect {
		return nil
	}

	return c.ctr.Redirect(ctx, routeNames.RouteNamePreferences)
}

func (c *register) sendVerificationEmail(ctx echo.Context, userEmail string) {
	// Generate a token
	token, err := c.ctr.Container.Auth.GenerateEmailVerificationToken(userEmail)
	if err != nil {
		ctx.Logger().Errorf("unable to generate email verification token: %v", err)
		return
	}

	url := ctx.Echo().Reverse(routeNames.RouteNameVerifyEmail, token)
	fullUrl := fmt.Sprintf("%s%s", c.ctr.Container.Config.HTTP.Domain, url)

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Data = viewmodels.EmailDefaultData{
		AppName:          string(c.ctr.Container.Config.App.Name),
		ConfirmationLink: fullUrl,
		SupportEmail:     c.ctr.Container.Config.Mail.FromAddress,
		Domain:           c.ctr.Container.Config.HTTP.Domain,
	}

	err = c.ctr.Container.Mail.
		Compose().
		To(userEmail).
		Subject("Confirm your email address").
		TemplateLayout(layouts.Email).
		Component(emails.RegistrationConfirmation(&page)).
		Send(ctx.Request().Context())

	if err != nil {
		ctx.Logger().Errorf("unable to send email verification link: %v", err)
		return
	}

	uxflashmessages.Info(ctx, "An email was sent to you to verify your email address.")
}
