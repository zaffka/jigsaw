package migrate

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	pgxv5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

//go:embed sql/migrations/*.sql
var migrationsFS embed.FS

// Run applies all pending database migrations.
// Returns the current version after migration.
func Run(pool *pgxpool.Pool, log *zap.Logger) (uint, error) {
	sqlDB := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer sqlDB.Close()

	driver, err := pgxv5.WithInstance(sqlDB, &pgxv5.Config{})
	if err != nil {
		return 0, fmt.Errorf("migrate driver: %w", err)
	}

	migrationsSubFS, err := fs.Sub(migrationsFS, "sql/migrations")
	if err != nil {
		return 0, fmt.Errorf("migrate fs: %w", err)
	}

	sourceDriver, err := iofs.New(migrationsSubFS, ".")
	if err != nil {
		return 0, fmt.Errorf("migrate source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "pgx5", driver)
	if err != nil {
		return 0, fmt.Errorf("migrate instance: %w", err)
	}

	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		ver, _, _ := m.Version()
		log.Info("database migrations: no changes", zap.Uint("version", ver))
		return ver, nil
	}
	if err != nil {
		return 0, fmt.Errorf("migrate up: %w", err)
	}

	ver, _, _ := m.Version()
	log.Info("database migrations applied", zap.Uint("version", ver))
	return ver, nil
}
