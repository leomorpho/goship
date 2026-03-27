package middleware

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	appcontext "github.com/leomorpho/goship/framework/appcontext"
	"github.com/leomorpho/goship/framework/ratelimit"
)

// RateLimit limits requests by route+method and caller identity.
// Authenticated users are keyed by user ID; anonymous requests are keyed by IP.
func RateLimit(store ratelimit.Store, max int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := fmt.Sprintf("%s:%s:%s", c.Request().Method, rateLimitPath(c), rateLimitActor(c))
			decision, err := store.Allow(key, max, window)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			if decision.Allowed {
				return next(c)
			}

			retryAfter := retryAfterSeconds(decision.RetryAfter)
			c.Response().Header().Set(echo.HeaderRetryAfter, strconv.Itoa(retryAfter))
			return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
		}
	}
}

func rateLimitPath(c echo.Context) string {
	if path := strings.TrimSpace(c.Path()); path != "" {
		return path
	}
	if c.Request() != nil && c.Request().URL != nil {
		return c.Request().URL.Path
	}
	return ""
}

func rateLimitActor(c echo.Context) string {
	if userID, ok := c.Get(appcontext.AuthenticatedUserIDKey).(int); ok && userID > 0 {
		return fmt.Sprintf("user:%d", userID)
	}
	ip := strings.TrimSpace(c.RealIP())
	if ip == "" && c.Request() != nil {
		remoteAddr := strings.TrimSpace(c.Request().RemoteAddr)
		if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
			ip = host
		} else {
			ip = remoteAddr
		}
	}
	if ip == "" {
		ip = "unknown"
	}
	return "ip:" + ip
}

func retryAfterSeconds(retryAfter time.Duration) int {
	if retryAfter <= 0 {
		return 1
	}
	return int(math.Ceil(retryAfter.Seconds()))
}
