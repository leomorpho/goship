package twofa

import (
	"net/http"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	templates "github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/web/middleware"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"
	pages "github.com/leomorpho/goship/modules/2fa/views/web/pages/gen"
)

type routeRegistrar interface {
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

func registerRoutes(r routeRegistrar, ctr ui.Controller, service *Service) {
	r.GET("/welcome/preferences/2fa/setup", getSetup(ctr, service), middleware.RequireAuthentication()).Name = routeNames.RouteNameTwoFactorSetup
	r.POST("/welcome/preferences/2fa/setup", postSetup(ctr, service), middleware.RequireAuthentication()).Name = routeNames.RouteNameTwoFactorSetupSubmit
	r.GET("/welcome/preferences/2fa/backup-codes", getBackupCodes(ctr, service), middleware.RequireAuthentication()).Name = routeNames.RouteNameTwoFactorBackupCodes
	r.GET("/auth/2fa/verify", getVerify(ctr)).Name = routeNames.RouteNameTwoFactorVerify
	r.POST("/auth/2fa/verify", postVerify(ctr, service)).Name = routeNames.RouteNameTwoFactorVerifySubmit
}

func getSetup(ctr ui.Controller, service *Service) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		userEmail, _ := ctx.Get(context.AuthenticatedUserEmailKey).(string)
		secret, qrCodeDataURL, err := service.GenerateSecret(userEmail)
		if err != nil {
			return ctr.Fail(err, "unable to generate totp secret")
		}
		sess, err := session.Get("session", ctx)
		if err != nil {
			return ctr.Fail(err, "unable to open session for two factor setup")
		}
		sess.Values["two_factor_pending_secret"] = secret
		if err := sess.Save(ctx.Request(), ctx.Response()); err != nil {
			return ctr.Fail(err, "unable to save two factor setup session")
		}

		page := ui.NewPage(ctx)
		page.Layout = layouts.Main
		page.Name = templates.PagePreferences
		page.Title = "Two-factor authentication"
		page.Form = viewmodels.NewTwoFactorSetupForm()
		data := viewmodels.NewTwoFactorSetupData()
		data.QRCodeDataURL = qrCodeDataURL
		data.ManualKey = ManualEntryKey(secret)
		page.Data = data
		page.Component = pages.Setup(&page)
		return ctr.RenderPage(ctx, page)
	}
}

func postSetup(ctr ui.Controller, service *Service) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		var form viewmodels.TwoFactorSetupForm
		if err := ctx.Bind(&form); err != nil {
			return ctr.Fail(err, "unable to parse two factor setup form")
		}
		sess, err := session.Get("session", ctx)
		if err != nil {
			return ctr.Fail(err, "unable to open session for two factor setup")
		}
		secret, _ := sess.Values["two_factor_pending_secret"].(string)
		if secret == "" || !service.ValidateCode(secret, form.Code) {
			uxflashmessages.Danger(ctx, "Invalid two-factor code.")
			return getSetup(ctr, service)(ctx)
		}
		delete(sess.Values, "two_factor_pending_secret")
		if err := sess.Save(ctx.Request(), ctx.Response()); err != nil {
			return ctr.Fail(err, "unable to clear two factor setup session")
		}

		userID, _ := ctx.Get(context.AuthenticatedUserIDKey).(int)
		codes := service.GenerateBackupCodes()
		if err := service.Enable(ctx.Request().Context(), userID, secret, codes); err != nil {
			return ctr.Fail(err, "unable to enable two factor authentication")
		}

		page := ui.NewPage(ctx)
		page.Layout = layouts.Main
		page.Name = templates.PagePreferences
		page.Title = "Backup codes"
		data := viewmodels.NewTwoFactorBackupCodesData()
		data.Codes = codes
		page.Data = data
		page.Component = pages.BackupCodes(&page)
		return ctr.RenderPage(ctx, page)
	}
}

func getBackupCodes(ctr ui.Controller, service *Service) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		userID, _ := ctx.Get(context.AuthenticatedUserIDKey).(int)
		codes, err := service.RegenerateBackupCodes(ctx.Request().Context(), userID)
		if err != nil {
			return ctr.Fail(err, "unable to regenerate backup codes")
		}
		page := ui.NewPage(ctx)
		page.Layout = layouts.Main
		page.Name = templates.PagePreferences
		page.Title = "Backup codes"
		data := viewmodels.NewTwoFactorBackupCodesData()
		data.Codes = codes
		page.Data = data
		page.Component = pages.BackupCodes(&page)
		return ctr.RenderPage(ctx, page)
	}
}

func getVerify(ctr ui.Controller) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		page := ui.NewPage(ctx)
		page.Layout = layouts.Auth
		page.Name = templates.PageLogin
		page.Title = "Verify sign in"
		page.Form = viewmodels.NewTwoFactorVerifyForm()
		page.Data = viewmodels.NewTwoFactorVerifyData()
		page.Component = pages.Verify(&page)
		return ctr.RenderPage(ctx, page)
	}
}

func postVerify(ctr ui.Controller, service *Service) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		var form viewmodels.TwoFactorVerifyForm
		if err := ctx.Bind(&form); err != nil {
			return ctr.Fail(err, "unable to parse two factor verify form")
		}
		userID, err := PendingUserIDFromCookie(ctx, ctr.Container.Config.App.EncryptionKey)
		if err != nil {
			uxflashmessages.Danger(ctx, "Your verification session expired. Please sign in again.")
			return ctr.Redirect(ctx, routeNames.RouteNameLogin)
		}
		valid, err := service.ValidateStoredCode(ctx.Request().Context(), userID, form.Code)
		if err != nil {
			return ctr.Fail(err, "unable to validate two factor code")
		}
		if !valid {
			uxflashmessages.Danger(ctx, "Invalid two-factor code.")
			return getVerify(ctr)(ctx)
		}
		if err := ClearPendingUserCookie(ctx); err != nil {
			return ctr.Fail(err, "unable to clear pending two factor cookie")
		}
		if err := ctr.Container.Auth.Login(ctx, userID); err != nil {
			return ctr.Fail(err, "unable to create authenticated session")
		}
		return completeLoginRedirect(ctx, ctr)
	}
}

func completeLoginRedirect(ctx echo.Context, ctr ui.Controller) error {
	sess, err := session.Get("session", ctx)
	if err == nil {
		redirectURL, ok := sess.Values["redirectAfterLogin"].(string)
		if ok && redirectURL != "" {
			delete(sess.Values, "redirectAfterLogin")
			_ = sess.Save(ctx.Request(), ctx.Response())
			return ctx.Redirect(http.StatusFound, redirectURL)
		}
	}
	userID, err := ctr.Container.Auth.GetAuthenticatedUserID(ctx)
	if err != nil {
		return ctr.Redirect(ctx, routeNames.RouteNameLogin)
	}
	identity, err := ctr.Container.Auth.GetIdentityByUserID(ctx.Request().Context(), userID)
	if err == nil && (identity == nil || !identity.ProfileFullyOnboarded) {
		return ctr.Redirect(ctx, routeNames.RouteNamePreferences)
	}
	return ctr.Redirect(ctx, routeNames.RouteNameHomeFeed)
}
