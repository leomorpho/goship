package middleware

import (
	"os"
	"strings"

	frameworkmiddleware "github.com/leomorpho/goship/framework/web/middleware"
)

const CachedPageGroup = frameworkmiddleware.CachedPageGroup

type CachedPage = frameworkmiddleware.CachedPage

var (
	CacheControl                      = frameworkmiddleware.CacheControl
	FilterSentryErrors                = frameworkmiddleware.FilterSentryErrors
	LoadAuthenticatedUser             = frameworkmiddleware.LoadAuthenticatedUser
	LoadUser                          = frameworkmiddleware.LoadUser
	LoadValidPasswordToken            = frameworkmiddleware.LoadValidPasswordToken
	LogRequestID                      = frameworkmiddleware.LogRequestID
	RateLimit                         = frameworkmiddleware.RateLimit
	RecoverPanics                     = frameworkmiddleware.RecoverPanics
	RedirectToOnboardingIfNotComplete = frameworkmiddleware.RedirectToOnboardingIfNotComplete
	RequireAdmin                      = frameworkmiddleware.RequireAdmin
	RequireAuthentication             = frameworkmiddleware.RequireAuthentication
	RequireManagedHookSignature       = frameworkmiddleware.RequireManagedHookSignature
	RequireNoAuthentication           = frameworkmiddleware.RequireNoAuthentication
	ServeCachedPage                   = frameworkmiddleware.ServeCachedPage
	SetDeviceTypeToServe              = frameworkmiddleware.SetDeviceTypeToServe
	SetLastSeenOnline                 = frameworkmiddleware.SetLastSeenOnline
)

func userIsAdmin(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	raw := strings.TrimSpace(os.Getenv("PAGODA_ADMIN_EMAILS"))
	if raw == "" {
		return false
	}
	for _, candidate := range strings.Split(raw, ",") {
		if strings.ToLower(strings.TrimSpace(candidate)) == email {
			return true
		}
	}
	return false
}
