// Package rest provides HTTP handlers for the API.
package rest

import (
	"context"
	"log"
	"os"

	"github.com/labstack/echo/v4"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

const appEnvDevelopment = "development"

// AuthService defines the interface for authentication business logic consumed by AuthMiddleware.
type AuthService interface {
	GetByUUID(ctx context.Context, uuid string) (domain.User, error)
}

// AuthMiddleware returns an Echo middleware that authenticates the request.
// In development mode, it retrieves the user by DEV_USER_UUID from the DB.
// DEV_USER_UUID is read from the environment at startup; the process exits if it is empty.
// In other environments, the process exits at startup because authentication is not yet implemented.
func AuthMiddleware(appEnv string, svc AuthService) echo.MiddlewareFunc {
	var loginUserUUID string
	if appEnv == appEnvDevelopment {
		loginUserUUID = os.Getenv("DEV_USER_UUID")
		if loginUserUUID == "" {
			log.Fatal("DEV_USER_UUID is required in development")
		}
	} else {
		// TODO: ALB OIDC の JWT を検証し、payload から UUID を取得して loginUserUUID にセットする
		log.Fatalf("authentication not implemented for APP_ENV=%s", appEnv)
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, err := svc.GetByUUID(c.Request().Context(), loginUserUUID)
			if err != nil {
				return respondError(c, err)
			}

			c.Set("authUser", user)

			return next(c)
		}
	}
}
