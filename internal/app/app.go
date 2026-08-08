// Package app wires the application together.
package app

import "context"

// Run starts the application and blocks until ctx is canceled
// or a fatal error occurs.
func Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
