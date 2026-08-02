# syntax=docker/dockerfile:1

# This Dockerfile builds a statically linked Linux binary for the F1SignalForge API

FROM golang:1.26-alpine AS build

WORKDIR /src

# Environment variables to build a statically linked binary for Linux.
ENV CGO_ENABLED=0
ENV GOOS=linux

# Copy dependency files first. Docker can reuse this layer when application
# code changes without a dependency change.
COPY go.mod go.sum ./
RUN go mod tidy && go mod vendor

# Build a statically linked Linux binary for a minimal runtime image.
COPY . .
RUN go build -trimpath -ldflags='-s -w' -o /out/f1sf-api ./cmd/api

# Runtime stage: contains only the compiled application binary.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy the statically linked binary from the build stage to the runtime stage.
COPY --from=build /out/f1sf-api /app/f1signalforge-api

EXPOSE 8080

# The application never needs root privileges at runtime.
USER nonroot:nonroot

ENTRYPOINT ["/app/f1signalforge-api"]