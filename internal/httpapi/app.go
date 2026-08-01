package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/cmerin0/F1SignalForge/internal/telemetry"
)

// maxRequestBodyBytes limits telemetry payloads to 64 KiB to protect the API
// from unexpectedly large request bodies.
const maxRequestBodyBytes = 64 * 1024

// statusResponse is the JSON shape returned by the health and readiness
// endpoints.
type statusResponse struct {
	Status        string `json:"status"`
	UptimeSeconds int64  `json:"uptime_seconds,omitempty"`
}

// errorResponse is the JSON shape returned when a request cannot be accepted.
// Fields contains validation messages keyed by API field name when applicable.
type errorResponse struct {
	Error  string                `json:"error"`
	Fields telemetry.FieldErrors `json:"fields,omitempty"`
}

// telemetryValidationResponse confirms that an event passed validation. It
// does not claim that the event has been persisted.
type telemetryValidationResponse struct {
	Status string `json:"status"`
}

// New function is the HTTP application constructor and ensures that server
// limits and the health, readiness, and telemetry routes are configured.
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

	// POST /v1/telemetry accepts one JSON telemetry event for decoding and
	// validation before persistence is added.
	app.Post("/v1/telemetry", handleTelemetry)

	return app
}

// handleTelemetry function is the telemetry request handler and ensures that
// content type, JSON structure, and event values are validated in order.
func handleTelemetry(c fiber.Ctx) error {
	if !hasJSONContentType(c) {
		return c.Status(fiber.StatusUnsupportedMediaType).JSON(errorResponse{
			Error: "Content-Type must be application/json",
		})
	}

	event, err := decodeTelemetry(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Error: "invalid JSON body",
		})
	}

	if fieldErrors := event.Validate(time.Now().UTC()); fieldErrors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Error:  "validation failed",
			Fields: fieldErrors,
		})
	}

	// PostgreSQL persistence is intentionally deferred. This response confirms
	// validation only and does not falsely claim the event was stored.
	return c.Status(fiber.StatusOK).JSON(telemetryValidationResponse{
		Status: "validated",
	})
}

// hasJSONContentType function checks the request media type and ensures that
// JSON decoding is attempted only for application/json requests.
func hasJSONContentType(c fiber.Ctx) bool {
	mediaType, _, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
	return err == nil && mediaType == "application/json"
}

// decodeTelemetry function parses one telemetry event and ensures that unknown
// fields or additional JSON values are rejected.
func decodeTelemetry(c fiber.Ctx) (telemetry.Event, error) {
	var event telemetry.Event

	// Fiber owns the request-body buffer. We decode it immediately and never
	// retain that buffer beyond this handler's lifetime.
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&event); err != nil {
		return telemetry.Event{}, err
	}

	// A telemetry request must contain one JSON object only. Rejecting a second
	// value prevents ambiguous payloads such as concatenated event objects.
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return telemetry.Event{}, errors.New("unexpected extra JSON value")
	}

	return event, nil
}
