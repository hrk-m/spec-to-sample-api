// Package rest provides HTTP handlers for the API.
package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// allowedHeaders is the set of request header names (canonical Pascal Case) that are
// recorded in the access log. Any header not in this list is silently dropped.
// Allowed: Referer, Sec-Ch-Ua, Sec-Ch-Ua-Mobile, Sec-Ch-Ua-Platform.
// Comparison is case-insensitive; see filterAllowedHeaders.
var allowedHeaders = []string{
	"Referer",
	"Sec-Ch-Ua",
	"Sec-Ch-Ua-Mobile",
	"Sec-Ch-Ua-Platform",
}

// maxBodyBytes is the maximum number of bytes recorded for request and response bodies.
// Bodies larger than this limit are replaced with a truncation marker.
const maxBodyBytes = 4096

// sensitiveBodyKeys is the set of JSON key names (lower-cased) whose values are
// replaced with [REDACTED] before logging. Comparison is case-insensitive.
var sensitiveBodyKeys = map[string]struct{}{
	"password":      {},
	"token":         {},
	"access_token":  {},
	"refresh_token": {},
	"api_key":       {},
	"secret":        {},
	"authorization": {},
}

// AccessLogMiddleware returns an Echo middleware that logs each request as structured JSON.
// It records the endpoint, authenticated user UUID, latency, status code, an
// allow-listed subset of request headers, and optionally the request body.
// Only headers in allowedHeaders are included in the log; all others (including
// Authorization and Cookie) are silently dropped.
// The request body is only logged when Content-Type contains application/json,
// is capped at 4096 bytes, and has sensitive keys redacted.
// The middleware must be registered inside AuthMiddleware so that authUser is already set.
func AccessLogMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			loginUser := ""
			if u, ok := c.Get("authUser").(domain.User); ok {
				loginUser = u.UUID
			}

			requestId := c.Request().Header.Get("X-Request-Id")
			if requestId != "" {
				c.Response().Header().Set("X-Request-ID", requestId)
			}
			c.Response().Header().Set("X-Login-User", loginUser)

			// Capture request body (restores body for downstream handlers).
			reqRaw := readRequestBody(c.Request())

			nextErr := next(c)

			endpoint := c.Request().Method + " " + c.Request().URL.Path
			status := c.Response().Status
			latencyS := time.Since(start).Seconds()

			// When the handler returns an error without calling WriteHeader, status is 0.
			// Treat this as an internal server error to ensure correct log level assignment.
			if nextErr != nil && status == 0 {
				status = http.StatusInternalServerError
			}

			filteredHeaders := filterAllowedHeaders(c.Request().Header)

			attrs := []any{
				"endpoint", endpoint,
				"login_user", loginUser,
				"latency_s", latencyS,
				"status", status,
				"header", filteredHeaders,
				"request_id", requestId,
			}

			// Append request_body if available.
			if reqRaw != nil {
				var sizeOverride int64
				if len(reqRaw) > maxBodyBytes {
					sizeOverride = c.Request().ContentLength
					if sizeOverride < int64(len(reqRaw)) {
						sizeOverride = int64(len(reqRaw))
					}
				}
				attrs = append(attrs, "request_body", buildBodyAttr(reqRaw, sizeOverride))
			}

			logFn := logger.Info
			if status >= http.StatusInternalServerError {
				errorMsg := ""
				if nextErr != nil {
					errorMsg = nextErr.Error()
				}

				attrs = append(attrs, "error", errorMsg)
				logFn = logger.Error
			}

			logFn("access", attrs...)

			return nextErr
		}
	}
}

// filterAllowedHeaders returns a new http.Header containing only the headers whose
// names appear in allowedHeaders. Comparison is case-insensitive.
// Headers not in the allow-list (e.g. Authorization, Cookie, Sec-Fetch-*) are
// omitted entirely — they are not masked, just excluded.
func filterAllowedHeaders(src http.Header) http.Header {
	filtered := make(http.Header, len(allowedHeaders))

	for k, v := range src {
		for _, allowed := range allowedHeaders {
			if strings.EqualFold(k, allowed) {
				vals := make([]string, len(v))
				copy(vals, v)
				filtered[k] = vals
				break
			}
		}
	}

	return filtered
}

// isJSONContentType reports whether ct (a Content-Type header value) indicates
// an application/json body.
func isJSONContentType(ct string) bool {
	return strings.Contains(ct, "application/json")
}

// readRequestBody reads up to maxBodyBytes+1 from the request body, restores the
// body on the request so handlers can read it again, and returns the raw bytes.
// If the content-type is not application/json or the body is empty, nil is returned.
// The caller must check len(buf) > maxBodyBytes to detect truncation.
func readRequestBody(r *http.Request) []byte {
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		return nil
	}

	if r.Body == nil {
		return nil
	}

	// Read at most maxBodyBytes+1 to detect truncation.
	limited := io.LimitReader(r.Body, int64(maxBodyBytes)+1)
	buf, err := io.ReadAll(limited)
	// Always restore the body so the downstream handler can read it.
	r.Body = io.NopCloser(bytes.NewReader(buf))
	if err != nil || len(buf) == 0 {
		return nil
	}

	return buf
}

// buildBodyAttr converts raw body bytes into a log-safe value according to the
// body recording policy: truncation marker, parse-error marker, or masked JSON.
// sizeOverride is the actual body size to use in the truncation marker; pass 0
// to use len(raw) (used when ContentLength is unavailable).
func buildBodyAttr(raw []byte, sizeOverride int64) any {
	if len(raw) > maxBodyBytes {
		size := sizeOverride
		if size <= int64(len(raw)) {
			size = int64(len(raw))
		}
		return map[string]any{
			"_truncated": true,
			"size_bytes": size,
		}
	}

	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return map[string]any{"_parse_error": true}
	}

	return maskSensitiveValues(parsed)
}

// maskSensitiveValues recursively traverses v and replaces the values of keys
// matching sensitiveBodyKeys (case-insensitive) with "[REDACTED]".
func maskSensitiveValues(v any) any {
	switch val := v.(type) {
	case map[string]any:
		masked := make(map[string]any, len(val))
		for k, vv := range val {
			if _, isSensitive := sensitiveBodyKeys[strings.ToLower(k)]; isSensitive {
				masked[k] = "[REDACTED]"
			} else {
				masked[k] = maskSensitiveValues(vv)
			}
		}
		return masked
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = maskSensitiveValues(item)
		}
		return out
	default:
		return v
	}
}
