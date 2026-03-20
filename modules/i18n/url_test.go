package i18n

import "testing"

func TestLanguageSwitcherURL_PreservesPathAndQuery(t *testing.T) {
	got := LanguageSwitcherURL("/user/login?next=%2Fwelcome%2Fpreferences", "fr")
	want := "/user/login?lang=fr&next=%2Fwelcome%2Fpreferences"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLanguageSwitcherURL_DefaultsWhenInputIsEmpty(t *testing.T) {
	got := LanguageSwitcherURL("", "")
	if got != "/?lang=en" {
		t.Fatalf("got %q, want /?lang=en", got)
	}
}
