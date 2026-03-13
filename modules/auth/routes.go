package auth

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	templates "github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/web/middleware"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/dberrors"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/repos/ratelimit"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"
	pages "github.com/leomorpho/goship/modules/auth/views/web/pages/gen"
	"log/slog"
)

var (
	authPostRateLimitOnce sync.Once
	authPostRateLimitMW   echo.MiddlewareFunc
)

func registerRoutes(r echoRouteRegistrar, service *Service) {
	postRateLimit := authPostRateLimitMiddleware()
	userGroup := r.Group("/user", middleware.RequireNoAuthentication())
	userGroup.GET("/login", service.getLogin).Name = routeNames.RouteNameLogin
	userGroup.POST("/login", service.postLogin, postRateLimit).Name = routeNames.RouteNameLoginSubmit
	userGroup.GET("/register", service.getRegister).Name = routeNames.RouteNameRegister
	userGroup.POST("/register", service.postRegister, postRateLimit).Name = routeNames.RouteNameRegisterSubmit
	userGroup.GET("/password", service.getForgotPassword).Name = routeNames.RouteNameForgotPassword
	userGroup.POST("/password", service.postForgotPassword, postRateLimit).Name = routeNames.RouteNameForgotPasswordSubmit

	resetGroup := userGroup.Group(
		"/password/reset",
		middleware.LoadUser(service.ctr.Container.Auth),
		middleware.LoadValidPasswordToken(service.ctr.Container.Auth),
	)
	resetGroup.GET("/token/:user/:password_token/:token", service.getResetPassword).Name = routeNames.RouteNameResetPassword
	resetGroup.POST("/token/:user/:password_token/:token", service.postResetPassword, postRateLimit).Name = routeNames.RouteNameResetPasswordSubmit

	allGroup := r.Group("/auth", middleware.RequireAuthentication())
	allGroup.GET("/logout", service.getLogout, middleware.RequireAuthentication()).Name = routeNames.RouteNameLogout

	r.GET("/auth/oauth/:provider", service.getOAuthProviderStart, middleware.RequireNoAuthentication()).Name = routeNames.RouteNameOAuthStart
	r.GET("/auth/oauth/:provider/callback", service.getOAuthProviderCallback, middleware.RequireNoAuthentication()).Name = routeNames.RouteNameOAuthCallback
	r.GET("/email/verify/:token", service.getVerifyEmail).Name = routeNames.RouteNameVerifyEmail
}

func authPostRateLimitMiddleware() echo.MiddlewareFunc {
	authPostRateLimitOnce.Do(func() {
		store, err := ratelimit.NewOtterStore(10_000)
		if err != nil {
			panic(err)
		}
		authPostRateLimitMW = middleware.RateLimit(store, 10, time.Minute)
	})
	return authPostRateLimitMW
}

type echoRouteRegistrar interface {
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	Group(prefix string, middleware ...echo.MiddlewareFunc) *echo.Group
}

func (s *Service) getLogin(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageLogin
	page.Title = "Log in"
	page.Form = viewmodels.NewLoginForm()
	data := viewmodels.NewLoginOAuthData()
	for _, provider := range s.oauth.EnabledProviders() {
		providerView := viewmodels.NewLoginOAuthProvider()
		providerView.Name = provider.Name
		providerView.Label = provider.Label
		data.Providers = append(data.Providers, providerView)
	}
	page.Data = data
	page.Component = pages.Login(&page)
	page.HTMX.Request.Boosted = true

	s.ctr.Container.Auth.Logout(ctx)

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.LoginForm)
	}

	return s.ctr.RenderPage(ctx, page)
}

func (s *Service) postLogin(ctx echo.Context) error {
	var form viewmodels.LoginForm
	ctx.Set(context.FormKey, &form)

	authFailed := func() error {
		uxflashmessages.Danger(ctx, "Invalid credentials. Please try again.")
		return s.getLogin(ctx)
	}

	if err := ctx.Bind(&form); err != nil {
		return s.ctr.Fail(err, "unable to parse login form")
	}
	if err := form.Submission.Process(ctx, form); err != nil {
		return s.ctr.Fail(err, "unable to process form submission")
	}
	if form.Submission.HasErrors() {
		return s.getLogin(ctx)
	}

	usr, err := s.ctr.Container.Auth.AuthenticateUserByEmailPassword(ctx, form.Email, form.Password)
	switch err.(type) {
	case nil:
	case foundation.InvalidCredentialsError:
		ctx.Logger().Debug("credentials incorrect")
		return authFailed()
	default:
		return s.ctr.Fail(err, "error authenticating user during login")
	}

	if err := s.ctr.Container.Auth.Login(ctx, usr.UserID); err != nil {
		return s.ctr.Fail(err, "unable to log in user")
	}

	return s.finishLogin(ctx, usr.UserID)
}

func (s *Service) getRegister(ctx echo.Context) error {
	mode := ctx.QueryParam("mode")

	page := ui.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageRegister
	page.Component = pages.Register(&page)
	page.Title = "Register"
	page.Form = viewmodels.NewRegisterForm()

	yearsAgo := time.Now().AddDate(-18, 0, 0)
	data := viewmodels.NewRegisterData()
	data.UserSignupEnabled = s.ctr.Container.Config.App.OperationalConstants.UserSignupEnabled
	data.RelationshipStatus = mode
	data.MinDate = yearsAgo.Format("2006-01-02")
	page.Data = data
	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.RegisterForm)
	}
	page.HTMX.Request.Boosted = true

	return s.ctr.RenderPage(ctx, page)
}

func (s *Service) postRegister(ctx echo.Context) error {
	var form viewmodels.RegisterForm
	ctx.Set(context.FormKey, &form)

	if err := ctx.Bind(&form); err != nil {
		return s.ctr.Fail(err, "unable to parse register form")
	}
	if err := form.Submission.Process(ctx, form); err != nil {
		return s.ctr.Fail(err, "unable to process form submission")
	}
	if form.Submission.HasErrors() {
		return s.getRegister(ctx)
	}

	pwHash, err := s.ctr.Container.Auth.HashPassword(form.Password)
	if err != nil {
		return s.ctr.Fail(err, "unable to hash password")
	}

	birthdate, err := time.ParseInLocation("2006-01-02", form.Birthdate, time.UTC)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid birthdate format")
	}
	birthdate = birthdate.UTC().Truncate(24 * time.Hour)

	if birthdate.After(time.Now().UTC().AddDate(-18, 0, 0)) {
		uxflashmessages.Warning(ctx, "You must be 18+ to register.")
		return s.getRegister(ctx)
	}

	registration, err := s.profileService.RegisterUserWithProfile(
		ctx.Request().Context(),
		form.Name,
		form.Email,
		pwHash,
		birthdate,
		s.subscriptionsService,
	)
	if err != nil {
		switch {
		case dberrors.IsConstraint(err):
			uxflashmessages.Warning(ctx, "A user with this email address already exists. Please log in.")
			return s.ctr.Redirect(ctx, routeNames.RouteNameLogin)
		default:
			return s.ctr.Fail(err, "unable to create user and profile")
		}
	}
	ctx.Logger().Infof("user and profile created successfully: %s", registration.UserName)

	for _, perm := range domain.NotificationPermissions.Members() {
		err := s.notificationPermissionService.CreatePermission(
			ctx.Request().Context(), registration.ProfileID, perm, &domain.NotificationPlatformEmail)
		if err != nil {
			slog.Error("failed to create notification permission", "error", err, "profileID", registration.ProfileID)
		}
	}

	if err := s.ctr.Container.Auth.Login(ctx, registration.UserID); err != nil {
		ctx.Logger().Errorf("unable to log in: %v", err)
		uxflashmessages.Info(ctx, "Your account has been created.")
		return s.ctr.Redirect(ctx, routeNames.RouteNameLogin)
	}

	uxflashmessages.Success(ctx, "Your account has been created. You are now logged in. 👌")
	s.sendVerificationEmail(ctx, registration.UserEmail)

	redirect, err := s.redirectAfterLogin(ctx)
	if err != nil {
		return err
	}
	if redirect {
		return nil
	}

	return s.ctr.Redirect(ctx, routeNames.RouteNamePreferences)
}

func (s *Service) finishLogin(ctx echo.Context, userID int) error {
	redirect, err := s.redirectAfterLogin(ctx)
	if err != nil {
		return err
	}
	if redirect {
		return nil
	}

	identity, err := s.ctr.Container.Auth.GetIdentityByUserID(ctx.Request().Context(), userID)
	if err != nil {
		return s.ctr.Fail(err, "unable to determine profile onboarding status")
	}
	if identity == nil || !identity.ProfileFullyOnboarded {
		return s.ctr.Redirect(ctx, routeNames.RouteNamePreferences)
	}
	return s.ctr.Redirect(ctx, routeNames.RouteNameHomeFeed)
}

func (s *Service) getOAuthProviderStart(ctx echo.Context) error {
	provider := ctx.Param("provider")
	state, err := s.ctr.Container.Auth.RandomToken(32)
	if err != nil {
		return s.ctr.Fail(err, "unable to create oauth state")
	}

	authorizationURL, err := s.oauth.AuthorizationURL(provider, state)
	if err != nil {
		uxflashmessages.Warning(ctx, "That sign-in provider is not available.")
		return s.ctr.Redirect(ctx, routeNames.RouteNameLogin)
	}

	sess, err := session.Get("session", ctx)
	if err != nil {
		return s.ctr.Fail(err, "unable to open session for oauth state")
	}
	sess.Values["oauth_state"] = state
	sess.Values["oauth_provider"] = provider
	if err := sess.Save(ctx.Request(), ctx.Response()); err != nil {
		return s.ctr.Fail(err, "unable to save oauth state")
	}

	return ctx.Redirect(http.StatusFound, authorizationURL)
}

func (s *Service) getOAuthProviderCallback(ctx echo.Context) error {
	provider := ctx.Param("provider")
	code := ctx.QueryParam("code")
	state := ctx.QueryParam("state")
	if strings.TrimSpace(code) == "" || strings.TrimSpace(state) == "" {
		uxflashmessages.Danger(ctx, "OAuth sign-in could not be completed.")
		return s.ctr.Redirect(ctx, routeNames.RouteNameLogin)
	}

	if err := consumeOAuthState(ctx, provider, state); err != nil {
		uxflashmessages.Danger(ctx, "OAuth state validation failed. Please try again.")
		return s.ctr.Redirect(ctx, routeNames.RouteNameLogin)
	}

	result, err := s.oauth.HandleCallback(ctx.Request().Context(), provider, code)
	if err != nil {
		return s.ctr.Fail(err, "unable to complete oauth callback")
	}
	if err := s.ctr.Container.Auth.Login(ctx, result.UserID); err != nil {
		return s.ctr.Fail(err, "unable to create oauth session")
	}
	return s.finishLogin(ctx, result.UserID)
}

func consumeOAuthState(ctx echo.Context, provider, state string) error {
	sess, err := session.Get("session", ctx)
	if err != nil {
		return err
	}
	storedState, _ := sess.Values["oauth_state"].(string)
	storedProvider, _ := sess.Values["oauth_provider"].(string)
	delete(sess.Values, "oauth_state")
	delete(sess.Values, "oauth_provider")
	if err := sess.Save(ctx.Request(), ctx.Response()); err != nil {
		return err
	}
	if storedState == "" || storedState != state || storedProvider != provider {
		return errOAuthStateInvalid
	}
	return nil
}

func (s *Service) getForgotPassword(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageForgotPassword
	page.Title = "Forgot password"
	page.Form = viewmodels.NewForgotPasswordForm()
	page.Component = pages.ForgotPassword(&page)
	page.HTMX.Request.Boosted = true

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.ForgotPasswordForm)
	}

	return s.ctr.RenderPage(ctx, page)
}

func (s *Service) postForgotPassword(ctx echo.Context) error {
	var form viewmodels.ForgotPasswordForm
	ctx.Set(context.FormKey, &form)

	succeed := func() error {
		ctx.Set(context.FormKey, nil)
		uxflashmessages.Success(ctx, "An email was sent to reset your password.")
		return s.getForgotPassword(ctx)
	}

	if err := ctx.Bind(&form); err != nil {
		return s.ctr.Fail(err, "unable to parse forgot password form")
	}
	if err := form.Submission.Process(ctx, form); err != nil {
		return s.ctr.Fail(err, "unable to process form submission")
	}
	if form.Submission.HasErrors() {
		return s.getForgotPassword(ctx)
	}

	u, err := s.ctr.Container.Auth.FindUserRecordByEmail(ctx, form.Email)
	switch {
	case dberrors.IsNotFound(err):
		return succeed()
	case err != nil:
		return s.ctr.Fail(err, "error querying user during forgot password")
	}

	token, tokenID, err := s.ctr.Container.Auth.GeneratePasswordResetToken(ctx, u.UserID)
	if err != nil {
		return s.ctr.Fail(err, "error generating password reset token")
	}
	ctx.Logger().Infof("generated password reset token for user %d", u.UserID)

	url := ctx.Echo().Reverse(routeNames.RouteNameResetPassword, u.UserID, tokenID, token)
	if err := s.sendPasswordResetEmail(ctx, u.Name, u.Email, url); err != nil {
		return err
	}

	return succeed()
}

func (s *Service) getResetPassword(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Auth
	page.Name = templates.PageResetPassword
	page.Title = "Reset password"
	page.Form = viewmodels.NewResetPasswordForm()
	page.Component = pages.ResetPassword(&page)

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.ResetPasswordForm)
	}

	return s.ctr.RenderPage(ctx, page)
}

func (s *Service) postResetPassword(ctx echo.Context) error {
	var form viewmodels.ResetPasswordForm
	ctx.Set(context.FormKey, &form)

	if err := ctx.Bind(&form); err != nil {
		return s.ctr.Fail(err, "unable to parse password reset form")
	}
	if err := form.Submission.Process(ctx, form); err != nil {
		return s.ctr.Fail(err, "unable to process form submission")
	}
	if form.Submission.HasErrors() {
		return s.getResetPassword(ctx)
	}

	hash, err := s.ctr.Container.Auth.HashPassword(form.Password)
	if err != nil {
		return s.ctr.Fail(err, "unable to hash password")
	}

	userIDRaw := ctx.Get(context.AuthenticatedUserIDKey)
	userID, ok := userIDRaw.(int)
	if !ok || userID <= 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "authenticated user id missing from context")
	}

	if err = s.ctr.Container.Auth.SetUserPasswordHashByUserID(ctx, userID, hash); err != nil {
		return s.ctr.Fail(err, "unable to update password")
	}
	if err = s.ctr.Container.Auth.DeletePasswordTokens(ctx, userID); err != nil {
		return s.ctr.Fail(err, "unable to delete password tokens")
	}

	uxflashmessages.Success(ctx, "Your password has been updated.")
	return s.ctr.Redirect(ctx, routeNames.RouteNameLogin)
}

func (s *Service) getLogout(ctx echo.Context) error {
	if err := s.ctr.Container.Auth.Logout(ctx); err != nil {
		uxflashmessages.Danger(ctx, "An error occurred. Please try again.")
	}
	return s.ctr.Redirect(ctx, routeNames.RouteNameLandingPage)
}

func (s *Service) getVerifyEmail(ctx echo.Context) error {
	token := ctx.Param("token")
	email, err := s.ctr.Container.Auth.ValidateEmailVerificationToken(token)
	if err != nil {
		uxflashmessages.Warning(ctx, "The link is either invalid or has expired.")
		return s.ctr.Redirect(ctx, routeNames.RouteNameLandingPage)
	}

	authEmail, authEmailOK := ctx.Get(context.AuthenticatedUserEmailKey).(string)
	authUserID, authUserIDOK := ctx.Get(context.AuthenticatedUserIDKey).(int)
	if authEmailOK && authUserIDOK && authEmail == email {
		if err = s.ctr.Container.Auth.MarkUserVerifiedByUserID(ctx, authUserID); err != nil {
			return s.ctr.Fail(err, "failed to set authenticated user as verified")
		}
	} else {
		usr, queryErr := s.ctr.Container.Auth.FindUserRecordByEmail(ctx, email)
		if queryErr != nil {
			return s.ctr.Fail(queryErr, "query failed loading email verification token user")
		}
		if !usr.IsVerified {
			if err = s.ctr.Container.Auth.MarkUserVerifiedByUserID(ctx, usr.UserID); err != nil {
				return s.ctr.Fail(err, "failed to set user as verified")
			}
		}
	}

	uxflashmessages.Success(ctx, "Your email has been successfully verified.")
	if ctx.Get(context.AuthenticatedUserIDKey) != nil {
		return s.ctr.Redirect(ctx, routeNames.RouteNamePreferences)
	}
	return s.ctr.Redirect(ctx, routeNames.RouteNameLogin)
}
