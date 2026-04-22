// Package rest provides HTTP handlers for the API.
package rest

import (
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
	switch err {
	case domain.ErrBadParamInput:
		return http.StatusBadRequest
	case domain.ErrInternalServerError:
		return http.StatusInternalServerError
	case domain.ErrNotFound:
		return http.StatusNotFound
	case domain.ErrConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
