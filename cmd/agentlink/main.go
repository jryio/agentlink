// Package main is the entry point for agentlink.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jryio/agentlink/internal/app"
)

var version = "dev" // set via -ldflags

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	logger.InfoContext(ctx, "starting", "version", version)

	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("app: %w", err)
	}
	return nil
}
