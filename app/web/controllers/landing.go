package controllers

import (
	"fmt"

	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"

	"github.com/labstack/echo/v4"
)

type (
	landingPage struct {
		ctr ui.Controller
	}
)

func NewLandingPageRoute(ctr ui.Controller) landingPage {
	return landingPage{
		ctr: ctr,
	}
}

func (c *landingPage) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.LandingPage

	if page.AuthUser != nil {
		return c.ctr.Redirect(ctx, routenames.RouteNameHomeFeed)

	}

	data := viewmodels.NewLandingPage()

	page.Metatags.Description = "Opinionated Go + HTMX framework for shipping production apps fast."
	page.Metatags.Keywords = []string{"Go", "HTMX", "Templ", "Starter", "Framework", "SaaS", "CLI", "LLM"}
	data.AppName = string(c.ctr.Container.Config.App.Name)
	data.UserSignupEnabled = c.ctr.Container.Config.App.OperationalConstants.UserSignupEnabled
	data.Title = "Ship Production Apps in Go"
	data.Subtitle = "An opinionated Go + HTMX framework and starter app designed for fast delivery, clean defaults, and low-infra deployments."
	data.GetNowText = "Get Started"
	data.IntroTitle = "Build Fast Without Rebuilding Foundations"
	data.HowItWorksTitle = "How it works."
	data.Quote1 = "Start with practical defaults for routing, auth, data, and operational wiring so product work starts on day one."
	data.Quote2 = "Use Go + HTMX + Templ for server-driven UI, and scale with modules plus the ship CLI instead of ad-hoc boilerplate."
	data.ExampleQuestion1 = "What’s the most laughable fashion trend you have ever followed?"
	data.ExampleQuestion2 = "What's one place you've never been to but feel drawn to visit?"
	data.ExampleQuestion3 = "What are your strategies for maintaining intimacy with your partner?"
	data.AboutUsTitle1 = "Why this project exists."
	data.AboutUsText1 = "GoShip is the stack I use to launch projects repeatedly without re-building the same platform plumbing every time."
	data.AboutUsTitle2 = "Current mission."
	data.AboutUsText2 = "Deliver excellent developer ergonomics in Go: clear conventions, strong defaults, and fast iteration for both humans and LLM agents."
	qa1 := viewmodels.NewQAItem()
	qa1.Question = "What do I get out of the box?"
	qa1.Answer = "A production-ready Go starter with routing, auth flows, data wiring, test setup, and deployment workflows. Optional batteries are meant to be added as modules."
	qa2 := viewmodels.NewQAItem()
	qa2.Question = "Can I customize the stack?"
	qa2.Answer = "Yes. GoShip is opinionated by default but built to evolve by modules and adapters as your app requirements grow."
	qa3 := viewmodels.NewQAItem()
	qa3.Question = "Is this just a UI template?"
	qa3.Answer = "No. It is an end-to-end application foundation with backend, frontend, infra workflows, and deployment paths."
	qa4 := viewmodels.NewQAItem()
	qa4.Question = "How does it help with LLM-assisted development?"
	qa4.Answer = "The project is being shaped to be LLM-friendly: consistent structure, explicit docs, and a CLI-driven generation path that reduces repetitive manual setup."
	data.QAItems = []viewmodels.QAItem{qa1, qa2, qa3, qa4}
	data.BackgroundPhoto2lg = "https://chatbond-static.s3.us-west-002.backblazeb2.com/cherie/home/team-image-lg.jpeg"
	data.BackgroundPhoto2xl = "https://chatbond-static.s3.us-west-002.backblazeb2.com/cherie/home/team-image-xl.jpeg"

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
