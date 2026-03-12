package di

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/zaffka/jigsaw/pkg/di/config"
	"github.com/zaffka/jigsaw/pkg/pgx"
)

// NewDBConfig returns default database configuration.
func NewDBConfig(dsn string) config.Database {
	return config.Database{
		DSN:             dsn,
		MaxConns:        5,
		MinConns:        1,
		ConnMaxLifetime: 30 * time.Minute,
		PingTimeout:     5 * time.Second,
	}
}

// RegisterDB registers a pgxpool.Pool in the DI container.
func RegisterDB(ctx context.Context, injector do.Injector, cfg config.Database) error {
	if cfg.Name == "" {
		cfg.Name = "default"
	}

	do.ProvideNamed(injector, cfg.Name, func(i do.Injector) (*pgxpool.Pool, error) {
		log := do.MustInvoke[*zap.Logger](i)

		pool, err := pgx.OpenPool(ctx, pgx.ConnOpts{
			DSN:             cfg.DSN,
			MaxConns:        cfg.MaxConns,
			MinConns:        cfg.MinConns,
			ConnMaxLifetime: cfg.ConnMaxLifetime,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to open pgx pool: %w", err)
		}

		pingCtx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to ping database (timeout: %v): %w", cfg.PingTimeout, err)
		}

		log.Info(cfg.Name + " database registered and connected")

		RegisterCleanup(injector, func() {
			log.Info("shutting down " + cfg.Name + " database connection")
			pool.Close()
		})

		return pool, nil
	})

	return nil
}
