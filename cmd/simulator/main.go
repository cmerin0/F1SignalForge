// Command simulator generates F1 telemetry and sends it to F1SignalForge.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cmerin0/F1SignalForge/internal/simulator"
	"github.com/cmerin0/F1SignalForge/internal/telemetry"
)

const (
	defaultAPIURL    = "http://localhost:8080"
	defaultCarNumber = 49
	defaultInterval  = time.Second
	requestTimeout   = 5 * time.Second
)

func main() {
	apiURL := flag.String(
		"api-url",
		defaultAPIURL,
		"base URL of the F1SignalForge API",
	)
	carNumber := flag.Int(
		"car-number",
		defaultCarNumber,
		"F1 car number to simulate",
	)
	interval := flag.Duration(
		"interval",
		defaultInterval,
		"delay between telemetry events",
	)
	duration := flag.Duration(
		"duration",
		0,
		"optional total simulator runtime; 0 runs until stopped",
	)
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if *interval <= 0 {
		logger.Error("interval must be greater than zero")
		os.Exit(1)
	}

	generator, err := simulator.NewNormalLapGenerator(*carNumber)
	if err != nil {
		logger.Error("invalid simulator configuration", "error", err)
		os.Exit(1)
	}

	runContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if *duration > 0 {
		var cancel context.CancelFunc

		runContext, cancel = context.WithTimeout(runContext, *duration)
		defer cancel()
	}

	endpoint := strings.TrimRight(*apiURL, "/") + "/v1/telemetry"

	logger.Info(
		"starting normal lap simulator",
		"car_number", *carNumber,
		"endpoint", endpoint,
		"interval", interval.String(),
		"duration", duration.String(),
	)

	client := &http.Client{
		Timeout: requestTimeout,
	}

	if err := run(
		runContext,
		logger,
		client,
		generator,
		endpoint,
		*interval,
	); err != nil {
		logger.Error("simulator stopped with an error", "error", err)
		os.Exit(1)
	}

	logger.Info("simulator stopped cleanly")
}

// run emits a telemetry event immediately, then repeats at the configured
// interval. Continuing after individual request failures is intentional:
// it lets us observe recovery when the API or a Kubernetes pod returns.
func run(
	ctx context.Context,
	logger *slog.Logger,
	client *http.Client,
	generator *simulator.NormalLapGenerator,
	endpoint string,
	interval time.Duration,
) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		event := generator.Next(time.Now().UTC())

		if err := sendEvent(ctx, client, endpoint, event); err != nil {
			logger.Error("could not send telemetry event", "error", err, "car_number", event.CarNumber)
		} else {
			logger.Info(
				"telemetry event accepted",
				"car_number", event.CarNumber,
				"speed_kph", math.Round(event.SpeedKPH*100)/100,
				"gear", event.Gear,
			)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// sendEvent sends one validated simulator event to the public API contract.
func sendEvent(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	event telemetry.Event,
) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode telemetry event: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create telemetry request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send telemetry request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 4*1024))
		if readErr != nil {
			return fmt.Errorf(
				"telemetry API returned status %d and its response could not be read: %w",
				response.StatusCode,
				readErr,
			)
		}

		return fmt.Errorf(
			"telemetry API returned status %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return nil
}
