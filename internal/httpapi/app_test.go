package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

// TestOperationalEndpoints function is the operational-route test and ensures
// that the health and readiness endpoints return the expected HTTP responses.
func TestOperationalEndpoints(t *testing.T) {
	// Build the application once so every endpoint case uses the same routing
	// configuration as the running service.
	app := New(time.Now())

	// Keep endpoint inputs and expected results together so the same assertions
	// can be reused for both operational routes.
	tests := []struct {
		name          string
		path          string
		desiredStatus int
		desiredBody   statusResponse
	}{
		{
			name:          "liveness endpoint",
			path:          "/healthz",
			desiredStatus: http.StatusOK,
			desiredBody: statusResponse{
				Status: "ok",
			},
		},
		{
			name:          "readiness endpoint",
			path:          "/readyz",
			desiredStatus: http.StatusOK,
			desiredBody: statusResponse{
				Status: "ready",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create an in-memory GET request without starting a real HTTP server.
			request := httptest.NewRequest(http.MethodGet, test.path, nil)

			// Send the request through Fiber and fail if the test server times out.
			response, err := app.Test(request, fiber.TestConfig{
				Timeout:       time.Second,
				FailOnTimeout: true,
			})
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			// The response body must be closed to release its resources.
			defer response.Body.Close()

			// Verify that the endpoint returns the HTTP status expected by probes.
			if response.StatusCode != test.desiredStatus {
				t.Fatalf("status = %d, desired %d", response.StatusCode, test.desiredStatus)
			}

			// Decode the JSON response so the payload can be checked by field.
			var body statusResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response body: %v", err)
			}

			// Verify that the response reports the correct operational state.
			if body.Status != test.desiredBody.Status {
				t.Errorf("status body = %q, desired %q", body.Status, test.desiredBody.Status)
			}
		})
	}
}

// TestTelemetryEndpoint function is the telemetry-route test and ensures that
// valid requests succeed while malformed or invalid requests are rejected.
func TestTelemetryEndpoint(t *testing.T) {
	// Build the application once so every case exercises the real route setup.
	app := New(time.Now())

	// Each case defines a request and the status and response text it must
	// produce.
	tests := []struct {
		name          string
		contentType   string
		body          string
		desiredStatus int
		desiredBody   string
	}{
		{
			name:        "accepts valid telemetry",
			contentType: "application/json",
			body: `{
				"car_number": 44,
				"observed_at": "2026-07-31T12:00:00Z",
				"speed_kph": 310.5,
				"engine_rpm": 11800,
				"gear": 7,
				"fuel_lt": 38.2,
				"brake_temp_c": 740,
				"tyre_temp_c": 96.4,
				"steering_angle": -12.5
			}`,
			desiredStatus: http.StatusOK,
			desiredBody:   `"status":"validated"`,
		},
		{
			name:          "rejects wrong content type",
			contentType:   "text/plain",
			body:          "not JSON",
			desiredStatus: http.StatusUnsupportedMediaType,
			desiredBody:   `"error":"Content-Type must be application/json"`,
		},
		{
			name:        "rejects invalid values",
			contentType: "application/json",
			body: `{
				"car_number": 100,
				"observed_at": "2026-07-31T12:00:00Z",
				"speed_kph": 310.5,
				"engine_rpm": 11800,
				"gear": 7,
				"fuel_lt": 38.2,
				"brake_temp_c": 740,
				"tyre_temp_c": 96.4,
				"steering_angle": -12.5
			}`,
			desiredStatus: http.StatusBadRequest,
			desiredBody:   `"error":"validation failed"`,
		},
		{
			name:        "rejects unknown JSON fields",
			contentType: "application/json",
			body: `{
				"car_number": 44,
				"observed_at": "2026-07-31T12:00:00Z",
				"unexpected_field": true
			}`,
			desiredStatus: http.StatusBadRequest,
			desiredBody:   `"error":"invalid JSON body"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create an in-memory POST request containing the test telemetry body.
			request := httptest.NewRequest(
				http.MethodPost,
				"/v1/telemetry",
				strings.NewReader(test.body),
			)
			// The content type determines whether the API will attempt JSON parsing.
			request.Header.Set("Content-Type", test.contentType)

			// Send the request through Fiber without starting a network listener.
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			// Close the response body after the subtest releases its resources.
			defer response.Body.Close()

			// Read the body once so error details can be included in assertion
			// failures and checked against the expected response text.
			responseBody, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read response body: %v", err)
			}

			// First verify the HTTP status returned by the endpoint.
			if response.StatusCode != test.desiredStatus {
				t.Fatalf(
					"status = %d, desired %d; body = %s",
					response.StatusCode,
					test.desiredStatus,
					responseBody,
				)
			}

			// Then verify that the response contains the expected API message.
			if !strings.Contains(string(responseBody), test.desiredBody) {
				t.Fatalf(
					"response body = %s, expected to contain %s",
					responseBody,
					test.desiredBody,
				)
			}
		})
	}
}
