package di

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

// DatabasePool represents database connection pool interface.
type DatabasePool interface {
	Close()
	Ping(ctx context.Context) error
}

// Logger represents logger interface.
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	With(fields ...zap.Field) *zap.Logger
	Sync() error
}

// HTTPClient represents HTTP client interface.
type HTTPClient interface {
	Get(url string) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
}

// ServiceHealthChecker represents health check interface for services.
type ServiceHealthChecker interface {
	HealthCheck(ctx context.Context) error
	Name() string
}

// DatabasePoolChecker implements ServiceHealthChecker for database.
type DatabasePoolChecker struct {
	pool DatabasePool
}

func NewDatabasePoolChecker(pool DatabasePool) *DatabasePoolChecker {
	return &DatabasePoolChecker{pool: pool}
}

func (c *DatabasePoolChecker) HealthCheck(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *DatabasePoolChecker) Name() string {
	return "database"
}

// ServiceRegistry holds all registered services for health checks and cleanup.
type ServiceRegistry struct {
	healthCheckers []ServiceHealthChecker
	cleanupFuncs   []func()
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{}
}

func (r *ServiceRegistry) RegisterHealthChecker(checker ServiceHealthChecker) {
	r.healthCheckers = append(r.healthCheckers, checker)
}

func (r *ServiceRegistry) RegisterCleanup(fn func()) {
	r.cleanupFuncs = append(r.cleanupFuncs, fn)
}

// HealthCheck performs health checks for all registered services.
func (r *ServiceRegistry) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for _, checker := range r.healthCheckers {
		if err := checker.HealthCheck(ctx); err != nil {
			results[checker.Name()] = err
		}
	}
	return results
}

// Cleanup executes all registered cleanup functions in LIFO order.
func (r *ServiceRegistry) Cleanup() {
	for i := len(r.cleanupFuncs) - 1; i >= 0; i-- {
		r.cleanupFuncs[i]()
	}
}
