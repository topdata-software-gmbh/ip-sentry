# ---- Stage 1: Build ----
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Cache dependencies before copying source
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ip-sentry .

# ---- Stage 2: Runtime ----
FROM alpine:latest

# ca-certificates for TLS (GeoIP updates, etc.)
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Create expected directory layout
RUN mkdir -p /app/configs /app/data/geoip

# Copy binary from builder
COPY --from=builder /build/ip-sentry /app/ip-sentry

# Copy helper scripts (e.g. GeoIP fetch)
COPY scripts/ /app/scripts/

ENTRYPOINT ["/app/ip-sentry"]
CMD ["run", "--config", "/app/configs/config.yaml"]
