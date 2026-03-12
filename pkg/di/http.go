package di

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/zaffka/jigsaw/pkg/di/config"
)

// NewHTTPConfig returns default HTTP server configuration.
func NewHTTPConfig(addr string) config.HTTPServer {
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}
	return config.HTTPServer{
		Addr:            addr,
		WriteTimeout:    10 * time.Second,
		ReadTimeout:     10 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		MaxHeaderBytes:  1 << 20, // 1 MB
	}
}

// RegisterServeMux registers a serve mux in the DI container.
func RegisterServeMux(injector do.Injector) error {
	do.Provide(injector, func(i do.Injector) (*http.ServeMux, error) {
		return http.NewServeMux(), nil
	})

	return nil
}

// RegisterHTTPServer registers an HTTP server in the DI container.
func RegisterHTTPServer(_ context.Context, injector do.Injector, cfg config.HTTPServer) error {
	do.Provide(injector, func(i do.Injector) (*http.Server, error) {
		log := do.MustInvoke[*zap.Logger](i)
		serveMux := do.MustInvoke[*http.ServeMux](i)

		httpServer := &http.Server{
			Addr:           cfg.Addr,
			WriteTimeout:   cfg.WriteTimeout,
			ReadTimeout:    cfg.ReadTimeout,
			MaxHeaderBytes: cfg.MaxHeaderBytes,
			Handler:        serveMux,
		}

		log.Info("http server registered", zap.String("addr", cfg.Addr))

		return httpServer, nil
	})

	return nil
}

// RegisterHTTPServerConfig registers HTTP server configuration values in the DI container.
func RegisterHTTPServerConfig(injector do.Injector, cfg config.HTTPServer) error {
	do.ProvideValue(injector, cfg)
	return nil
}
