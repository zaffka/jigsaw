package pgx

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnOpts struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	ConnMaxLifetime time.Duration // if set to zero - the default pgx value will be used.
}

// OpenPool initializes PostgreSQL connection pool and returns it.
func OpenPool(ctx context.Context, opts ConnOpts) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse db dsn: %w", err)
	}

	// TLS configuration is automatically handled by pgxpool.ParseConfig based on sslmode parameter in DSN.

	// Enable for Simple Protocol.
	config.ConnConfig.RuntimeParams["standard_conforming_strings"] = "on"

	// Avoid error: "simple protocol queries must be run with client_encoding=UTF8".
	config.ConnConfig.RuntimeParams["client_encoding"] = "UTF8"

	config.MaxConns = opts.MaxConns
	config.MinConns = opts.MinConns
	config.MaxConnLifetime = opts.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to init db connection pool: %w", err)
	}

	return pool, nil
}
