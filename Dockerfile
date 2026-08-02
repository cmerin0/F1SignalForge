# syntax=docker/dockerfile:1

# This Dockerfile builds a statically linked Linux binary for the F1SignalForge API

FROM golang:1.26-alpine AS build

WORKDIR /src

# Environment variables to build a statically linked binary for Linux.
ENV CGO_ENABLED=0
ENV GOOS=linux

# Copy dependency files first. Docker can reuse this layer when application
# code changes without a dependency change.
COPY go.mod go.sum vendor ./

# Build a statically linked Linux binary for a minimal runtime image.
COPY . .
RUN go build -trimpath -ldflags='-s -w' -mod=vendor -o /out/f1sf-api ./cmd/api
RUN go build -trimpath -ldflags='-s -w' -mod=vendor -o /out/f1sf-simulator ./cmd/simulator

# Runtime stage: contains only the compiled application binary.

# api stage: used to run the F1SignalForge API.
FROM gcr.io/distroless/static-debian12:nonroot AS api

WORKDIR /app

COPY --from=build /out/f1sf-api /app/f1signalforge-api

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/f1signalforge-api"]

# simulator stage: used to run the F1SignalForge simulator.
FROM gcr.io/distroless/static-debian12:nonroot AS simulator

WORKDIR /app

COPY --from=build /out/f1sf-simulator /app/f1signalforge-simulator

USER nonroot:nonroot

ENTRYPOINT ["/app/f1signalforge-simulator"]