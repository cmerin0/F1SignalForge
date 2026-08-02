package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/cmerin0/F1SignalForge/internal/telemetry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// insertTelemetryEventQuery is the SQL statement used to persist telemetry events.
const insertTelemetryEventQuery = `
	INSERT INTO telemetry_events (race_car_id, observed_at, speed_kph, engine_rpm, gear, fuel_lt, brake_temp_c, tyre_temp_c, steering_angle)
	SELECT id, $2, $3, $4, $5, $6, $7, $8, $9 FROM race_cars WHERE car_number = $1; `

// selectLatestTelemetryEventQuery is the SQL statement used to retrieve the most recent telemetry event for a given car number.
const selectLatestTelemetryEventQuery = `
	SELECT race_cars.car_number, telemetry_events.observed_at, telemetry_events.speed_kph, telemetry_events.engine_rpm, telemetry_events.gear,
	telemetry_events.fuel_lt, telemetry_events.brake_temp_c, telemetry_events.tyre_temp_c, telemetry_events.steering_angle 
	FROM telemetry_events JOIN race_cars ON race_cars.id = telemetry_events.race_car_id 
	WHERE race_cars.car_number = $1 ORDER BY telemetry_events.observed_at DESC LIMIT 1;`

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
func (repository *TelemetryRepository) Store(ctx context.Context, event telemetry.Event) error {
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

// Latest returns the most recently observed telemetry for one F1 car.
//
// We order by observed_at rather than received_at because delayed telemetry
// must not replace a newer on-track measurement merely because it arrived later.
func (repository *TelemetryRepository) Latest(ctx context.Context, carNumber int) (telemetry.Event, error) {
	var event telemetry.Event

	err := repository.pool.QueryRow(
		ctx,
		selectLatestTelemetryEventQuery,
		carNumber,
	).Scan(
		&event.CarNumber,
		&event.ObservedAt,
		&event.SpeedKPH,
		&event.EngineRPM,
		&event.Gear,
		&event.FuelLiters,
		&event.BrakeTempC,
		&event.TyreTempC,
		&event.SteeringAngle,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return telemetry.Event{}, telemetry.ErrTelemetryNotFound
	}
	if err != nil {
		return telemetry.Event{}, fmt.Errorf("select latest telemetry event: %w", err)
	}

	return event, nil
}
