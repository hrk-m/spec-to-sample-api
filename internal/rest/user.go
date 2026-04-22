// Package rest provides HTTP handlers for the API.
package rest

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// UserService defines the interface for the user use case.
type UserService interface {
	ListUsers(ctx context.Context, q string, limit, offset int) ([]domain.User, int, error)
}

// UserHandler handles HTTP requests for the user endpoints.
type UserHandler struct {
	Service UserService
}

// NewUserHandler registers the user routes on the given Echo router group.
func NewUserHandler(g *echo.Group, svc UserService) {
	h := &UserHandler{Service: svc}
	g.GET("/users", h.ListUsers)
}

type userListResponse struct {
	Users []domain.User `json:"users"`
	Total int           `json:"total"`
}

// ListUsers handles GET /api/v1/users.
func (h *UserHandler) ListUsers(c echo.Context) error {
	ctx := c.Request().Context()

	limit, limitErr := parseLimit(c.QueryParam("limit"))
	if limitErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	offset, offsetErr := parseOffset(c.QueryParam("offset"))
	if offsetErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	q := c.QueryParam("q")

	users, total, err := h.Service.ListUsers(ctx, q, limit, offset)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, userListResponse{Users: users, Total: total})
}
