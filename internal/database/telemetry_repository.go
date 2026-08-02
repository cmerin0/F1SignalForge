package database

import (
	"context"
	"fmt"

	"github.com/cmerin0/F1SignalForge/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
)

const insertTelemetryEventQuery = `
	INSERT INTO telemetry_events (race_car_id, observed_at, speed_kph, engine_rpm, gear, fuel_lt, brake_temp_c, tyre_temp_c, steering_angle)
	SELECT id, $2, $3, $4, $5, $6, $7, $8, $9 FROM race_cars WHERE car_number = $1; `

// TelemetryRepository stores validated telemetry in PostgreSQL.
type TelemetryRepository struct {
	pool *pgxpool.Pool
}

// NewTelemetryRepository creates the PostgreSQL implementation of the
// telemetry repository. The pool lifecycle remains owned by main.go.
func NewTelemetryRepository(pool *pgxpool.Pool) *TelemetryRepository {
	return &TelemetryRepository{
		pool: pool,
	}
}

// Store writes one immutable telemetry measurement.
//
// The INSERT resolves car_number to race_car_id inside one SQL statement.
// This avoids an unnecessary lookup query and prevents persisting telemetry
// for a car that does not exist in race_cars.
func (repository *TelemetryRepository) Store(
	ctx context.Context,
	event telemetry.Event,
) error {
	commandTag, err := repository.pool.Exec(
		ctx,
		insertTelemetryEventQuery,
		event.CarNumber,
		event.ObservedAt,
		event.SpeedKPH,
		event.EngineRPM,
		event.Gear,
		event.FuelLiters,
		event.BrakeTempC,
		event.TyreTempC,
		event.SteeringAngle,
	)
	// If the INSERT fails, wrap the error with context and return it to the caller.
	if err != nil {
		return fmt.Errorf("insert telemetry event: %w", err)
	}
	// If the INSERT succeeds but no rows were affected, the car_number was not found
	if commandTag.RowsAffected() == 0 {
		return telemetry.ErrRaceCarNotFound
	}

	return nil
}
