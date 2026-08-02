package telemetry

import (
	"context"
	"errors"
)

// ErrRaceCarNotFound means telemetry arrived for a car that has not been
// registered in SignalForge. It is a client-facing condition, not a server
// failure, so the HTTP layer will return 404 rather than 500.
var ErrRaceCarNotFound = errors.New("race car not found")

// Repository defines the persistence behavior required by telemetry ingestion.
// The HTTP package depends on this interface, not on PostgreSQL directly.
type Repository interface {
	Store(context.Context, Event) error
}

// RepositoryFunc adapts a function to Repository. It keeps HTTP tests small
// and avoids requiring PostgreSQL for route-level tests.
type RepositoryFunc func(context.Context, Event) error

// Store method calls the adapted function with the given context and event.
func (function RepositoryFunc) Store(ctx context.Context, event Event) error {
	return function(ctx, event)
}
