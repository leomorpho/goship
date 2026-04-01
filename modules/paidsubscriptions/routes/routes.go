package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	paidsubscriptions "github.com/leomorpho/goship/v2-modules/paidsubscriptions"
	"github.com/leomorpho/goship/v2/framework/core"
	"github.com/leomorpho/goship/v2/framework/domain"
	frameworkauthcontext "github.com/leomorpho/goship/v2/framework/http/authcontext"
	layouts "github.com/leomorpho/goship/v2/framework/http/layouts/gen"
	pages "github.com/leomorpho/goship/v2/framework/http/pages/gen"
	customctx "github.com/leomorpho/goship/v2/framework/http/requestcontext"
	"github.com/leomorpho/goship/v2/framework/http/ui"
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

type paymentProcessorPublicKey struct {
	Key string `json:"key"`
}

type createCheckoutSessionForm struct {
	Submission ui.FormSubmission
	PriceID    string `form:"price_id" validate:"required"`
}

type productDescription struct {
	Name     string
	Subtitle string
	Price    string
	Points   []string
	PlanKey  string
}

type pricingPageData struct {
	ProductProCode        string
	ProductProPrice       string
	FreePlanKey           string
	DefaultPaidPlanKey    string
	ActivePlanKey         string
	ActivePlanIsPaid      bool
	IsTrial               bool
	HasSubscriptionExpiry bool
	SubscriptionExpiresOn string
	ProductDescriptions   []productDescription
}

func NewRouteModule(controller ui.Controller, service *paidsubscriptions.Service) *RouteModule {
	return &RouteModule{
		controller: controller,
		service:    service,
	}
}

func (m *RouteModule) ID() string {
	return paidsubscriptions.ModuleID
}

func (m *RouteModule) Migrations() fs.FS {
	return nil
}

func (m *RouteModule) RegisterRoutes(r core.Router) error {
	r.GET("/payments/get-public-key", m.GetPaymentProcessorPublickey).Name = "payment_processor.get_public_key"
	r.POST("/payments/create-checkout-session", m.CreateCheckoutSession).Name = "stripe.create_checkout_session"
	r.POST("/payments/create-portal-session", m.CreatePortalSession).Name = "stripe.create_portal_session"
	r.GET("/payments/pricing", m.PricingPage).Name = "pricing_page"
	r.GET("/payments/success", m.SuccessfullySubscribed).Name = "stripe.success"
	return nil
}

func (m *RouteModule) RegisterExternalRoutes(r core.Router, stripeWebhookPath string) error {
	r.POST(stripeWebhookPath, m.HandleWebhook).Name = "stripe.webhook"
	return nil
}

func (m *RouteModule) GetPaymentProcessorPublickey(ctx echo.Context) error {
	key := paymentProcessorPublicKey{}
	key.Key = m.controller.Container.Config.App.PublicStripeKey
	return ctx.JSON(http.StatusOK, key)
}

func (m *RouteModule) CreateCheckoutSession(ctx echo.Context) error {
	var form createCheckoutSessionForm
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

	successURL := ctx.Echo().Reverse("stripe.success")
	fullSuccessURL := fmt.Sprintf("%s%s", m.controller.Container.Config.HTTP.Domain, successURL)
	cancelURL := ctx.Echo().Reverse("preferences")
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

	returnURL := ctx.Echo().Reverse("preferences")
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
		slog.Error("error verifying webhook signature", "error", err, "StripeWebhookSecret", secret)
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "error verifying webhook signature"})
	}

	switch event.Type {
	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			slog.Error("error parsing webhook JSON", "error", err)
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
			slog.Error("error parsing webhook JSON", "error", err)
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
			slog.Error("error parsing webhook JSON", "error", err)
			return echo.ErrBadRequest
		}

		profileID, err := m.service.GetProfileIDFromStripeCustomerID(ctx.Request().Context(), subscription.Customer.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		planKey, err := m.service.DefaultPaidPlanKey()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if err := m.service.ActivatePlan(ctx.Request().Context(), profileID, planKey); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			slog.Error("error parsing webhook JSON", "error", err)
			return echo.ErrBadRequest
		}

		profileID, err := m.service.GetProfileIDFromStripeCustomerID(ctx.Request().Context(), invoice.Customer.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		fullURL := ctx.Echo().Reverse("preferences")
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
			slog.Error("failed to create notification",
				"error", err,
				"selfProfileID", profileID,
				"type", domain.NotificationTypePaymentFailed.Value,
			)
		}

		if err := m.service.CancelWithGracePeriod(ctx.Request().Context(), profileID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	default:
		slog.Error("unhandled event type", "eventType", string(event.Type))
	}

	return ctx.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (m *RouteModule) PricingPage(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.PricingPage(&page)
	page.Name = "pricing"

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	activePlan, subscriptionExpiredOn, isTrial, err := m.service.GetCurrentlyActiveProduct(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}
	defaultPaidPlanKey, err := m.service.DefaultPaidPlanKey()
	if err != nil {
		return err
	}
	activePlanKey := m.activePlanKey(activePlan)

	data := pricingPageData{ProductDescriptions: []productDescription{}}
	data.ProductProCode = m.controller.Container.Config.App.OperationalConstants.ProductProCode
	data.ProductProPrice = fmt.Sprintf("%.2f", m.controller.Container.Config.App.OperationalConstants.ProductProPrice)
	data.FreePlanKey = m.service.FreePlanKey()
	data.DefaultPaidPlanKey = defaultPaidPlanKey
	data.ActivePlanKey = activePlanKey
	data.ActivePlanIsPaid = m.isPaidPlanKey(activePlanKey)
	data.HasSubscriptionExpiry = subscriptionExpiredOn != nil
	data.IsTrial = isTrial
	description := productDescription{Points: []string{}}
	data.ProductDescriptions = []productDescription{description}
	if subscriptionExpiredOn != nil {
		data.SubscriptionExpiresOn = subscriptionExpiredOn.Format(time.RFC3339Nano)
	}
	page.Data = data
	page.HTMX.Request.Boosted = true

	return m.controller.RenderPage(ctx, page)
}

func (m *RouteModule) SuccessfullySubscribed(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.PaymentSuccess(&page)
	page.Name = "successfully_subscribed"
	page.HTMX.Request.Boosted = true

	return m.controller.RenderPage(ctx, page)
}

func (m *RouteModule) activePlanKey(pt *paidsubscriptions.ProductType) string {
	return m.service.ActivePlanKey(pt)
}

func (m *RouteModule) isPaidPlanKey(planKey string) bool {
	return m.service.IsPaidPlanKey(planKey)
}

func authenticatedProfileID(ctx echo.Context) (int, error) {
	return frameworkauthcontext.AuthenticatedProfileID(ctx)
}

func authenticatedUserEmail(ctx echo.Context) (string, error) {
	return frameworkauthcontext.AuthenticatedUserEmail(ctx)
}
