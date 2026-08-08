package main

import (
	"context"
	"errors"
	"testing"

	"github.com/jryio/agentlink/internal/app"
)

func TestCommandErrorReturnsSilentFailureForCancellation(t *testing.T) {
	t.Parallel()

	err := commandError(context.Canceled)
	var exitErr *app.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 || exitErr.Err != nil {
		t.Fatalf("commandError(context.Canceled) = %#v, want silent exit status 1", err)
	}
}
