// Package main is the entry point for the sample-api server.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "github.com/go-sql-driver/mysql"

	authSvc "github.com/hrk-m/spec-to-dev-workflow/sample-api/auth"
	groupSvc "github.com/hrk-m/spec-to-dev-workflow/sample-api/group"
	mysqlRepo "github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/repository/mysql"
	"github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/rest"
	userSvc "github.com/hrk-m/spec-to-dev-workflow/sample-api/user"
)

const (
	dbStartupTimeout = 30 * time.Second
	dbRetryInterval  = time.Second
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func waitForMySQL(ctx context.Context, db rest.DBPinger, retryInterval time.Duration, logger *log.Logger) error {
	if logger == nil {
		logger = log.Default()
	}

	var lastErr error

	for {
		err := db.PingContext(ctx)
		if err == nil {
			return nil
		}
		lastErr = err

		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("timed out waiting for MySQL: %w", lastErr)
		case <-timer.C:
			logger.Printf("waiting for MySQL to become ready: %v", lastErr)
			timer.Stop()
		}
	}
}

func main() {
	host := getEnv("MYSQL_HOST", "localhost")
	port := getEnv("MYSQL_PORT", "3306")
	user := getEnv("MYSQL_USER", "root")
	pass := getEnv("MYSQL_PASSWORD", "password")
	dbname := getEnv("MYSQL_DATABASE", "sample")
	listenPort := getEnv("PORT", "8080")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), dbStartupTimeout)
	defer cancel()

	if err := waitForMySQL(ctx, db, dbRetryInterval, log.Default()); err != nil {
		log.Fatal("failed to connect to MySQL: ", err)
	}

	appEnv := getEnv("APP_ENV", "")
	if appEnv == "" {
		log.Fatal("APP_ENV is required")
	}
	// defaultUserUUID の取得・バリデーションは AuthMiddleware 内で行う

	e := echo.New()
	e.Use(middleware.CORS())

	rest.RegisterHealthHandler(e, db)

	groupRepo := mysqlRepo.NewGroupRepository(db)
	userRepo := mysqlRepo.NewUserRepository(db)
	gSvc := groupSvc.NewService(groupRepo, userRepo)
	uSvc := userSvc.NewService(userRepo)

	apiGroup := e.Group("/api/v1")
	aSvc := authSvc.NewService(userRepo)
	apiGroup.Use(rest.AuthMiddleware(appEnv, aSvc))
	rest.NewAuthHandler(apiGroup)
	rest.NewGroupHandler(apiGroup, gSvc)
	rest.NewUserHandler(apiGroup, uSvc)

	e.Logger.Fatal(e.Start(":" + listenPort))
}
