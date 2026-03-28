package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
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

	e := echo.NewWithConfig(echo.Config{
		Logger: slog.With("component", "server"),
	})

	sc := echo.StartConfig{
		Address:         ":" + getenv("PORT"),
		GracefulTimeout: 30 * time.Second,
	}

	err = sc.Start(ctx, e)
	if err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}
