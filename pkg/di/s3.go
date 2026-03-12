package di

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/zaffka/jigsaw/pkg/di/config"
	"github.com/zaffka/jigsaw/pkg/s3"
)

// RegisterS3Client registers S3 bucket client in the DI container.
func RegisterS3Client(ctx context.Context, injector do.Injector, cfg config.S3) error {
	do.Provide(injector, func(i do.Injector) (*s3.BucketCli, error) {
		log := do.MustInvoke[*zap.Logger](i)

		client, err := s3.NewBucketCli(ctx, s3.Config{
			Endpoint:   cfg.Endpoint,
			AccessKey:  cfg.AccessKey,
			SecretKey:  cfg.SecretKey,
			BucketName: cfg.BucketName,
			Region:     cfg.Region,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create S3 client: %w", err)
		}

		log.Info("s3 registered and connected", zap.String("bucket", cfg.BucketName))

		return client, nil
	})

	return nil
}

// RegisterS3Config registers S3 configuration values in the DI container.
func RegisterS3Config(injector do.Injector, cfg config.S3) error {
	do.ProvideValue(injector, cfg)
	return nil
}
