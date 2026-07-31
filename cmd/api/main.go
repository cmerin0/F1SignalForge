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

const defaultListenAddress = ":8080"
const shutdownTimeout = 10 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Kubernetes sends SIGTERM before ending a pod. Converting it to a context
	// gives Fiber a clean, testable signal to begin graceful shutdown.
	gracefulContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	listenAddress := os.Getenv("LISTEN_ADDR")
	if listenAddress == "" {
		listenAddress = defaultListenAddress
	}

	app := httpapi.New(time.Now())

	logger.Info("starting Fiber API", "address", listenAddress)

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("starting Fiber API", "address", listenAddress)
		serverErrors <- app.Listen(listenAddress)
	}()

	select {
	case err := <-serverErrors:
		if err != nil {
			logger.Error("Fiber API stopped unexpectedly", "error", err)
			os.Exit(1)
		}

	case <-gracefulContext.Done():
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
