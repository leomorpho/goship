package types

import "html/template"

type LandingPage struct {
	AppName           string
	UserSignupEnabled bool

	Title      string
	Subtitle   string
	GetNowText string

	IntroTitle string
	IntroText  template.HTML

	OtherAppPitchTitle string
	OtherAppPitchText  string
	OtherAppURL        string
	OtherAppName       string

	HowItWorksTitle string

	Quote1 string
	Quote2 string

	ExampleQuestion1 string
	ExampleQuestion2 string
	ExampleQuestion3 string

	AboutUsTitle1 string
	AboutUsText1  string
	AboutUsTitle2 string
	AboutUsText2  string

	QAItems []QAItem

	HeroSmImageURL     string
	HeroMdImageURL     string
	HeroLgImageURL     string
	BackgroundPhoto2lg string
	BackgroundPhoto2xl string

	IsPaymentEnabled bool
	ProductProCode   string
	ProductProPrice  string

	ContactEmail string
}

type QAItem struct {
	Question string
	Answer   string
}
