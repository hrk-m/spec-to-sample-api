// Package rest provides HTTP handlers for the API.
package rest

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// ResponseError represents a JSON error response body.
type ResponseError struct {
	Message string `json:"message"`
}

// getStatusCode maps domain sentinel errors to HTTP status codes.
func getStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	switch {
	case errors.Is(err, domain.ErrBadParamInput):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, domain.ErrInternalServerError):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// respondError writes a ResponseError JSON body with the HTTP status mapped from err.
// Use this for domain sentinel errors. For non-domain messages with a known status
// (e.g. Bind / validator failures), use badRequest instead.
func respondError(c echo.Context, err error) error {
	return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
}

// badRequest writes a 400 ResponseError JSON body with the given message.
// Use this for non-domain errors whose message must be passed through verbatim
// (e.g. echo.Bind / json.Unmarshal failures).
func badRequest(c echo.Context, message string) error {
	return c.JSON(http.StatusBadRequest, ResponseError{Message: message})
}
