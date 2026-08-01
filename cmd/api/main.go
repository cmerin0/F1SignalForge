package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cmerin0/F1SignalForge/internal/httpapi"
)

const (
	// Use port 8080 when LISTEN_ADDR is not provided by the environment.
	defaultListenAddress = ":8080"
	// Give active requests up to ten seconds to finish during shutdown.
	shutdownTimeout = 10 * time.Second
)

func main() {
	// Emit structured JSON logs so startup and shutdown events are easy to
	// search and consume in a container environment.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Kubernetes sends SIGTERM before ending a pod. Converting it to a context
	// gives Fiber a clean, testable signal to begin graceful shutdown.
	gracefulContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Read the bind address from the environment, falling back to the default
	// address used by the local development and container setup.
	listenAddress := os.Getenv("LISTEN_ADDR")
	if listenAddress == "" {
		listenAddress = defaultListenAddress
	}

	// Build the Fiber application and record its creation time for /healthz.
	app := httpapi.New(time.Now())

	logger.Info("starting Fiber API", "address", listenAddress)

	serverErrors := make(chan error, 1)

	// Run the blocking Fiber listener in the background so main can also wait
	// for an operating-system shutdown signal.
	go func() {
		logger.Info("starting Fiber API", "address", listenAddress)
		serverErrors <- app.Listen(listenAddress)
	}()

	// Continue until the server fails or Kubernetes/the operating system asks
	// the process to terminate.
	select {
	case err := <-serverErrors:
		// A listener error is unexpected unless it is caused by normal shutdown.
		if err != nil {
			logger.Error("Fiber API stopped unexpectedly", "error", err)
			os.Exit(1)
		}

	case <-gracefulContext.Done():
		// A shutdown signal starts the bounded graceful-shutdown sequence.
		logger.Info("shutdown signal received")

		shutdownContext, cancel := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer cancel()

		// Stop accepting new connections, then allow active requests to finish
		// until the Kubernetes-compatible shutdown deadline is reached.
		if err := app.ShutdownWithContext(shutdownContext); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}

		logger.Info("Fiber API stopped cleanly")
	}
}
