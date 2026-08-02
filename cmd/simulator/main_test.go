package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cmerin0/F1SignalForge/internal/telemetry"
)

func TestSendEventPostsTelemetryToAPI(t *testing.T) {
	expectedEvent := telemetry.Event{
		CarNumber:     49,
		ObservedAt:    time.Date(2026, time.August, 1, 22, 30, 0, 0, time.UTC),
		SpeedKPH:      315.7,
		EngineRPM:     11800,
		Gear:          7,
		FuelLiters:    42.3,
		BrakeTempC:    710.4,
		TyreTempC:     96.8,
		SteeringAngle: -4.2,
	}

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if request.Method != http.MethodPost {
				t.Fatalf("method = %s, desired %s", request.Method, http.MethodPost)
			}

			if request.URL.Path != "/v1/telemetry" {
				t.Fatalf("path = %s, desired /v1/telemetry", request.URL.Path)
			}

			if request.Header.Get("Content-Type") != "application/json" {
				t.Fatalf(
					"Content-Type = %q, desired application/json",
					request.Header.Get("Content-Type"),
				)
			}

			var receivedEvent telemetry.Event
			if err := json.NewDecoder(request.Body).Decode(&receivedEvent); err != nil {
				t.Fatalf("decode request body: %v", err)
			}

			if receivedEvent.CarNumber != expectedEvent.CarNumber {
				t.Fatalf(
					"car number = %d, desired %d",
					receivedEvent.CarNumber,
					expectedEvent.CarNumber,
				)
			}

			writer.WriteHeader(http.StatusCreated)
		},
	))
	defer server.Close()

	err := sendEvent(
		context.Background(),
		server.Client(),
		server.URL+"/v1/telemetry",
		expectedEvent,
	)
	if err != nil {
		t.Fatalf("sendEvent() error = %v", err)
	}
}

func TestSendEventReturnsErrorForUnexpectedAPIStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusServiceUnavailable)
			writer.Write([]byte(`{"error":"telemetry storage is unavailable"}`))
		},
	))
	defer server.Close()

	err := sendEvent(
		context.Background(),
		server.Client(),
		server.URL+"/v1/telemetry",
		telemetry.Event{},
	)
	if err == nil {
		t.Fatal("sendEvent() error = nil, desired an error")
	}

	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("error = %q, desired it to contain 503", err)
	}
}
