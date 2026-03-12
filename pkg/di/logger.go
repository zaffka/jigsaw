package di

import (
	"fmt"
	"os"

	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/zaffka/jigsaw/pkg/di/config"
	"github.com/zaffka/jigsaw/pkg/logger"
)

// RegisterLogger registers a zap.Logger in the DI container.
func RegisterLogger(injector do.Injector, cfg config.Logger) error {
	do.Provide(injector, func(i do.Injector) (*zap.Logger, error) {
		lopts := logger.Opts{
			Service:    cfg.ServiceName,
			Version:    cfg.Version,
			Level:      cfg.LogLevel,
			UseJSONFmt: cfg.LogFormat != cfg.LogFormatDefault,
		}

		l, err := logger.New(os.Stderr, lopts)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize logger with service %q version %q: %w",
				cfg.ServiceName, cfg.Version, err)
		}

		host, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("failed to get system hostname for logger: %w", err)
		}

		log := l.With(zap.String("host", host))
		log.Info("logger registered")

		RegisterCleanup(injector, func() {
			log.Info("shutting down logger")
			_ = log.Sync()
		})

		return log, nil
	})

	return nil
}
