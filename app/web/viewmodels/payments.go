package viewmodels

import (
	"time"

	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/domain"
)

type (
	PaymentProcessorPublicKey struct {
		Key string `json:"key"`
	}

	CreateCheckoutSessionForm struct {
		Submission ui.FormSubmission
		PriceID    string `form:"price_id", validate:required`
	}

	ProductDescription struct {
		Name        string
		Subtitle    string
		Price       string
		Points      []string
		ProductType domain.ProductType
	}
	PricingPageData struct {
		ProductProCode        string
		ProductProPrice       string
		ActivePlan            domain.ProductType
		IsTrial               bool
		SubscriptionExpiresOn *time.Time
		ProductDescriptions   []ProductDescription
	}
)
