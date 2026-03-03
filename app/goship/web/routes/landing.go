package routes

import (
	"fmt"

	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/app/goship/web/routenames"
	"github.com/leomorpho/goship/pkg/controller"
	"github.com/leomorpho/goship/pkg/types"

	"github.com/labstack/echo/v4"
)

type (
	landingPage struct {
		ctr controller.Controller
	}
)

func NewLandingPageRoute(ctr controller.Controller) landingPage {
	return landingPage{
		ctr: ctr,
	}
}

func (c *landingPage) Get(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.LandingPage

	if page.AuthUser != nil {
		return c.ctr.Redirect(ctx, routenames.RouteNameHomeFeed)

	}

	var data types.LandingPage

	page.Metatags.Description = "Opinionated Go + HTMX framework for shipping production apps fast."
	page.Metatags.Keywords = []string{"Go", "HTMX", "Templ", "Starter", "Framework", "SaaS", "CLI", "LLM"}
	data = types.LandingPage{
		AppName:           string(c.ctr.Container.Config.App.Name),
		UserSignupEnabled: c.ctr.Container.Config.App.OperationalConstants.UserSignupEnabled,

		Title:      "Ship Production Apps in Go",
		Subtitle:   "An opinionated Go + HTMX framework and starter app designed for fast delivery, clean defaults, and low-infra deployments.",
		GetNowText: "Get Started",
		IntroTitle: "Build Fast Without Rebuilding Foundations",

		HowItWorksTitle: "How it works.",

		Quote1: "Start with practical defaults for routing, auth, data, and operational wiring so product work starts on day one.",
		Quote2: "Use Go + HTMX + Templ for server-driven UI, and scale with modules plus the ship CLI instead of ad-hoc boilerplate.",

		ExampleQuestion1: "What’s the most laughable fashion trend you have ever followed?",
		ExampleQuestion2: "What's one place you've never been to but feel drawn to visit?",
		ExampleQuestion3: "What are your strategies for maintaining intimacy with your partner?",

		AboutUsTitle1: "Why this project exists.",
		AboutUsText1:  "GoShip is the stack I use to launch projects repeatedly without re-building the same platform plumbing every time.",
		AboutUsTitle2: "Current mission.",
		AboutUsText2:  "Deliver excellent developer ergonomics in Go: clear conventions, strong defaults, and fast iteration for both humans and LLM agents.",

		QAItems: []types.QAItem{
			{
				Question: "What do I get out of the box?",
				Answer:   "A production-ready Go starter with routing, auth flows, data wiring, test setup, and deployment workflows. Optional batteries are meant to be added as modules.",
			},
			{
				Question: "Can I customize the stack?",
				Answer:   "Yes. GoShip is opinionated by default but built to evolve by modules and adapters as your app requirements grow.",
			},
			{
				Question: "Is this just a UI template?",
				Answer:   "No. It is an end-to-end application foundation with backend, frontend, infra workflows, and deployment paths.",
			},
			{
				Question: "How does it help with LLM-assisted development?",
				Answer:   "The project is being shaped to be LLM-friendly: consistent structure, explicit docs, and a CLI-driven generation path that reduces repetitive manual setup.",
			},
		},

		BackgroundPhoto2lg: "https://chatbond-static.s3.us-west-002.backblazeb2.com/cherie/home/team-image-lg.jpeg",
		BackgroundPhoto2xl: "https://chatbond-static.s3.us-west-002.backblazeb2.com/cherie/home/team-image-xl.jpeg",
	}

	data.UserSignupEnabled = c.ctr.Container.Config.App.OperationalConstants.UserSignupEnabledOnLandingPage
	data.ContactEmail = c.ctr.Container.Config.Mail.FromAddress
	data.ProductProCode = c.ctr.Container.Config.App.OperationalConstants.ProductProCode
	data.ProductProPrice = fmt.Sprintf("%.2f", c.ctr.Container.Config.App.OperationalConstants.ProductProPrice)
	data.IsPaymentEnabled = c.ctr.Container.Config.App.OperationalConstants.PaymentsEnabled
	page.Data = data
	page.Name = templates.PageLanding
	page.Component = pages.LandingPage(&page)

	// if c.ctr.Container.Config.App.Environment == config.EnvProduction {
	// 	page.Cache.Enabled = true
	// } else {
	// 	page.Cache.Enabled = false
	// }

	return c.ctr.RenderPage(ctx, page)
}
