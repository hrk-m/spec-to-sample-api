// Package rest provides HTTP handlers for the API.
package rest

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// AuthHandler handles HTTP requests for the auth endpoints.
type AuthHandler struct {
}

// NewAuthHandler registers the auth routes on the given Echo instance.
func NewAuthHandler(g *echo.Group) {
	h := &AuthHandler{}
	g.GET("/me", h.GetMe)
}

// GetMe handles GET /api/v1/me.
func (h *AuthHandler) GetMe(c echo.Context) error {
	user, ok := c.Get("authUser").(domain.User)
	if !ok {
		return respondError(c, domain.ErrUnauthorized)
	}

	return c.JSON(http.StatusOK, user)
}
