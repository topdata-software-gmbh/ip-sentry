# ---- Stage 1: Build ----
FROM golang:1.21-alpine AS builder

WORKDIR /build
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ip-sentry .

# ---- Stage 2: Runtime ----
FROM alpine:latest

# ca-certificates for TLS
RUN apk add --no-cache ca-certificates

WORKDIR /app
RUN mkdir -p /app/configs /app/data/geoip

# Copy binary and scripts
COPY --from=builder /build/ip-sentry /app/ip-sentry
COPY scripts/ /app/scripts/

# --- FIX: Generate the preamble with the actual build time ---
# This runs DURING the docker build process and bakes the date into a file
RUN echo "ip-sentry starting - docker image built: $(date -u +'%Y-%m-%dT%H:%M:%SZ')" > /app/preamble.txt

# Create the entrypoint script to print that file first
RUN printf '#!/bin/sh\ncat /app/preamble.txt\nexec "$@"\n' > /app/entrypoint.sh && chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh", "/app/ip-sentry"]
CMD ["run", "--config", "/app/configs/config.yaml"]