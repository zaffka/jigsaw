// Package di provides dependency injection container setup and management.
//
// This package implements the Dependency Injection pattern using the samber/do library.
// It provides centralized service registration, configuration validation, and lifecycle management.
//
// Usage:
//
//	cfgReader := di.NewViperConfigReader()
//	appConfig := di.AppConfig{ServiceName: "jigsaw", ServiceVersion: "1.0.0"}
//	container, err := di.InitializeContainer(ctx, cfgReader, appConfig)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	logger := do.MustInvoke[*zap.Logger](container)
//	db := do.MustInvoke[*pgxpool.Pool](container)
package di

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"
	"github.com/zaffka/jigsaw/pkg/di/config"
)

// ConfigReader interface for reading configuration values.
type ConfigReader interface {
	GetString(key string) string
	GetDuration(key string) time.Duration
	GetBool(key string) bool
}

// AppConfig holds application-level configuration.
type AppConfig struct {
	ServiceName    string
	ServiceVersion string
}

// InitializeContainer creates and configures the DI container with all application dependencies.
func InitializeContainer(ctx context.Context, cfgReader ConfigReader, appConfig AppConfig) (*do.RootScope, error) {
	container := do.New()

	commonConfig, err := buildCommonConfig(cfgReader, appConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build common config: %w", err)
	}

	if err := RegisterCommonServices(ctx, container, commonConfig); err != nil {
		return nil, fmt.Errorf("failed to register common services: %w", err)
	}

	// Register S3 client
	s3Config := config.S3{
		Endpoint:   cfgReader.GetString("S3_ENDPOINT"),
		AccessKey:  cfgReader.GetString("S3_ACCESS_KEY"),
		SecretKey:  cfgReader.GetString("S3_SECRET_KEY"),
		BucketName: cfgReader.GetString("S3_BUCKET_NAME"),
		Region:     cfgReader.GetString("S3_REGION"),
	}

	if err := s3Config.Validate(); err != nil {
		return nil, fmt.Errorf("s3 config validation: %w", err)
	}

	if err := RegisterS3Config(container, s3Config); err != nil {
		return nil, fmt.Errorf("failed to register s3 config: %w", err)
	}

	if err := RegisterS3Client(ctx, container, s3Config); err != nil {
		return nil, fmt.Errorf("failed to register s3 client: %w", err)
	}

	// Register HTTP server config for workers
	if err := RegisterHTTPServerConfig(container, commonConfig.HTTP); err != nil {
		return nil, fmt.Errorf("failed to register http server config: %w", err)
	}

	return container, nil
}

// RegisterCommonServices registers common services in the DI container.
func RegisterCommonServices(ctx context.Context, injector do.Injector, cfg config.CommonServices) error {
	if err := RegisterServeMux(injector); err != nil {
		return fmt.Errorf("failed to register serve mux: %w", err)
	}

	if err := RegisterHTTPServer(ctx, injector, cfg.HTTP); err != nil {
		return fmt.Errorf("failed to register http server: %w", err)
	}

	if err := RegisterLogger(injector, cfg.Logger); err != nil {
		return fmt.Errorf("failed to register logger: %w", err)
	}

	if err := RegisterDB(ctx, injector, cfg.Database); err != nil {
		return fmt.Errorf("failed to register database: %w", err)
	}

	return nil
}

func buildCommonConfig(cfgReader ConfigReader, appConfig AppConfig) (config.CommonServices, error) {
	cfg := config.CommonServices{
		Logger: config.Logger{
			ServiceName:      appConfig.ServiceName,
			Version:          appConfig.ServiceVersion,
			LogLevel:         cfgReader.GetString("LOG_LEVEL"),
			LogFormat:        cfgReader.GetString("LOG_FMT"),
			LogFormatDefault: "console",
		},
		Database: NewDBConfig(cfgReader.GetString("DB_DSN")),
		HTTP:     NewHTTPConfig(cfgReader.GetString("PORT")),
	}

	if err := cfg.Logger.Validate(); err != nil {
		return cfg, fmt.Errorf("logger config validation: %w", err)
	}
	if err := cfg.Database.Validate(); err != nil {
		return cfg, fmt.Errorf("database config validation: %w", err)
	}
	if err := cfg.HTTP.Validate(); err != nil {
		return cfg, fmt.Errorf("http server config validation: %w", err)
	}

	return cfg, nil
}
