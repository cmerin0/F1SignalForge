package telemetry

import "time"

const maxFutureClockSkew = 5 * time.Minute

// Event represents one reading emitted by a simulated Formula race car.
// It uses a car number because the car identifies itself externally; the
// PostgreSQL layer will later resolve it to the internal race_car ID.
type Event struct {
	CarNumber     int       `json:"car_number"`
	ObservedAt    time.Time `json:"observed_at"`
	SpeedKPH      float64   `json:"speed_kph"`
	EngineRPM     int       `json:"engine_rpm"`
	Gear          int       `json:"gear"`
	FuelLiters    float64   `json:"fuel_lt"`
	BrakeTempC    float64   `json:"brake_temp_c"`
	TyreTempC     float64   `json:"tyre_temp_c"`
	SteeringAngle float64   `json:"steering_angle"`
}

// FieldErrors maps an API field name to an explanation that a caller can use
// to correct its telemetry payload.
type FieldErrors map[string]string

// Validate checks telemetry invariants independently from HTTP and storage.
func (event Event) Validate(now time.Time) FieldErrors {
	errors := FieldErrors{}

	if event.CarNumber < 1 || event.CarNumber > 99 {
		errors["car_number"] = "must be between 1 and 99"
	}

	if event.ObservedAt.IsZero() {
		errors["observed_at"] = "is required"
	} else if event.ObservedAt.After(now.Add(maxFutureClockSkew)) {
		errors["observed_at"] = "must not be more than 5 minutes in the future"
	}

	if event.SpeedKPH < 0 || event.SpeedKPH > 450 {
		errors["speed_kph"] = "must be between 0 and 450"
	}

	if event.EngineRPM < 0 || event.EngineRPM > 20_000 {
		errors["engine_rpm"] = "must be between 0 and 20000"
	}

	if event.Gear < 0 || event.Gear > 8 {
		errors["gear"] = "must be between 0 and 8"
	}

	if event.FuelLiters < 0 || event.FuelLiters > 150 {
		errors["fuel_lt"] = "must be between 0 and 150"
	}

	if event.BrakeTempC < 0 || event.BrakeTempC > 1_500 {
		errors["brake_temp_c"] = "must be between 0 and 1500"
	}

	if event.TyreTempC < -20 || event.TyreTempC > 250 {
		errors["tyre_temp_c"] = "must be between -20 and 250"
	}

	if event.SteeringAngle < -90 || event.SteeringAngle > 90 {
		errors["steering_angle"] = "must be between -90 and 90"
	}

	if len(errors) == 0 {
		return nil
	}

	// Delayed events remain valid. A car can lose connectivity during a race
	// and replay buffered telemetry after a safety-car restart.
	return errors
}
