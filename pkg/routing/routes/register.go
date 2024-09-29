package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/msg"
	"github.com/mikestefanello/pagoda/pkg/routing/routenames"
	routeNames "github.com/mikestefanello/pagoda/pkg/routing/routenames"

	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/emails"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
	"github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
)

type (
	register struct {
		ctr                            controller.Controller
		profileRepo                    profilerepo.ProfileRepo
		subscriptionsRepo              subscriptions.SubscriptionsRepo
		notificationSendPermissionRepo *notifierrepo.NotificationSendPermissionRepo
	}
)

func NewRegisterRoute(
	ctr controller.Controller,
	profileRepo profilerepo.ProfileRepo,
	subscriptionsRepo subscriptions.SubscriptionsRepo,
	notificationSendPermissionRepo *notifierrepo.NotificationSendPermissionRepo,
) register {
	return register{
		ctr:                            ctr,
		profileRepo:                    profileRepo,
		subscriptionsRepo:              subscriptionsRepo,
		notificationSendPermissionRepo: notificationSendPermissionRepo,
	}
}

func (c *register) Get(ctx echo.Context) error {
	mode := ctx.QueryParam("mode")

	page := controller.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageRegister
	page.Component = pages.Register(&page)
	page.Title = "Register"
	page.Form = &types.RegisterForm{}

	// Get the current time
	currentTime := time.Now()
	// Subtract 18 years from the current time
	yearsAgo := currentTime.AddDate(-18, 0, 0)
	// Format the date as yyyy-mm-dd
	formattedDate := yearsAgo.Format("2006-01-02")

	page.Data = types.RegisterData{
		UserSignupEnabled:  c.ctr.Container.Config.App.OperationalConstants.UserSignupEnabled,
		RelationshipStatus: mode,
		MinDate:            formattedDate,
	}
	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*types.RegisterForm)
	}
	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}

func (c *register) Post(ctx echo.Context) error {
	var form types.RegisterForm
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
		msg.Warning(ctx, "You must be 18+ to register.")
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
		switch err.(type) {
		case *ent.ConstraintError:
			msg.Warning(ctx, "A user with this email address already exists. Please log in.")
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
		SetAge(profilerepo.CalculateAge(birthdate)).
		Save(ctx.Request().Context())

	if err != nil {
		tx.Rollback()
		ctx.Logger().Errorf("failed to create profile: %v", err)
		msg.Info(ctx, "unable to create user")
		return c.ctr.Redirect(ctx, routenames.RouteNameLogin)
	}

	err = c.subscriptionsRepo.CreateSubscription(ctx.Request().Context(), tx, profile.ID)
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
		err := c.notificationSendPermissionRepo.CreatePermission(
			ctx.Request().Context(), profile.ID, perm, &domain.NotificationPlatformEmail)
		if err != nil {
			log.Error().Err(err).Int("profileID", profile.ID).Msg("failed to create notification permission")
		}
	}

	// Log the user in
	err = c.ctr.Container.Auth.Login(ctx, u.ID)
	if err != nil {
		ctx.Logger().Errorf("unable to log in: %v", err)
		msg.Info(ctx, "Your account has been created.")
		return c.ctr.Redirect(ctx, routeNames.RouteNameLogin)
	}

	msg.Success(ctx, "Your account has been created. You are now logged in. 👌")

	// Send the verification email
	c.sendVerificationEmail(ctx, u)

	redirect, err := redirectAfterLogin(ctx)
	if err != nil {
		return err
	}
	if redirect {
		return nil
	}

	return c.ctr.Redirect(ctx, routeNames.RouteNamePreferences)
}

func (c *register) sendVerificationEmail(ctx echo.Context, usr *ent.User) {
	// Generate a token
	token, err := c.ctr.Container.Auth.GenerateEmailVerificationToken(usr.Email)
	if err != nil {
		ctx.Logger().Errorf("unable to generate email verification token: %v", err)
		return
	}

	url := ctx.Echo().Reverse(routeNames.RouteNameVerifyEmail, token)
	fullUrl := fmt.Sprintf("%s%s", c.ctr.Container.Config.HTTP.Domain, url)

	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Data = types.EmailDefaultData{
		AppName:          string(c.ctr.Container.Config.App.Name),
		ConfirmationLink: fullUrl,
		SupportEmail:     c.ctr.Container.Config.Mail.FromAddress,
		Domain:           c.ctr.Container.Config.HTTP.Domain,
	}

	err = c.ctr.Container.Mail.
		Compose().
		To(usr.Email).
		Subject("Confirm your email address").
		TemplateLayout(layouts.Email).
		Component(emails.RegistrationConfirmation(&page)).
		Send(ctx.Request().Context())

	if err != nil {
		ctx.Logger().Errorf("unable to send email verification link: %v", err)
		return
	}

	msg.Info(ctx, "An email was sent to you to verify your email address.")
}
