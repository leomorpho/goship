package routes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/ent"
	internalContext "github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	routeNames "github.com/mikestefanello/pagoda/pkg/routing/routenames"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v78"
	portalsession "github.com/stripe/stripe-go/v78/billingportal/session"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/customer"
	"github.com/stripe/stripe-go/v78/webhook"
)

type (
	paymentsRoute struct {
		ctr               controller.Controller
		orm               *ent.Client
		subscriptionsRepo *subscriptions.SubscriptionsRepo
	}
)

func NewPaymentsRoute(
	ctr controller.Controller, orm *ent.Client, subscriptionsRepo *subscriptions.SubscriptionsRepo,
) paymentsRoute {
	return paymentsRoute{
		ctr:               ctr,
		orm:               orm,
		subscriptionsRepo: subscriptionsRepo,
	}
}

func (p *paymentsRoute) GetPaymentProcessorPublickey(ctx echo.Context) error {
	key := types.PaymentProcessorPublicKey{
		Key: p.ctr.Container.Config.App.PublicStripeKey,
	}
	return ctx.JSON(http.StatusOK, key)
}

func (p *paymentsRoute) CreateCheckoutSession(ctx echo.Context) error {
	var form types.CreateCheckoutSessionForm
	ctx.Set(internalContext.FormKey, &form)

	// Parse the form values
	if err := ctx.Bind(&form); err != nil {
		return p.ctr.Fail(err, "unable to parse login form")
	}

	if err := form.Submission.Process(ctx, form); err != nil {
		return p.ctr.Fail(err, "unable to process form submission")
	}
	if form.Submission.HasErrors() {
		return p.PricingPage(ctx)
	}
	successURL := ctx.Echo().Reverse(routeNames.RouteNamePaymentProcessorSuccess)
	fullSuccessUrl := fmt.Sprintf("%s%s", p.ctr.Container.Config.HTTP.Domain, successURL)
	cancelURL := ctx.Echo().Reverse(routeNames.RouteNamePreferences)
	fullCancelUrl := fmt.Sprintf("%s%s", p.ctr.Container.Config.HTTP.Domain, cancelURL)

	usr := ctx.Get(internalContext.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	// Create or retrieve the Stripe customer
	customerParams := &stripe.CustomerParams{
		Email: stripe.String(usr.Email),
	}
	customer, err := customer.New(customerParams)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create or retrieve Stripe customer"})
	}

	// Store the Stripe customer ID in your database
	err = p.subscriptionsRepo.StoreStripeCustomerID(ctx.Request().Context(), profile.ID, customer.ID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store Stripe customer ID"})
	}

	checkoutParams := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customer.ID),
		// PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(form.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(fullSuccessUrl),
		CancelURL:  stripe.String(fullCancelUrl),
		// AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(true)},
	}

	session, err := session.New(checkoutParams)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.Redirect(http.StatusSeeOther, session.URL)
}

func (p *paymentsRoute) CreatePortalSession(ctx echo.Context) error {
	usr := ctx.Get(internalContext.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	returnURL := ctx.Echo().Reverse(routeNames.RouteNamePreferences)
	fullReturnsUrl := fmt.Sprintf("%s%s", p.ctr.Container.Config.HTTP.Domain, returnURL)

	// Authenticate your user.
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(profile.StripeID),
		ReturnURL: stripe.String(fullReturnsUrl),
	}
	ps, err := portalsession.New(params)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Redirect(http.StatusSeeOther, ps.URL)
}

func (p *paymentsRoute) HandleWebhook(c echo.Context) error {

	const MaxBodyBytes = int64(65536)
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "error reading request body"})
	}

	secret := p.ctr.Container.Config.App.StripeWebhookSecret

	event, err := webhook.ConstructEvent(payload, c.Request().Header.Get("Stripe-Signature"), secret)
	if err != nil {
		log.Error().Err(err).Str("StripeWebhookSecret", secret).Msg("error verifying webhook signature")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "error verifying webhook signature"})
	}

	// TODO: convert all cases to use the tasks in pkg/tasks/subscriptions.go

	// Unmarshal the event data into an appropriate struct depending on its Type
	switch event.Type {
	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}
		log.Info().Str("subscriptionID", subscription.ID).Msg("Subscription deleted")

		profileID, err := p.subscriptionsRepo.GetProfileIDFromStripeCustomerID(
			c.Request().Context(), subscription.Customer.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		err = p.subscriptionsRepo.CancelWithGracePeriod(c.Request().Context(), profileID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	case "customer.subscription.updated":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}

		// Get the profile ID from Stripe customer ID
		profileID, err := p.subscriptionsRepo.GetProfileIDFromStripeCustomerID(
			c.Request().Context(), subscription.Customer.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		// Extract the cancellation date
		var cancelDate *time.Time
		if subscription.CancelAt != 0 {
			t := time.Unix(subscription.CancelAt, 0)
			cancelDate = &t
		} else if subscription.CanceledAt != 0 {
			t := time.Unix(subscription.CanceledAt, 0)
			cancelDate = &t
		}
		// Call function to handle subscription cancellation
		err = p.subscriptionsRepo.CancelOrRenew(c.Request().Context(), profileID, cancelDate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		log.Info().Str("subscriptionID", subscription.ID).Msg("Subscription updated")

	case "customer.subscription.created":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}
		log.Info().Str("subscriptionID", subscription.ID).Msg("Subscription created")
		// Change to pro by default, alert customer if payment failed later
		profileID, err := p.subscriptionsRepo.GetProfileIDFromStripeCustomerID(
			c.Request().Context(), subscription.Customer.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		err = p.subscriptionsRepo.UpdateToPaidPro(c.Request().Context(), profileID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		// TODO: Listen to "invoice.payment_succeeded" to set the "paid" field. Then, we can have a daily task to check
		// with Stripe API to see if we missed payments on granted subscriptions that were not yet paid.

	case "invoice.payment_failed":
		// Start TypeSubscriptionPaymentFailed task
		var invoice stripe.Invoice
		err := json.Unmarshal(event.Data.Raw, &invoice)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing webhook JSON")
			return echo.ErrBadRequest
		}
		log.Info().Str("invoiceID", invoice.ID).Msg("Invoice payment failed")
		profileID, err := p.subscriptionsRepo.GetProfileIDFromStripeCustomerID(
			c.Request().Context(), invoice.Customer.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		fullURL := c.Echo().Reverse(routeNames.RouteNamePreferences)
		err = p.ctr.Container.Notifier.PublishNotification(
			c.Request().Context(),
			domain.Notification{
				Type:      domain.NotificationTypePaymentFailed,
				ProfileID: profileID,
				Title:     "Payment issue",
				Text: fmt.Sprintf(fmt.Sprintf("💸 Oh no, your payment didn’t go through 🙁. Update your payment info within %d days to ensure uninterrupted pro membership.",
					p.ctr.Container.Config.App.OperationalConstants.PaymentFailedGracePeriodInDays)),
				Link:                    fullURL,
				ProfileIDWhoCausedNotif: profileID,
			},
			true, true,
		)

		if err != nil {
			log.Error().Err(err).
				Int("selfProfileID", profileID).
				Str("type", domain.NotificationTypePaymentFailed.Value).
				Msg("failed to create notification")

		}
		err = p.subscriptionsRepo.CancelWithGracePeriod(c.Request().Context(), profileID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	default:
		log.Error().Str("eventType", string(event.Type)).Msg("Unhandled event type")
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (p *paymentsRoute) PricingPage(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.PricingPage(&page)
	page.Name = templates.PagePricing

	usr := ctx.Get(internalContext.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	activePlan, subscriptionExpiredOn, isTrial, err := p.subscriptionsRepo.GetCurrentlyActiveProduct(
		ctx.Request().Context(), profile.ID,
	)
	if err != nil {
		return err
	}

	page.Data = types.PricingPageData{
		ProductProCode:        p.ctr.Container.Config.App.OperationalConstants.ProductProCode,
		ProductProPrice:       fmt.Sprintf("%.2f", p.ctr.Container.Config.App.OperationalConstants.ProductProPrice),
		ActivePlan:            *activePlan,
		SubscriptionExpiresOn: subscriptionExpiredOn,
		IsTrial:               isTrial,
		ProductDescriptions: []types.ProductDescription{
			types.ProductDescription{
				Name:     "",
				Subtitle: "",
			},
		},
	}
	page.HTMX.Request.Boosted = true

	return p.ctr.RenderPage(ctx, page)
}

func (p *paymentsRoute) SuccessfullySubscribed(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.PaymentSuccess(&page)
	page.Name = templates.PageSuccessfullySubscribed
	page.HTMX.Request.Boosted = true

	return p.ctr.RenderPage(ctx, page)
}
