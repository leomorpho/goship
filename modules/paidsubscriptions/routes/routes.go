package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	appsubscriptions "github.com/leomorpho/goship/app/subscriptions"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	customctx "github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v78"
	portalsession "github.com/stripe/stripe-go/v78/billingportal/session"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/customer"
	"github.com/stripe/stripe-go/v78/webhook"
)

type RouteModule struct {
	controller ui.Controller
	service    *paidsubscriptions.Service
}

func NewRouteModule(controller ui.Controller, service *paidsubscriptions.Service) *RouteModule {
	return &RouteModule{
		controller: controller,
		service:    service,
	}
}

func (m *RouteModule) ID() string {
	return "paidsubscriptions"
}

func (m *RouteModule) Migrations() fs.FS {
	return nil
}

func (m *RouteModule) RegisterRoutes(r core.Router) error {
	r.GET("/payments/get-public-key", m.GetPaymentProcessorPublickey).Name = routeNames.RouteNamePaymentProcessorGetPublicKey
	r.POST("/payments/create-checkout-session", m.CreateCheckoutSession).Name = routeNames.RouteNameCreateCheckoutSession
	r.POST("/payments/create-portal-session", m.CreatePortalSession).Name = routeNames.RouteNameCreatePortalSession
	r.GET("/payments/pricing", m.PricingPage).Name = routeNames.RouteNamePricingPage
	r.GET("/payments/success", m.SuccessfullySubscribed).Name = routeNames.RouteNamePaymentProcessorSuccess
	return nil
}

func (m *RouteModule) RegisterExternalRoutes(r core.Router, stripeWebhookPath string) error {
	r.POST(stripeWebhookPath, m.HandleWebhook).Name = routeNames.RouteNamePaymentProcessorWebhook
	return nil
}

func (m *RouteModule) GetPaymentProcessorPublickey(ctx echo.Context) error {
	key := viewmodels.PaymentProcessorPublicKey{
		Key: m.controller.Container.Config.App.PublicStripeKey,
	}
	return ctx.JSON(http.StatusOK, key)
}

func (m *RouteModule) CreateCheckoutSession(ctx echo.Context) error {
	var form viewmodels.CreateCheckoutSessionForm
	ctx.Set(customctx.FormKey, &form)

	if err := ctx.Bind(&form); err != nil {
		return m.controller.Fail(err, "unable to parse login form")
	}
	if err := form.Submission.Process(ctx, form); err != nil {
		return m.controller.Fail(err, "unable to process form submission")
	}
	if form.Submission.HasErrors() {
		return m.PricingPage(ctx)
	}

	successURL := ctx.Echo().Reverse(routeNames.RouteNamePaymentProcessorSuccess)
	fullSuccessURL := fmt.Sprintf("%s%s", m.controller.Container.Config.HTTP.Domain, successURL)
	cancelURL := ctx.Echo().Reverse(routeNames.RouteNamePreferences)
	fullCancelURL := fmt.Sprintf("%s%s", m.controller.Container.Config.HTTP.Domain, cancelURL)

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	userEmail, err := authenticatedUserEmail(ctx)
	if err != nil {
		return err
	}

	customerParams := &stripe.CustomerParams{
		Email: stripe.String(userEmail),
	}
	stripeCustomer, err := customer.New(customerParams)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create or retrieve Stripe customer"})
	}

	if err := m.service.StoreStripeCustomerID(ctx.Request().Context(), profileID, stripeCustomer.ID); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store Stripe customer ID"})
	}

	checkoutParams := &stripe.CheckoutSessionParams{
		Customer: stripe.String(stripeCustomer.ID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Price:    stripe.String(form.PriceID),
			Quantity: stripe.Int64(1),
		}},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(fullSuccessURL),
		CancelURL:  stripe.String(fullCancelURL),
	}

	checkoutSession, err := session.New(checkoutParams)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.Redirect(http.StatusSeeOther, checkoutSession.URL)
}

func (m *RouteModule) CreatePortalSession(ctx echo.Context) error {
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	stripeCustomerID, err := m.service.GetStripeCustomerIDByProfileID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	returnURL := ctx.Echo().Reverse(routeNames.RouteNamePreferences)
	fullReturnURL := fmt.Sprintf("%s%s", m.controller.Container.Config.HTTP.Domain, returnURL)

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(stripeCustomerID),
		ReturnURL: stripe.String(fullReturnURL),
	}
	portalSession, err := portalsession.New(params)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Redirect(http.StatusSeeOther, portalSession.URL)
}

func (m *RouteModule) HandleWebhook(ctx echo.Context) error {
	const maxBodyBytes = int64(65536)

	ctx.Request().Body = http.MaxBytesReader(ctx.Response(), ctx.Request().Body, maxBodyBytes)
	payload, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "error reading request body"})
	}

	secret := m.controller.Container.Config.App.StripeWebhookSecret
	event, err := webhook.ConstructEvent(payload, ctx.Request().Header.Get("Stripe-Signature"), secret)
	if err != nil {
		log.Error().Err(err).Str("StripeWebhookSecret", secret).Msg("error verifying webhook signature")
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "error verifying webhook signature"})
	}

	switch event.Type {
	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Error().Err(err).Msg("error parsing webhook JSON")
			return echo.ErrBadRequest
		}

		profileID, err := m.service.GetProfileIDFromStripeCustomerID(ctx.Request().Context(), subscription.Customer.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		if err := m.service.CancelWithGracePeriod(ctx.Request().Context(), profileID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	case "customer.subscription.updated":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Error().Err(err).Msg("error parsing webhook JSON")
			return echo.ErrBadRequest
		}

		profileID, err := m.service.GetProfileIDFromStripeCustomerID(ctx.Request().Context(), subscription.Customer.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		var cancelDate *time.Time
		if subscription.CancelAt != 0 {
			t := time.Unix(subscription.CancelAt, 0)
			cancelDate = &t
		} else if subscription.CanceledAt != 0 {
			t := time.Unix(subscription.CanceledAt, 0)
			cancelDate = &t
		}
		if err := m.service.CancelOrRenew(ctx.Request().Context(), profileID, cancelDate); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	case "customer.subscription.created":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Error().Err(err).Msg("error parsing webhook JSON")
			return echo.ErrBadRequest
		}

		profileID, err := m.service.GetProfileIDFromStripeCustomerID(ctx.Request().Context(), subscription.Customer.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		if err := m.service.ActivatePlan(ctx.Request().Context(), profileID, appsubscriptions.PlanProKey); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Error().Err(err).Msg("error parsing webhook JSON")
			return echo.ErrBadRequest
		}

		profileID, err := m.service.GetProfileIDFromStripeCustomerID(ctx.Request().Context(), invoice.Customer.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		fullURL := ctx.Echo().Reverse(routeNames.RouteNamePreferences)
		err = m.controller.Container.Notifier.PublishNotification(
			ctx.Request().Context(),
			domain.Notification{
				Type:      domain.NotificationTypePaymentFailed,
				ProfileID: profileID,
				Title:     "Payment issue",
				Text: fmt.Sprintf(
					"💸 Oh no, your payment didn’t go through 🙁. Update your payment info within %d days to ensure uninterrupted pro membership.",
					m.controller.Container.Config.App.OperationalConstants.PaymentFailedGracePeriodInDays,
				),
				Link:                    fullURL,
				ProfileIDWhoCausedNotif: profileID,
			},
			true,
			true,
		)
		if err != nil {
			log.Error().Err(err).
				Int("selfProfileID", profileID).
				Str("type", domain.NotificationTypePaymentFailed.Value).
				Msg("failed to create notification")
		}

		if err := m.service.CancelWithGracePeriod(ctx.Request().Context(), profileID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	default:
		log.Error().Str("eventType", string(event.Type)).Msg("unhandled event type")
	}

	return ctx.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (m *RouteModule) PricingPage(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.PricingPage(&page)
	page.Name = templates.PagePricing

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	activePlan, subscriptionExpiredOn, isTrial, err := m.service.GetCurrentlyActiveProduct(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}
	activePlanKey := activePlanKey(activePlan)

	page.Data = viewmodels.PricingPageData{
		ProductProCode:        m.controller.Container.Config.App.OperationalConstants.ProductProCode,
		ProductProPrice:       fmt.Sprintf("%.2f", m.controller.Container.Config.App.OperationalConstants.ProductProPrice),
		ActivePlanKey:         activePlanKey,
		ActivePlanIsPaid:      isPaidPlanKey(activePlanKey),
		SubscriptionExpiresOn: subscriptionExpiredOn,
		IsTrial:               isTrial,
		ProductDescriptions: []viewmodels.ProductDescription{{
			Name:     "",
			Subtitle: "",
		}},
	}
	page.HTMX.Request.Boosted = true

	return m.controller.RenderPage(ctx, page)
}

func (m *RouteModule) SuccessfullySubscribed(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.PaymentSuccess(&page)
	page.Name = templates.PageSuccessfullySubscribed
	page.HTMX.Request.Boosted = true

	return m.controller.RenderPage(ctx, page)
}

func activePlanKey(pt *paidsubscriptions.ProductType) string {
	if pt == nil || pt.Value == "" {
		return appsubscriptions.PlanFreeKey
	}
	return pt.Value
}

func isPaidPlanKey(planKey string) bool {
	return planKey != appsubscriptions.PlanFreeKey
}

func authenticatedProfileID(ctx echo.Context) (int, error) {
	v := ctx.Get(customctx.AuthenticatedProfileIDKey)
	profileID, ok := v.(int)
	if !ok || profileID <= 0 {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "authenticated profile id missing from context")
	}
	return profileID, nil
}

func authenticatedUserEmail(ctx echo.Context) (string, error) {
	v := ctx.Get(customctx.AuthenticatedUserEmailKey)
	email, ok := v.(string)
	if !ok || email == "" {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "authenticated user email missing from context")
	}
	return email, nil
}
