package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
