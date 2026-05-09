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
	GetUser(ctx context.Context, id uint64) (*domain.User, error)
}

// UserHandler handles HTTP requests for the user endpoints.
type UserHandler struct {
	Service UserService
}

// NewUserHandler registers the user routes on the given Echo router group.
func NewUserHandler(g *echo.Group, svc UserService) {
	h := &UserHandler{Service: svc}
	g.GET("/users", h.ListUsers)
	g.GET("/users/:id", h.GetUser)
}

type userListResponse struct {
	Users []domain.User `json:"users"`
	Total int           `json:"total"`
}

// GetUser handles GET /api/v1/users/:id.
func (h *UserHandler) GetUser(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	u, err := h.Service.GetUser(ctx, id)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(http.StatusOK, u)
}

// ListUsers handles GET /api/v1/users.
func (h *UserHandler) ListUsers(c echo.Context) error {
	ctx := c.Request().Context()

	limit, limitErr := parseLimit(c.QueryParam("limit"))
	if limitErr != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	offset, offsetErr := parseOffset(c.QueryParam("offset"))
	if offsetErr != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	q := c.QueryParam("q")

	users, total, err := h.Service.ListUsers(ctx, q, limit, offset)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(http.StatusOK, userListResponse{Users: users, Total: total})
}
