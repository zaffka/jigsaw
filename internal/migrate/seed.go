package migrate

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// SeedAdmin creates the first admin user if it doesn't exist.
// Email and password are read from SEED_ADMIN_EMAIL and SEED_ADMIN_PASSWORD env vars.
// Defaults: admin@jigsaw.local / changeme
func SeedAdmin(ctx context.Context, pool *pgxpool.Pool) error {
	email := os.Getenv("SEED_ADMIN_EMAIL")
	if email == "" {
		email = "admin@jigsaw.local"
	}
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if password == "" {
		password = "changeme"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, 'admin')
		ON CONFLICT (email) DO NOTHING
	`, email, string(hash))
	return err
}
