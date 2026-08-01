-- +goose Up

-- race_cars is the source of truth for the F1 cars known to the application.
-- We keep a surrogate primary key internally, while car_number remains the
-- business identifier received from telemetry devices.
CREATE TABLE race_cars (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    car_number SMALLINT NOT NULL UNIQUE,
    team_name TEXT NOT NULL,
    driver_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- These constraints protect data even if another future service bypasses
    -- the Go API validation layer.
    CONSTRAINT race_cars_car_number_range
        CHECK (car_number BETWEEN 1 AND 99),
    CONSTRAINT race_cars_status_valid
        CHECK (status IN ('active', 'garage', 'retired', 'disqualified'))
);

-- telemetry_events stores the immutable measurements produced by each car.
-- received_at is deliberately separate from observed_at: it lets us later
-- measure ingestion delay and identify delayed or replayed telemetry.
CREATE TABLE telemetry_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    race_car_id BIGINT NOT NULL REFERENCES race_cars(id) ON DELETE RESTRICT,
    observed_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    speed_kph DOUBLE PRECISION NOT NULL,
    engine_rpm INTEGER NOT NULL,
    gear SMALLINT NOT NULL,
    fuel_lt DOUBLE PRECISION NOT NULL,
    brake_temp_c DOUBLE PRECISION NOT NULL,
    tyre_temp_c DOUBLE PRECISION NOT NULL,
    steering_angle DOUBLE PRECISION NOT NULL,

    -- Database constraints mirror the API's allowed physical ranges.
    CONSTRAINT telemetry_events_speed_range
        CHECK (speed_kph BETWEEN 0 AND 450),
    CONSTRAINT telemetry_events_engine_rpm_range
        CHECK (engine_rpm BETWEEN 0 AND 20000),
    CONSTRAINT telemetry_events_gear_range
        CHECK (gear BETWEEN 0 AND 8),
    CONSTRAINT telemetry_events_fuel_range
        CHECK (fuel_lt BETWEEN 0 AND 150),
    CONSTRAINT telemetry_events_brake_temp_range
        CHECK (brake_temp_c BETWEEN 0 AND 1500),
    CONSTRAINT telemetry_events_tyre_temp_range
        CHECK (tyre_temp_c BETWEEN -20 AND 250),
    CONSTRAINT telemetry_events_steering_angle_range
        CHECK (steering_angle BETWEEN -90 AND 90)
);

-- The main future query is: "show this car's latest telemetry".
CREATE INDEX telemetry_events_race_car_observed_at_idx
    ON telemetry_events (race_car_id, observed_at DESC);

-- +goose Down

-- Drop dependent data first because telemetry_events references race_cars.
DROP TABLE telemetry_events;
DROP TABLE race_cars;