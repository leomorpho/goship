package middleware

import appmiddleware "github.com/leomorpho/goship/app/web/middleware"

const CachedPageGroup = appmiddleware.CachedPageGroup

type CachedPage = appmiddleware.CachedPage

var (
	CacheControl                      = appmiddleware.CacheControl
	FilterSentryErrors                = appmiddleware.FilterSentryErrors
	LoadAuthenticatedUser             = appmiddleware.LoadAuthenticatedUser
	LoadUser                          = appmiddleware.LoadUser
	LoadValidPasswordToken            = appmiddleware.LoadValidPasswordToken
	LogRequestID                      = appmiddleware.LogRequestID
	RateLimit                         = appmiddleware.RateLimit
	RecoverPanics                     = appmiddleware.RecoverPanics
	RedirectToOnboardingIfNotComplete = appmiddleware.RedirectToOnboardingIfNotComplete
	RequireAdmin                      = appmiddleware.RequireAdmin
	RequireAuthentication             = appmiddleware.RequireAuthentication
	RequireManagedHookSignature       = appmiddleware.RequireManagedHookSignature
	RequireNoAuthentication           = appmiddleware.RequireNoAuthentication
	ServeCachedPage                   = appmiddleware.ServeCachedPage
	SetDeviceTypeToServe              = appmiddleware.SetDeviceTypeToServe
	SetLastSeenOnline                 = appmiddleware.SetLastSeenOnline
)
