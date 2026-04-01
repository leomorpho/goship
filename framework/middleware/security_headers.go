package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/v2/config"
)

const cspNonceContextKey = "csp_nonce"

// CSPNonce returns the request-scoped CSP nonce set by SecurityHeaders middleware.
func CSPNonce(c echo.Context) string {
	if c == nil {
		return ""
	}
	nonce, ok := c.Get(cspNonceContextKey).(string)
	if !ok {
		return ""
	}
	return nonce
}

// SecurityHeaders adds baseline secure defaults, including CSP with request-scoped nonce.
func SecurityHeaders(cfg config.SecurityConfig, appEnv string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !cfg.Headers.Enabled {
				return next(c)
			}

			nonce, err := generateCSPNonce()
			if err != nil {
				return err
			}
			c.Set(cspNonceContextKey, nonce)

			headers := c.Response().Header()
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "SAMEORIGIN")
			headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			headers.Set("X-XSS-Protection", "0")

			if cfg.Headers.HSTS {
				headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			csp := strings.TrimSpace(cfg.Headers.CSP)
			if csp == "" {
				csp = defaultCSP(nonce, appEnv)
			}
			headers.Set("Content-Security-Policy", csp)

			return next(c)
		}
	}
}

func generateCSPNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func defaultCSP(nonce, appEnv string) string {
	devConnectSrc := ""
	if strings.EqualFold(appEnv, string(config.EnvDevelop)) || strings.EqualFold(appEnv, string(config.EnvLocal)) {
		devConnectSrc = " ws://localhost:5173 http://localhost:5173"
	}

	return fmt.Sprintf(
		"default-src 'self'; "+
			"base-uri 'self'; "+
			"frame-ancestors 'self'; "+
			"object-src 'none'; "+
			"form-action 'self'; "+
			"img-src 'self' data: https:; "+
			"font-src 'self' data: https:; "+
			"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://unpkg.com https://cdnjs.cloudflare.com; "+
			"script-src 'self' 'nonce-%s' https://cdn.jsdelivr.net https://unpkg.com https://js.stripe.com https://cdnjs.cloudflare.com https://d3js.org https://js.sentry-cdn.com; "+
			"script-src-attr 'unsafe-inline'; "+
			"connect-src 'self' wss: ws:%s; "+
			"worker-src 'self' blob:; "+
			"frame-src 'self' https://js.stripe.com",
		nonce,
		devConnectSrc,
	)
}
