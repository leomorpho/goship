package routes

import (
	"fmt"

	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/routing/routenames"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"

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

	page.Metatags.Description = "Ship your app in record time."
	page.Metatags.Keywords = []string{"Boilerplate", "HTMX", "AlpineJS", "Javascript", "Starter kit", "Startup", "Solopreneur", "Indie Hacking"}
	data = types.LandingPage{
		AppName:           string(c.ctr.Container.Config.App.Name),
		UserSignupEnabled: c.ctr.Container.Config.App.OperationalConstants.UserSignupEnabled,

		Title:      "Ship in Record Time",
		Subtitle:   "A Go + HTMX boilerplate with all the essentials for your SaaS, AI tools, or web apps. Start earning online quickly without the hassle.",
		GetNowText: "Get Started",
		IntroTitle: "Build & Launch Without the Headaches",

		HowItWorksTitle: "How it works.",

		Quote1: "Save time on setup: Stop spending countless hours configuring email, payment gateways, and authentication. GoShip handles it for you.",
		Quote2: "Simplify your tech stack: GoShip minimizes JavaScript, focusing on the simplicity and speed of Go and HTMX, while still providing the modern features you need.",

		ExampleQuestion1: "What’s the most laughable fashion trend you have ever followed?",
		ExampleQuestion2: "What's one place you've never been to but feel drawn to visit?",
		ExampleQuestion3: "What are your strategies for maintaining intimacy with your partner?",

		AboutUsTitle1: "Championing trust and authentic connections.",
		AboutUsText1:  "We're not a faceless conglomerate. We're a dedicated small team, committed to safeguarding your privacy and fostering real relationships. Your data is encrypted, not up for sale, and can be deleted at any time.",
		AboutUsTitle2: "Our mission?",
		AboutUsText2:  "To make Chérie so successful that you won't need us for long – because you'll be immersed in enjoying real-world relationships!",

		QAItems: []types.QAItem{
			{
				Question: "What exactly is included?",
				Answer:   "You get a fully functional Go + HTMX boilerplate with built-in integrations for payment processing, authentication, database management, and more.",
			},
			{
				Question: "Can I customize the tech stack?",
				Answer:   "Yes! While GoShip is optimized for Go and HTMX, you can easily adapt it to suit your specific tech stack.",
			},
			{
				Question: "Is this a website template?",
				Answer:   "No, GoShip is a complete boilerplate with backend and frontend code ready to use.",
			},
			{
				Question: "How does GoShip compare to other boilerplates?",
				Answer:   "GoShip focuses on simplicity and efficiency by minimizing JavaScript, allowing you to build interactive apps with minimal complexity.",
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
