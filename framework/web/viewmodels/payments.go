package viewmodels

import "github.com/leomorpho/goship/framework/web/ui"

type (
	PaymentProcessorPublicKey struct {
		Key string `json:"key"`
	}

	CreateCheckoutSessionForm struct {
		Submission ui.FormSubmission
		PriceID    string `form:"price_id", validate:required`
	}

	ProductDescription struct {
		Name     string
		Subtitle string
		Price    string
		Points   []string
		PlanKey  string
	}
	PricingPageData struct {
		ProductProCode        string
		ProductProPrice       string
		FreePlanKey           string
		DefaultPaidPlanKey    string
		ActivePlanKey         string
		ActivePlanIsPaid      bool
		IsTrial               bool
		HasSubscriptionExpiry bool
		SubscriptionExpiresOn string
		ProductDescriptions   []ProductDescription
	}
)
