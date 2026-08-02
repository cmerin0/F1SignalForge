// Package simulator generates controlled F1 telemetry scenarios.
package simulator

import (
	"errors"
	"math"
	"time"

	"github.com/cmerin0/F1SignalForge/internal/telemetry"
)

const samplesPerLap = 100

// NormalLapGenerator produces a repeatable approximation of normal on-track
// behavior: braking zones, acceleration, varying gears, and gradual fuel use.
type NormalLapGenerator struct {
	carNumber int
	sequence  uint64
}

// NewNormalLapGenerator creates a generator for one registered F1 car.
func NewNormalLapGenerator(carNumber int) (*NormalLapGenerator, error) {
	if carNumber < 1 || carNumber > 99 {
		return nil, errors.New("car number must be between 1 and 99")
	}

	return &NormalLapGenerator{
		carNumber: carNumber,
	}, nil
}

// Next returns the next telemetry measurement in the simulated lap.
//
// The generator is deterministic: the same sequence always produces the same
// measurement pattern. Deterministic scenarios are important because they make
// incidents reproducible during load, Kubernetes, and failure testing.
func (generator *NormalLapGenerator) Next(observedAt time.Time) telemetry.Event {
	// lapPosition is a normalized value between 0 and 1 that represents the
	// car's position in the lap. It is used to calculate speed, acceleration,
	// and steering angle.
	lapPosition := float64(generator.sequence%samplesPerLap) / samplesPerLap

	// acceleration ranges from 0 at a braking zone to 1 on a straight.
	acceleration := (math.Sin(2*math.Pi*lapPosition) + 1) / 2

	speedKPH := 95 + 245*acceleration
	gear := gearForSpeed(speedKPH)

	// Fuel decreases gradually but never becomes invalid during long demos.
	fuelLiters := math.Max(5, 110-float64(generator.sequence)*0.02)

	event := telemetry.Event{
		CarNumber:     generator.carNumber,
		ObservedAt:    observedAt.UTC(),
		SpeedKPH:      speedKPH,
		EngineRPM:     int(8000 + speedKPH*24),
		Gear:          gear,
		FuelLiters:    fuelLiters,
		BrakeTempC:    520 + 260*(1-acceleration),
		TyreTempC:     88 + 12*acceleration,
		SteeringAngle: 32 * math.Sin(4*math.Pi*lapPosition),
	}

	generator.sequence++

	return event
}

// gearForSpeed keeps gear selection consistent with speed. It intentionally
// avoids neutral and reverse because this scenario models normal race laps.
func gearForSpeed(speedKPH float64) int {
	switch {
	case speedKPH < 120:
		return 3
	case speedKPH < 165:
		return 4
	case speedKPH < 215:
		return 5
	case speedKPH < 270:
		return 6
	case speedKPH < 320:
		return 7
	default:
		return 8
	}
}
