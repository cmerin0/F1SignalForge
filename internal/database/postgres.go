// Package database owns the application's PostgreSQL connection pool.
package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultMaxConnections int32 = 4

// Config contains only the database settings needed to create a pool.
// Keeping configuration here prevents PostgreSQL details spreading across
// HTTP handlers and business logic.
type Config struct {
	URL      string
	MaxConns int32
}

// NewPool opens and verifies a PostgreSQL connection pool.
//
// It performs Ping before returning so the application can fail during startup
// rather than reporting itself as healthy and failing on its first request.
func NewPool(ctx context.Context, config Config) (*pgxpool.Pool, error) {

	// Validate the configuration before attempting to open a pool. A missing URL
	if strings.TrimSpace(config.URL) == "" {
		return nil, errors.New("database URL is required")
	}

	// Validate the URL by parsing it into a pgxpool.Config. This ensures that
	// the URL is valid and that any default values are applied before opening a pool
	poolConfig, err := pgxpool.ParseConfig(config.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	// Apply the maximum connection limit to the pool configuration.
	// A zero or negative value is treated as unlimited,
	// which is not a safe default for a production service.
	poolConfig.MaxConns = config.MaxConns
	if poolConfig.MaxConns <= 0 {
		// A conservative default matters later in Kubernetes: every API pod
		// owns a pool, so an unlimited pool could overwhelm PostgreSQL when
		// the deployment scales out.
		poolConfig.MaxConns = defaultMaxConnections
	}

	// Open the pool and verify connectivity with a Ping. The pool is returned
	// only if it is successfully opened and the database is reachable.
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL pool: %w", err)
	}

	// Ping the database to verify connectivity. If the ping fails, close the pool
	// and return an error. This ensures that the application does not start
	// with a non-functional database connection.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return pool, nil
}
