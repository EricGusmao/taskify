package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EricGusmao/taskify/api"
	"github.com/EricGusmao/taskify/internal/auth"
	"github.com/EricGusmao/taskify/internal/docs"
	"github.com/EricGusmao/taskify/internal/infra"
	"github.com/EricGusmao/taskify/internal/middleware"
	"github.com/EricGusmao/taskify/internal/tasks"
	"github.com/EricGusmao/taskify/internal/teams"
	"github.com/EricGusmao/taskify/internal/users"
	"github.com/EricGusmao/taskify/internal/validate"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, getenv func(string) string) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	slog.SetDefault(slog.New(zapslog.NewHandler(logger.Core())))

	jwtSecret := getenv("JWT_SECRET")
	if len(jwtSecret) < 64 {
		return fmt.Errorf("JWT_SECRET must be at least 64 characters")
	}

	db, err := infra.NewDB(getenv("DATABASE_DSN"))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	e := echo.NewWithConfig(echo.Config{
		Logger:    slog.With("component", "server"),
		Validator: echoValidator{},
	})

	e.Use(
		echomw.Recover(),
		echomw.RequestLogger(),
		echomw.SecureWithConfig(
			echomw.SecureConfig{
				Skipper: func(c *echo.Context) bool {
					p := c.Request().URL.Path
					return p == "/docs" || p == "/api/openapi.yaml"
				},
				XSSProtection:         "0",
				ContentSecurityPolicy: "default-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'; sandbox;",
				ContentTypeNosniff:    "nosniff",
				XFrameOptions:         "DENY",
				ReferrerPolicy:        "no-referrer",
			},
		),
	)

	docsHandler := docs.NewHandler(api.Spec)
	e.GET("/api/openapi.yaml", docsHandler.ServeSpec)
	e.GET("/docs", docsHandler.ServeRedoc)

	authRepo := auth.NewUserRepository(db)
	authSvc := auth.NewService(authRepo, logger, []byte(jwtSecret))
	authHandler := auth.NewHandler(authSvc)
	auth.RegisterRoutes(e.Group("/auth"), authHandler)

	teamsRepo := teams.NewRepository(db)
	teamsUserRepo := teams.NewUserRepository(db)
	teamsSvc := teams.NewService(teamsRepo, teamsUserRepo, logger)
	teamsHandler := teams.NewHandler(teamsSvc)
	teamsGroup := e.Group("/teams")
	teamsGroup.Use(middleware.JWTAuth([]byte(jwtSecret)))
	teams.RegisterRoutes(teamsGroup, teamsHandler)

	tasksRepo := tasks.NewRepository(db)
	tasksSvc := tasks.NewService(tasksRepo, logger)
	tasksHandler := tasks.NewHandler(tasksSvc)
	tasks.RegisterRoutes(teamsGroup, tasksHandler)

	tasksGroup := e.Group("/tasks")
	tasksGroup.Use(middleware.JWTAuth([]byte(jwtSecret)))
	tasks.RegisterTaskRoutes(tasksGroup, tasksHandler)

	uploadPath := getenv("UPLOAD_PATH")
	if uploadPath == "" {
		uploadPath = "./uploads/avatars"
	}
	localStorage, err := users.NewLocalStorage(uploadPath)
	if err != nil {
		return fmt.Errorf("failed to create local storage: %w", err)
	}
	usersRepo := users.NewRepository(db)
	usersSvc := users.NewService(usersRepo, localStorage, logger)
	usersHandler := users.NewHandler(usersSvc)
	usersGroup := e.Group("/users")
	usersGroup.Use(middleware.JWTAuth([]byte(jwtSecret)))
	usersGroup.Use(echomw.BodyLimit(5 << 20))
	users.RegisterRoutes(usersGroup, usersHandler)
	e.Static("/uploads", "./uploads")

	sc := echo.StartConfig{
		Address:         ":" + getenv("PORT"),
		GracefulTimeout: 30 * time.Second,
	}

	if err := sc.Start(ctx, e); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}

// echoValidator implements echo.Validator using the shared validate package.
type echoValidator struct{}

func (echoValidator) Validate(i any) error {
	return validate.Struct(i)
}
