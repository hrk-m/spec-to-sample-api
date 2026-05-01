// Package rest provides HTTP handlers for the API.
package rest

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// AccessLogMiddleware returns an Echo middleware that logs each request as structured JSON.
// It records the endpoint, authenticated user UUID, latency, status code, and request headers.
// The Authorization header value is masked as [REDACTED] before logging.
// The middleware must be registered inside AuthMiddleware so that authUser is already set.
func AccessLogMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			loginUser := ""
			if u, ok := c.Get("authUser").(domain.User); ok {
				loginUser = u.UUID
			}

			c.Response().Header().Set("X-Login-User", loginUser)

			nextErr := next(c)

			endpoint := c.Request().Method + " " + c.Request().URL.Path
			status := c.Response().Status
			latencyS := time.Since(start).Seconds()

			maskedHeaders := maskAuthorizationHeader(c.Request().Header)

			logger.Info("access",
				"endpoint", endpoint,
				"login_user", loginUser,
				"latency_s", latencyS,
				"status", status,
				"header", maskedHeaders,
			)

			return nextErr
		}
	}
}

// maskAuthorizationHeader returns a copy of headers with the Authorization value replaced by [REDACTED].
func maskAuthorizationHeader(src http.Header) http.Header {
	masked := make(http.Header, len(src))

	for k, v := range src {
		if strings.EqualFold(k, "Authorization") {
			masked[k] = []string{"[REDACTED]"}
		} else {
			vals := make([]string, len(v))
			copy(vals, v)
			masked[k] = vals
		}
	}

	return masked
}
