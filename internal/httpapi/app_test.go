package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cmerin0/F1SignalForge/internal/telemetry"
	"github.com/gofiber/fiber/v3"
)

// fakeTelemetryRepository lets HTTP tests verify route behavior without
// requiring a running PostgreSQL container.
type fakeTelemetryRepository struct {
	err         error
	storedEvent []telemetry.Event
}

func (repository *fakeTelemetryRepository) Store(
	_ context.Context,
	event telemetry.Event,
) error {
	repository.storedEvent = append(repository.storedEvent, event)
	return repository.err
}

func newTestApp(readinessError error, telemetryRepository telemetry.Repository) *fiber.App {
	return New(
		time.Now(),
		func(context.Context) error {
			return readinessError
		},
		telemetryRepository,
	)
}

func performRequest(t *testing.T, app *fiber.App, request *http.Request) *http.Response {
	t.Helper()

	response, err := app.Test(request, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	t.Cleanup(func() {
		response.Body.Close()
	})

	return response
}

func validTelemetryBody() string {
	observedAt := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)

	return fmt.Sprintf(`{
		"car_number": 44,
		"observed_at": %q,
		"speed_kph": 315.7,
		"engine_rpm": 11800,
		"gear": 7,
		"fuel_lt": 42.3,
		"brake_temp_c": 710.4,
		"tyre_temp_c": 96.8,
		"steering_angle": -4.2
	}`, observedAt)
}

func TestHealthzEndpoint(t *testing.T) {
	repository := &fakeTelemetryRepository{}
	app := newTestApp(nil, repository)

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := performRequest(t, app, request)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var body statusResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("status body = %q, want %q", body.Status, "ok")
	}
}

func TestReadyzEndpointReturnsOKWhenDatabaseIsReachable(t *testing.T) {
	repository := &fakeTelemetryRepository{}
	app := newTestApp(nil, repository)

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := performRequest(t, app, request)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

func TestReadyzEndpointReturnsServiceUnavailableWhenDatabaseIsDown(t *testing.T) {
	repository := &fakeTelemetryRepository{}
	app := newTestApp(errors.New("database unavailable"), repository)

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := performRequest(t, app, request)

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusServiceUnavailable,
		)
	}
}

func TestTelemetryEndpointStoresValidEvent(t *testing.T) {
	repository := &fakeTelemetryRepository{}
	app := newTestApp(nil, repository)

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/telemetry",
		strings.NewReader(validTelemetryBody()),
	)
	request.Header.Set(fiber.HeaderContentType, "application/json")

	response := performRequest(t, app, request)

	if response.StatusCode != http.StatusCreated {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusCreated,
		)
	}

	if len(repository.storedEvent) != 1 {
		t.Fatalf(
			"stored events = %d, want 1",
			len(repository.storedEvent),
		)
	}

	if repository.storedEvent[0].CarNumber != 44 {
		t.Fatalf(
			"stored car number = %d, want 44",
			repository.storedEvent[0].CarNumber,
		)
	}
}

func TestTelemetryEndpointRejectsInvalidEventWithoutStoringIt(t *testing.T) {
	repository := &fakeTelemetryRepository{}
	app := newTestApp(nil, repository)

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/telemetry",
		strings.NewReader(`{
			"car_number": 44,
			"observed_at": "2026-01-01T00:00:00Z",
			"speed_kph": 999,
			"engine_rpm": 11800,
			"gear": 7,
			"fuel_lt": 42.3,
			"brake_temp_c": 710.4,
			"tyre_temp_c": 96.8,
			"steering_angle": -4.2
		}`),
	)
	request.Header.Set(fiber.HeaderContentType, "application/json")

	response := performRequest(t, app, request)

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusBadRequest,
		)
	}

	if len(repository.storedEvent) != 0 {
		t.Fatalf(
			"stored events = %d, want 0",
			len(repository.storedEvent),
		)
	}
}

func TestTelemetryEndpointReturnsNotFoundForUnknownCar(t *testing.T) {
	repository := &fakeTelemetryRepository{
		err: telemetry.ErrRaceCarNotFound,
	}
	app := newTestApp(nil, repository)

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/telemetry",
		strings.NewReader(validTelemetryBody()),
	)
	request.Header.Set(fiber.HeaderContentType, "application/json")

	response := performRequest(t, app, request)

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusNotFound,
		)
	}
}
