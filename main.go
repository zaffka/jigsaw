package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/do/v2"
	"github.com/spf13/viper"
	"github.com/zaffka/jigsaw/internal/migrate"
	"github.com/zaffka/jigsaw/pkg/di"
	"go.uber.org/zap"
)

var serviceVersion = "0.0.0"

func main() {
	ctx, stopFn := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopFn()

	// Initialize viper for config reading
	viper.AutomaticEnv()

	// Create DI container
	cfgReader := di.NewViperConfigReader()
	appConfig := di.AppConfig{
		ServiceName:    "jigsaw",
		ServiceVersion: serviceVersion,
	}

	container, err := di.InitializeContainer(ctx, cfgReader, appConfig)
	if err != nil {
		print(err.Error())
		os.Exit(-1)
	}

	// Invoke logger and database pool from container
	log, err := do.Invoke[*zap.Logger](container)
	if err != nil {
		print(err.Error())
		os.Exit(-1)
	}

	pool, err := do.InvokeNamed[*pgxpool.Pool](container, "default")
	if err != nil {
		log.Error("failed to invoke pgxpool", zap.Error(err))

		return
	}

	// Run migrations
	if _, err := migrate.Run(pool, log); err != nil {
		log.Error("failed to run migrations", zap.Error(err))

		return
	}

	<-ctx.Done()
}
