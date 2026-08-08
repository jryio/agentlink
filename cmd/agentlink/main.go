// Package main is the entry point for agentlink.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jryio/agentlink/internal/app"
)

var version = "dev" // set via -ldflags

func main() {
	if err := run(); err != nil {
		var exitErr *app.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.Err != nil {
				fmt.Fprintln(os.Stderr, "agentlink:", exitErr.Err)
			}
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, "agentlink:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	streams := app.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
	if err := app.Run(ctx, os.Args[1:], version, streams); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
