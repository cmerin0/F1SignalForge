package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestOperationalEndpoints(t *testing.T) {
	app := New(time.Now())

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   statusResponse
	}{
		{
			name:       "liveness endpoint",
			path:       "/healthz",
			wantStatus: http.StatusOK,
			wantBody: statusResponse{
				Status: "ok",
			},
		},
		{
			name:       "readiness endpoint",
			path:       "/readyz",
			wantStatus: http.StatusOK,
			wantBody: statusResponse{
				Status: "ready",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)

			response, err := app.Test(request, fiber.TestConfig{
				Timeout:       time.Second,
				FailOnTimeout: true,
			})
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}

			var body statusResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response body: %v", err)
			}

			if body.Status != test.wantBody.Status {
				t.Errorf("status body = %q, want %q", body.Status, test.wantBody.Status)
			}
		})
	}
}
