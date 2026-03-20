package controllers_test

import (
	"strings"
	"testing"

	"github.com/leomorpho/goship/framework/testutil"
)

func TestLoginPageRendersLanguageSwitcherWithRoutePreservingLinks(t *testing.T) {
	s := testutil.NewTestServer(t)
	s.Get("/user/login?next=/welcome/preferences").
		AssertStatus(200).
		AssertContains(`data-component="language-switcher"`).
		AssertContains("lang=fr").
		AssertContains("next=%2Fwelcome%2Fpreferences")
}

func TestLanguageSwitcherQuerySetsLangCookieOnPageRoute(t *testing.T) {
	s := testutil.NewTestServer(t)
	resp := s.Get("/user/login?lang=fr").AssertStatus(200)

	setCookie := strings.Join(resp.Header.Values("Set-Cookie"), ";")
	if !strings.Contains(setCookie, "lang=fr") {
		t.Fatalf("expected lang cookie to be set from query switch, got %q", setCookie)
	}
}
