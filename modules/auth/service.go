package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/framework/events/types"
	"github.com/leomorpho/goship/framework/views/emails/gen"
	"github.com/leomorpho/goship/framework/web/layouts/gen"
	routeNames "github.com/leomorpho/goship/framework/web/routenames"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/leomorpho/goship/framework/web/viewmodels"
	profilesvc "github.com/leomorpho/goship/modules/profile"
	"github.com/mileusna/useragent"
)

type Service struct {
	ctr                           ui.Controller
	profileService                profilesvc.ProfileService
	subscriptionsService          *paidsubscriptions.Service
	notificationPermissionService *notifications.NotificationPermissionService
	oauth                         *OAuthService
	twoFactor                     TwoFactorAuth
}

type TwoFactorAuth interface {
	IsEnabled(ctx context.Context, userID int) (bool, error)
	BeginPendingLogin(ctx echo.Context, userID int) error
}

func NewService(deps Deps) *Service {
	return &Service{
		ctr:                           deps.Controller,
		profileService:                deps.ProfileService,
		subscriptionsService:          deps.SubscriptionsService,
		notificationPermissionService: deps.NotificationPermissionService,
		oauth: NewOAuthService(
			deps.Controller.Container.Config,
			deps.Controller.Container.Database,
			deps.Controller.Container.Auth,
			deps,
		),
		twoFactor: deps.TwoFactorAuth,
	}
}

func (s *Service) redirectAfterLogin(ctx echo.Context) (bool, error) {
	sess, _ := session.Get("session", ctx)

	redirectURL, ok := sess.Values["redirectAfterLogin"].(string)
	if ok && redirectURL != "" {
		delete(sess.Values, "redirectAfterLogin")
		sess.Save(ctx.Request(), ctx.Response())
		return true, ctx.Redirect(http.StatusFound, redirectURL)
	}
	return false, nil
}

func (s *Service) sendVerificationEmail(ctx echo.Context, userEmail string) {
	token, err := s.ctr.Container.Auth.GenerateEmailVerificationToken(userEmail)
	if err != nil {
		ctx.Logger().Errorf("unable to generate email verification token: %v", err)
		return
	}

	url := ctx.Echo().Reverse(routeNames.RouteNameVerifyEmail, token)
	fullURL := fmt.Sprintf("%s%s", s.ctr.Container.Config.HTTP.Domain, url)

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	data := viewmodels.NewEmailDefaultData()
	data.AppName = string(s.ctr.Container.Config.App.Name)
	data.ConfirmationLink = fullURL
	data.SupportEmail = s.ctr.Container.Config.Mail.FromAddress
	data.Domain = s.ctr.Container.Config.HTTP.Domain
	page.Data = data

	err = s.ctr.Container.Mail.
		Compose().
		To(userEmail).
		Subject("Confirm your email address").
		TemplateLayout(layouts.Email).
		Component(emails.RegistrationConfirmation(&page)).
		Send(ctx.Request().Context())
	if err != nil {
		ctx.Logger().Errorf("unable to send email verification link: %v", err)
	}
}

func (s *Service) sendPasswordResetEmail(ctx echo.Context, profileName, email, url string) error {
	fullURL := fmt.Sprintf("%s%s", s.ctr.Container.Config.HTTP.Domain, url)
	ua := useragent.Parse(ctx.Request().UserAgent())

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	data := viewmodels.NewEmailPasswordResetData()
	data.AppName = string(s.ctr.Container.Config.App.Name)
	data.ProfileName = profileName
	data.PasswordResetLink = fullURL
	data.SupportEmail = s.ctr.Container.Config.Mail.FromAddress
	data.OperatingSystem = ua.OS
	data.BrowserName = ua.Name
	data.Domain = s.ctr.Container.Config.HTTP.Domain
	page.Data = data

	err := s.ctr.Container.Mail.
		Compose().
		To(email).
		Subject("Reset your password").
		TemplateLayout(layouts.Email).
		Component(emails.PasswordReset(&page)).
		Send(ctx.Request().Context())
	if err != nil {
		ctx.Logger().Errorf("unable to send email reset link: %v", err)
		return err
	}
	return nil
}

func (s *Service) publishEvent(ctx context.Context, event any) {
	if s == nil || s.ctr.Container == nil || s.ctr.Container.EventBus == nil {
		return
	}
	if err := s.ctr.Container.EventBus.Publish(ctx, event); err != nil {
		s.ctr.Container.Logger.Errorf("failed to publish domain event: %v", err)
	}
}

func (s *Service) publishUserRegistered(ctx echo.Context, userID int, email string) {
	s.publishEvent(ctx.Request().Context(), types.UserRegistered{
		UserID: int64(userID),
		Email:  email,
		At:     time.Now().UTC(),
	})
}

func (s *Service) publishUserLoggedIn(ctx echo.Context, userID int) {
	s.publishEvent(ctx.Request().Context(), types.UserLoggedIn{
		UserID: int64(userID),
		IP:     ctx.RealIP(),
		At:     time.Now().UTC(),
	})
}

func (s *Service) publishUserLoggedOut(ctx echo.Context, userID int) {
	s.publishEvent(ctx.Request().Context(), types.UserLoggedOut{
		UserID: int64(userID),
		At:     time.Now().UTC(),
	})
}

func (s *Service) publishPasswordChanged(ctx echo.Context, userID int) {
	s.publishEvent(ctx.Request().Context(), types.PasswordChanged{
		UserID: int64(userID),
		At:     time.Now().UTC(),
	})
}

func (s *Service) t(ctx echo.Context, key, fallback string) string {
	if s == nil || s.ctr.Container == nil || s.ctr.Container.I18n == nil {
		return fallback
	}
	translated := strings.TrimSpace(s.ctr.Container.I18n.T(ctx.Request().Context(), key))
	if translated == "" || translated == key {
		return fallback
	}
	return translated
}
