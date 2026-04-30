// Package rest provides HTTP handlers for the API.
package rest

import (
	"errors"
	"net/http"

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
