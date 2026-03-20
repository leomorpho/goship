package controllers_test

import (
	"testing"

	"github.com/leomorpho/goship/framework/testutil"
)

func TestLandingPageUsesFrenchLangQuery(t *testing.T) {
	s := testutil.NewTestServer(t)
	s.Get("/?lang=fr").
		AssertStatus(200).
		AssertContains(`<html lang="fr"`)
}

func TestLandingPageFallsBackToEnglishForUnsupportedLanguage(t *testing.T) {
	s := testutil.NewTestServer(t)
	s.Get("/?lang=zz-ZZ").
		AssertStatus(200).
		AssertContains(`<html lang="en"`)
}

func TestEmailSubscribePageUsesFrenchLangQuery(t *testing.T) {
	s := testutil.NewTestServer(t)
	s.Get("/emailSubscribe?lang=fr").
		AssertStatus(200).
		AssertContains(`<html lang="fr"`)
}
