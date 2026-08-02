package simulator

import (
	"testing"
	"time"
)

func TestNewNormalLapGeneratorRejectsInvalidCarNumber(t *testing.T) {
	_, err := NewNormalLapGenerator(0)

	if err == nil {
		t.Fatal("NewNormalLapGenerator() error = nil, want an error")
	}
}

func TestNormalLapGeneratorProducesValidTelemetry(t *testing.T) {
	generator, err := NewNormalLapGenerator(44)
	if err != nil {
		t.Fatalf("NewNormalLapGenerator() error = %v", err)
	}

	observedAt := time.Date(2026, time.August, 1, 22, 30, 0, 0, time.UTC)

	for sample := 0; sample < samplesPerLap; sample++ {
		event := generator.Next(observedAt)

		if event.CarNumber != 44 {
			t.Fatalf("car number = %d, want 44", event.CarNumber)
		}

		if fieldErrors := event.Validate(observedAt); fieldErrors != nil {
			t.Fatalf(
				"sample %d validation errors = %#v, want nil",
				sample,
				fieldErrors,
			)
		}
	}
}

func TestNormalLapGeneratorProducesChangingTelemetry(t *testing.T) {
	generator, err := NewNormalLapGenerator(44)
	if err != nil {
		t.Fatalf("NewNormalLapGenerator() error = %v", err)
	}

	observedAt := time.Date(2026, time.August, 1, 22, 30, 0, 0, time.UTC)

	firstEvent := generator.Next(observedAt)
	secondEvent := generator.Next(observedAt)

	if firstEvent.SpeedKPH == secondEvent.SpeedKPH {
		t.Fatal("speed did not change between simulator samples")
	}

	if firstEvent.SteeringAngle == secondEvent.SteeringAngle {
		t.Fatal("steering angle did not change between simulator samples")
	}
}
