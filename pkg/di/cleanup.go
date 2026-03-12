package di

import (
	"fmt"
	"sync"
	"time"

	"github.com/samber/do/v2"
)

var (
	cleanupFns []func()
	cleanupMu  sync.Mutex
)

// RegisterCleanup registers a cleanup function that will be called during shutdown.
// Cleanup functions are executed in reverse order of registration (LIFO).
func RegisterCleanup(injector do.Injector, fn func()) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	cleanupFns = append(cleanupFns, fn)
}

// ExecuteCleanup executes all registered cleanup functions in reverse order
// and resets the registry to prevent double-execution.
func ExecuteCleanup() {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()

	for i := len(cleanupFns) - 1; i >= 0; i-- {
		cleanupFns[i]()
	}

	cleanupFns = nil
}

// ShutdownWithTimeout performs graceful shutdown of all services with timeout.
func ShutdownWithTimeout(timeout time.Duration) error {
	done := make(chan struct{})

	go func() {
		ExecuteCleanup()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}
