package telemetry

import (
	"testing"
	"time"
)

// TestEventValidate verifies that Event.Validate accepts valid telemetry,
// reports every invalid field, and allows events received after a delay.
func TestEventValidate(t *testing.T) {
	// Use a fixed UTC time so timestamp checks produce the same result on every
	// machine and in every test run.
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)

	// Each table entry defines an input event and the field errors expected from
	// validating that event.
	tests := []struct {
		name          string
		event         Event
		desiredErrors FieldErrors
	}{
		{
			name:          "valid telemetry",
			event:         validEvent(now),
			desiredErrors: nil,
		},
		{
			name: "invalid telemetry values",
			event: Event{
				CarNumber:     100,
				ObservedAt:    now.Add(6 * time.Minute),
				SpeedKPH:      451,
				EngineRPM:     20_001,
				Gear:          9,
				FuelLiters:    151,
				BrakeTempC:    1_501,
				TyreTempC:     251,
				SteeringAngle: 91,
			},
			desiredErrors: FieldErrors{
				"car_number":     "must be between 1 and 99",
				"observed_at":    "must not be more than 5 minutes in the future",
				"speed_kph":      "must be between 0 and 450",
				"engine_rpm":     "must be between 0 and 20000",
				"gear":           "must be between 0 and 8",
				"fuel_lt":        "must be between 0 and 150",
				"brake_temp_c":   "must be between 0 and 1500",
				"tyre_temp_c":    "must be between -20 and 250",
				"steering_angle": "must be between -90 and 90",
			},
		},
		{
			name: "delayed telemetry remains valid",
			event: func() Event {
				// Simulate an event delivered one day after it was observed. Delayed
				// telemetry is valid because only excessive future timestamps fail.
				event := validEvent(now)
				event.ObservedAt = now.Add(-24 * time.Hour)
				return event
			}(),
			desiredErrors: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Validate the complete event and compare the number of errors first,
			// ensuring that no unexpected fields were accepted or omitted.
			obtainedErrors := test.event.Validate(now)
			if len(obtainedErrors) != len(test.desiredErrors) {
				t.Fatalf(
					"error count = %d, desired %d; errors = %#v",
					len(obtainedErrors),
					len(test.desiredErrors),
					obtainedErrors,
				)
			}

			// Compare each returned field message with the expected client-facing
			// explanation.
			for field, desiredMessage := range test.desiredErrors {
				if obtainedMessage := obtainedErrors[field]; obtainedMessage != desiredMessage {
					t.Errorf(
						"field %q error = %q, desired %q",
						field,
						obtainedMessage,
						desiredMessage,
					)
				}
			}
		})
	}
}

// validEvent creates a representative telemetry reading with values inside
// every range enforced by Event.Validate.
func validEvent(now time.Time) Event {
	// Keep the fixture realistic so successful validation also covers the normal
	// shape of a race-car telemetry event.
	return Event{
		CarNumber:     44,
		ObservedAt:    now.Add(-2 * time.Second),
		SpeedKPH:      310.5,
		EngineRPM:     11_800,
		Gear:          7,
		FuelLiters:    38.2,
		BrakeTempC:    740,
		TyreTempC:     96.4,
		SteeringAngle: -12.5,
	}
}
