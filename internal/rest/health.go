// Package rest provides HTTP handlers for the API.
package rest

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

// DBPinger is a consumer-side interface for database health checks.
// *sql.DB satisfies this interface implicitly.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

type healthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// RegisterHealthHandler registers the health check route on the given Echo instance.
func RegisterHealthHandler(e *echo.Echo, db DBPinger) {
	e.GET("/health", func(c echo.Context) error {
		return healthCheck(c, db)
	})
}

func healthCheck(c echo.Context, db DBPinger) error {
	if err := db.PingContext(c.Request().Context()); err != nil {
		return c.JSON(http.StatusServiceUnavailable, healthResponse{
			Status:  "error",
			Message: "db unavailable",
		})
	}

	return c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}
