// Package cleanup registers functions to run at program exit in LIFO order.
package cleanup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	// cleanupFuncs holds functions to be executed before program termination.
	// Functions are executed in reverse order of registration (LIFO).
	cleanupFuncs []func() error

	// cleanupMu protects concurrent access to cleanupFuncs.
	cleanupMu sync.Mutex
)

// Register registers a function to be called during program cleanup.
func Register(fn func() error) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	cleanupFuncs = append(cleanupFuncs, fn)
}

// Run executes all registered cleanup functions in reverse order.
func Run() error {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()

	n := len(cleanupFuncs)
	if n == 0 {
		return nil
	}

	for i := n - 1; i >= 0; i-- {
		if err := cleanupFuncs[i](); err != nil {
			return err
		}
	}

	return nil
}

// Listen sets up cleanup shutdown with callbacks.
func Listen(ctx context.Context, cancel context.CancelFunc, logger *slog.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		select {
		case sig := <-sigChan:
			fmt.Println("interrupted by user...")
			logger.Info("received interruption signal", "signal", sig)
			if err := Run(); err != nil {
				logger.Error("cleanup", "error", err)
			}
			cancel()
		case <-ctx.Done():
			logger.Debug("interrupt handler canceled by context")
		}
	}()
}
