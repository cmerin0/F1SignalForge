package httpapi

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

const maxRequestBodyBytes = 64 * 1024

type statusResponse struct {
	Status        string `json:"status"`
	UptimeSeconds int64  `json:"uptime_seconds,omitempty"`
}

// New creates the HTTP application and owns only HTTP routing and configuration.
// Domain behavior, persistence, and telemetry validation will be added separately.
func New(startedAt time.Time) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "F1SignalForge",

		// Telemetry events are compact JSON documents. A global limit protects
		// the API from unexpectedly large or malicious request bodies.
		BodyLimit: maxRequestBodyBytes,

		// These limits prevent slow or idle clients from holding connections
		// indefinitely. They will be revisited after load-test evidence exists.
		ReadTimeout:  5 * time.Second,
		IdleTimeout:  60 * time.Second,
		WriteTimeout: 15 * time.Second,
	})

	// GET /healthz is the liveness endpoint; it confirms the process is alive
	// and reports its uptime.
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(statusResponse{
			Status:        "ok",
			UptimeSeconds: int64(time.Since(startedAt).Seconds()),
		})
	})

	// GET /readyz is the readiness endpoint; it reports whether the service can
	// accept traffic. It currently has no external dependency to verify.
	app.Get("/readyz", func(c fiber.Ctx) error {
		// No external dependency exists yet. Once PostgreSQL is introduced,
		// this endpoint will verify database connectivity before returning ready.
		return c.Status(fiber.StatusOK).JSON(statusResponse{
			Status: "ready",
		})
	})

	return app
}
